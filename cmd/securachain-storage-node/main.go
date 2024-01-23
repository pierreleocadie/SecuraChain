package main

import (
	"bytes"
	"context"
	"flag"
	"os"
	"path/filepath"

	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/internal/ipfs"
	"github.com/pierreleocadie/SecuraChain/internal/node"
	"github.com/pierreleocadie/SecuraChain/pkg/utils"

	"github.com/ipfs/boxo/path"
	ipfsLog "github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/network"
)

var yamlConfigFilePath = flag.String("config", "", "Path to the yaml config file")

func main() {
	log := ipfsLog.Logger("storage-node")
	err := ipfsLog.SetLogLevel("storage-node", "DEBUG")
	if err != nil {
		log.Errorln("Error setting log level : ", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	flag.Parse()

	// Load the config file
	if *yamlConfigFilePath == "" {
		log.Panicln("Please provide a path to the yaml config file")
	}

	cfg, err := config.LoadConfig(*yamlConfigFilePath)
	if err != nil {
		log.Panicln("Error loading config file : ", err)
	}

	/*
	* IPFS NODE
	 */
	// Spawn an IPFS node
	ipfsAPI, nodeIpfs, err := ipfs.SpawnNode(ctx)
	if err != nil {
		log.Panicf("Failed to spawn IPFS node: %s", err)
	}

	log.Debugf("IPFS node spawned with PeerID: %s", nodeIpfs.Identity.String())

	/*
	* MEMORY SHARE TO THE BLOCKCHAIN BY THE NODE
	 */

	storageMax, err := ipfs.ChangeStorageMax(nodeIpfs, cfg.MemorySpace)
	if err != nil {
		log.Warnf("Failed to change storage max: %s", err)
	}

	log.Debugf("Storage max set to %dGB", storageMax)

	freeMemoryGB, err := ipfs.FreeMemoryAvailable(ctx, nodeIpfs)
	if err != nil {
		log.Warnf("Failed to get free memory available: %s", err)
	}
	log.Debugf("Free memory available: %fGB", freeMemoryGB)

	memoryUsedGB, err := ipfs.MemoryUsed(ctx, nodeIpfs)
	if err != nil {
		log.Warnf("Failed to get memory used: %s", err)
	}
	log.Debugf("Memory used: %fGB", memoryUsedGB)

	/*
	* NODE LIBP2P
	 */
	// Initialize the storage node
	host := node.Initialize(*cfg)
	defer host.Close()

	// Setup DHT discovery
	node.SetupDHTDiscovery(ctx, host, false)

	log.Debugf("Storage node initialized with PeerID: %s", host.ID().String())

	/*
	* PUBSUB
	 */
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		panic(err)
	}

	// Join the topic clientAnnouncementStringFlag
	clientAnnouncementTopic, err := ps.Join(cfg.ClientAnnouncementStringFlag)
	if err != nil {
		panic(err)
	}

	// Subscribe to clientAnnouncementStringFlag topic
	subClientAnnouncement, err := clientAnnouncementTopic.Subscribe()
	if err != nil {
		panic(err)
	}

	// Handle incoming ClientAnnouncement messages
	go func() {
		for {
			msg, err := subClientAnnouncement.Next(ctx)
			if err != nil {
				panic(err)
			}
			log.Debugln("Received ClientAnnouncement message from ", msg.GetFrom().String())
			log.Debugln("Client Announcement: ", string(msg.Data))
			clientAnnouncement, err := transaction.DeserializeClientAnnouncement(msg.Data)
			if err != nil {
				log.Errorf("Failed to deserialize ClientAnnouncement: %s", err)
				continue
			}

			// Verify the ClientAnnouncement
			if !clientAnnouncement.VerifyTransaction(clientAnnouncement, clientAnnouncement.OwnerSignature, clientAnnouncement.OwnerAddress) {
				log.Errorf("Failed to verify ClientAnnouncement: %s", err)
				continue
			}

			// Download the file
			fileImmutablePath := path.FromCid(clientAnnouncement.FileCid)
			log.Debugf("Downloading file %s", fileImmutablePath)
			err = ipfs.GetFile(ctx, ipfsAPI, fileImmutablePath)
			if err != nil {
				log.Errorf("Failed to download file: %s", err)
				continue
			}

			// Verify that the file we downloaded is the same as the one announced
			home, err := os.UserHomeDir()
			if err != nil {
				log.Errorf("Failed to get user home directory: %s", err)
				continue
			}

			downloadedFilePath := filepath.Join(home, ".IPFS_Downloads", clientAnnouncement.FileCid.String())
			checksum, err := utils.ComputeFileChecksum(downloadedFilePath)
			if err != nil {
				log.Errorf("Failed to compute checksum of downloaded file: %s", err)
				continue
			}

			if bytes.Equal(checksum, clientAnnouncement.Checksum) {
				log.Errorf("Downloaded file checksum does not match announced checksum")
				continue
			}

			fileInfo, err := os.Stat(downloadedFilePath)
			if err != nil {
				log.Errorf("Failed to get file info: %s", err)
				continue
			}

			if uint64(fileInfo.Size()) != clientAnnouncement.FileSize {
				log.Errorf("Downloaded file size does not match announced size")
				continue
			}

			/*
				Check if the file exists and announce the response to the client
				TODO: Send the response to the client about the file download
			*/

			// Join the topic StorageNodeResponseStringFlag
			storageNodeResponseTopic, err := ps.Join(config.StorageNodeResponseStringFlag)
			if err != nil {
				panic(err)
			}

			if os.IsNotExist(err) {
				log.Debugf("File does not exist")
				responseDownloadToClient := "File does not exist"

				// Handle publishing StorageNodeResponse messages
				go func() {
					for {
						responseDownloadToClient := []byte(responseDownloadToClient)

						log.Debugln("Publishing StorageNodeResponse : ", responseDownloadToClient)

						err = storageNodeResponseTopic.Publish(ctx, responseDownloadToClient)
						if err != nil {
							log.Errorln("Error publishing StorageNodeResponse : ", err)
							continue
						}
					}
				}()
			} else {
				log.Debugf("File downloaded successfully")
				responseDownloadToClient := "File downloaded successfully"

				// Handle publishing StorageNodeResponse messages
				go func() {
					for {
						responseDownloadToClient := []byte(responseDownloadToClient)

						log.Debugln("Publishing StorageNodeResponse : ", responseDownloadToClient)

						err = storageNodeResponseTopic.Publish(ctx, responseDownloadToClient)
						if err != nil {
							log.Errorln("Error publishing StorageNodeResponse : ", err)
							continue
						}
					}
				}()
			}
		}
	}()

	/*
	* DISPLAY PEER CONNECTEDNESS CHANGES
	 */
	// Subscribe to EvtPeerConnectednessChanged events
	subNet, err := host.EventBus().Subscribe(new(event.EvtPeerConnectednessChanged))
	if err != nil {
		log.Errorln("Failed to subscribe to EvtPeerConnectednessChanged:", err)
	}
	defer subNet.Close()

	// Handle connection events in a separate goroutine
	go func() {
		for e := range subNet.Out() {
			evt := e.(event.EvtPeerConnectednessChanged)
			if evt.Connectedness == network.Connected {
				log.Debugln("Peer connected:", evt.Peer)
			} else if evt.Connectedness == network.NotConnected {
				log.Debugln("Peer disconnected:", evt.Peer)
			}
		}
	}()

	utils.WaitForTermSignal()
}
