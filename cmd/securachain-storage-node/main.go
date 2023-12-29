package main

import (
	"bytes"
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/ipfs/boxo/path"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/internal/ipfs"
	"github.com/pierreleocadie/SecuraChain/internal/node"
	"github.com/pierreleocadie/SecuraChain/pkg/utils"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/p2p/host/autonat"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	/*
	* IPFS NODE
	 */
	// Spawn an IPFS node
	ipfsApi, nodeIpfs, err := ipfs.SpawnNode(ctx)
	if err != nil {
		log.Fatalf("Failed to spawn IPFS node: %s", err)
	}

	log.Printf("IPFS node spawned with PeerID: %s", nodeIpfs.Identity.String())

	/*
	* NODE LIBP2P
	 */
	// Initialize the storage node
	host := node.Initialize()
	defer host.Close()

	// Setup DHT discovery
	node.SetupDHTDiscovery(ctx, host, false)

	log.Printf("Storage node initialized with PeerID: %s", host.ID().String())

	/*
	* AUTONAT SERVICE
	 */
	_, err = autonat.New(host)
	if err != nil {
		log.Fatalf("Failed to create new autonat service: %s", err)
	}

	/*
	* PUBSUB
	 */
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		panic(err)
	}

	// Join the topic clientAnnouncementStringFlag
	clientAnnouncementTopic, err := ps.Join(config.ClientAnnouncementStringFlag)
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
			log.Println("Received ClientAnnouncement message from ", msg.GetFrom().String())
			log.Println("Client Announcement: ", string(msg.Data))
			clientAnnouncement, err := transaction.DeserializeClientAnnouncement(msg.Data)
			if err != nil {
				log.Printf("Failed to deserialize ClientAnnouncement: %s", err)
				continue
			}

			// Verify the ClientAnnouncement
			if !clientAnnouncement.VerifyTransaction(clientAnnouncement, clientAnnouncement.OwnerSignature, clientAnnouncement.OwnerAddress) {
				log.Printf("Failed to verify ClientAnnouncement: %s", err)
				continue
			}

			// Download the file
			fileImmutablePath := path.FromCid(clientAnnouncement.FileCid)
			log.Printf("Downloading file %s", fileImmutablePath)
			err = ipfs.GetFile(ctx, ipfsApi, fileImmutablePath)
			if err != nil {
				log.Printf("Failed to download file: %s", err)
				continue
			}

			// Verify that the file we downloaded is the same as the one announced
			home, err := os.UserHomeDir()
			if err != nil {
				log.Printf("Failed to get user home directory: %s", err)
				continue
			}

			downloadedFilePath := filepath.Join(home, ".IPFS_Downloads", clientAnnouncement.FileCid.String())
			checksum, err := utils.ComputeFileChecksum(downloadedFilePath)
			if err != nil {
				log.Printf("Failed to compute checksum of downloaded file: %s", err)
				continue
			}

			if bytes.Equal(checksum, clientAnnouncement.Checksum) {
				log.Printf("Downloaded file checksum does not match announced checksum")
				continue
			}

			fileInfo, err := os.Stat(downloadedFilePath)
			if err != nil {
				log.Printf("Failed to get file info: %s", err)
				continue
			}

			if uint64(fileInfo.Size()) != clientAnnouncement.FileSize {
				log.Printf("Downloaded file size does not match announced size")
				continue
			}

			log.Printf("File downloaded successfully")
		}
	}()

	/*
	* DISPLAY PEER CONNECTEDNESS CHANGES
	 */
	// Subscribe to EvtPeerConnectednessChanged events
	subNet, err := host.EventBus().Subscribe(new(event.EvtPeerConnectednessChanged))
	if err != nil {
		log.Println("Failed to subscribe to EvtPeerConnectednessChanged: ", err)
	}
	defer subNet.Close()

	// Handle connection events in a separate goroutine
	go func() {
		for e := range subNet.Out() {
			evt := e.(event.EvtPeerConnectednessChanged)
			if evt.Connectedness == network.Connected {
				log.Println("Peer connected:", evt.Peer)
			} else if evt.Connectedness == network.NotConnected {
				log.Println("Peer disconnected: ", evt.Peer)
			}
		}
	}()

	utils.WaitForTermSignal()
}
