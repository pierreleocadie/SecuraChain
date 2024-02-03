package main

import (
	// "bytes"
	"context"
	// "encoding/base64"
	"flag"
	// "os"
	// "path/filepath"

	// "github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/internal/discovery"
	// "github.com/pierreleocadie/SecuraChain/internal/ipfs"
	"github.com/pierreleocadie/SecuraChain/internal/node"
	"github.com/pierreleocadie/SecuraChain/pkg/utils"

	// "github.com/ipfs/boxo/path"
	// "github.com/ipfs/go-cid"
	ipfsLog "github.com/ipfs/go-log/v2"
	// pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
)

var (
	yamlConfigFilePath = flag.String("config", "", "Path to the yaml config file")
	generateKeys       = flag.Bool("genKeys", false, "Generate new ECDSA and AES keys to the paths specified in the config file")
)

func main() {
	log := ipfsLog.Logger("storage-node")
	err := ipfsLog.SetLogLevel("*", "DEBUG")
	if err != nil {
		log.Errorln("Error setting log level : ", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	flag.Parse()

	cfg := node.LoadConfig(yamlConfigFilePath, log)

	if *generateKeys {
		node.GenerateKeys(cfg, log)
	}

	// ecdsaKeyPair, _ := node.LoadKeys(cfg, log)

	// /*
	// * IPFS NODE
	//  */
	// ipfsAPI, nodeIpfs := node.InitializeIPFSNode(ctx, cfg, log)
	// dhtApi := ipfsAPI.Dht()

	// /*
	// * IPFS MEMORY MANAGEMENT
	//  */
	// storageMax, err := ipfs.ChangeStorageMax(nodeIpfs, cfg.MemorySpace)
	// if err != nil {
	// 	log.Warnf("Failed to change storage max: %s", err)
	// }
	// log.Debugf("Storage max changed: %v", storageMax)

	// freeMemorySpace, err := ipfs.FreeMemoryAvailable(ctx, nodeIpfs)
	// if err != nil {
	// 	log.Warnf("Failed to get free memory available: %s", err)
	// }
	// log.Debugf("Free memory available: %v", freeMemorySpace)

	// memoryUsedGB, err := ipfs.MemoryUsed(ctx, nodeIpfs)
	// if err != nil {
	// 	log.Warnf("Failed to get memory used: %s", err)
	// }
	// log.Debugf("Memory used: %v", memoryUsedGB)

	/*
	* NODE LIBP2P
	 */
	host := node.Initialize(*cfg)
	defer host.Close()
	log.Debugf("Storage node initialized with PeerID: %s", host.ID().String())

	/*
	* DHT DISCOVERY
	 */
	node.SetupDHTDiscovery(ctx, cfg, host, false)

	/*
	* RELAY SERVICE
	 */
	// Check if the node is behind NAT
	behindNAT := discovery.NATDiscovery(log)

	if !behindNAT {
		log.Debugln("Node is not behind NAT")
		// Start the relay service
		_, err = relay.New(host)
		if err != nil {
			log.Errorln("Error instantiating relay service : ", err)
		}
	}

	/*
	* PUBSUB
	 */
	// ps, err := pubsub.NewGossipSub(ctx, host)
	// if err != nil {
	// 	log.Panicf("Failed to create new pubsub: %s", err)
	// }

	// // When a file is fully downloaded without errors, into this chan
	// toTransactionChan := make(chan map[string]interface{})

	// // Join the topic StorageNodeResponseStringFlag
	// storageNodeResponseTopic, err := ps.Join(cfg.StorageNodeResponseStringFlag)
	// if err != nil {
	// 	log.Panicf("Failed to join StorageNodeResponseStringFlag topic: %s", err)
	// }

	// // Subscribe to StorageNodeResponseStringFlag topic
	// // subStorageNodeResponse, err := storageNodeResponseTopic.Subscribe()
	// // if err != nil {
	// // 	log.Panicf("Failed to subscribe to StorageNodeResponseStringFlag topic: %s", err)
	// // }

	// // Join the topic clientAnnouncementStringFlag
	// clientAnnouncementTopic, err := ps.Join(cfg.ClientAnnouncementStringFlag)
	// if err != nil {
	// 	log.Panicf("Failed to join clientAnnouncementStringFlag topic: %s", err)
	// }

	// // Subscribe to clientAnnouncementStringFlag topic
	// subClientAnnouncement, err := clientAnnouncementTopic.Subscribe()
	// if err != nil {
	// 	log.Panicf("Failed to subscribe to clientAnnouncementStringFlag topic: %s", err)
	// }

	// // Handle incoming ClientAnnouncement messages
	// go func() {
	// 	for {
	// 		msg, err := subClientAnnouncement.Next(ctx)
	// 		if err != nil {
	// 			panic(err)
	// 		}
	// 		log.Debugln("Received ClientAnnouncement message from ", msg.GetFrom().String())
	// 		log.Debugln("Client Announcement: ", string(msg.Data))
	// 		clientAnnouncement, err := transaction.DeserializeClientAnnouncement(msg.Data)
	// 		if err != nil {
	// 			log.Errorf("Failed to deserialize ClientAnnouncement: %s", err)
	// 			continue
	// 		}

	// 		// Verify the ClientAnnouncement
	// 		if !clientAnnouncement.VerifyTransaction(clientAnnouncement, clientAnnouncement.OwnerSignature, clientAnnouncement.OwnerAddress) {
	// 			log.Errorf("Failed to verify ClientAnnouncement: %s", err)
	// 			continue
	// 		}

	// 		// Check if we have enough space to store the file
	// 		freeMemorySpace, err := ipfs.FreeMemoryAvailable(ctx, nodeIpfs)
	// 		if err != nil {
	// 			log.Warnf("Failed to get free memory available: %s", err)
	// 		}
	// 		log.Debugf("Free memory available: %v", freeMemorySpace)

	// 		if clientAnnouncement.FileSize > freeMemorySpace {
	// 			log.Debugf("Not enough free memory available to store file")
	// 			continue
	// 		}

	// 		// Download the file
	// 		fileImmutablePath := path.FromCid(clientAnnouncement.FileCid)
	// 		providers, err := dhtApi.FindProviders(ctx, fileImmutablePath)
	// 		if err != nil {
	// 			log.Errorln("Error finding providers : ", err)
	// 			continue
	// 		}
	// 	outer:
	// 		for {
	// 			providers, err := dhtApi.FindProviders(ctx, fileImmutablePath)
	// 			if err != nil {
	// 				log.Errorln("Error finding providers : ", err)
	// 				continue
	// 			}
	// 			for provider := range providers {
	// 				log.Debugln("Found provider : ", provider.ID.String())
	// 				break outer
	// 			}
	// 			log.Debugf("Channel contains %d providers", len(providers))
	// 			if len(providers) >= 1 {
	// 				break
	// 			}
	// 		}
	// 		for provider := range providers {
	// 			ipfsAPI.Swarm().Connect(ctx, provider)
	// 		}
	// 		log.Debugf("Downloading file %s", fileImmutablePath)
	// 		err = ipfs.GetFile(ctx, cfg, ipfsAPI, fileImmutablePath)
	// 		if err != nil {
	// 			log.Errorf("Failed to download file: %s", err)
	// 			continue
	// 		}

	// 		// Verify that the file we downloaded is the same as the one announced
	// 		home, err := os.UserHomeDir()
	// 		if err != nil {
	// 			log.Errorf("Failed to get user home directory: %s", err)
	// 			continue
	// 		}

	// 		downloadedFilePath := filepath.Join(home, ".IPFS_Downloads", clientAnnouncement.FileCid.String())
	// 		checksum, err := utils.ComputeFileChecksum(downloadedFilePath)
	// 		if err != nil {
	// 			log.Errorf("Failed to compute checksum of downloaded file: %s", err)
	// 			continue
	// 		}

	// 		if !bytes.Equal(checksum, clientAnnouncement.Checksum) {
	// 			b64_1 := base64.URLEncoding.EncodeToString(checksum)
	// 			b64_2 := base64.URLEncoding.EncodeToString(clientAnnouncement.Checksum)
	// 			log.Errorf("Downloaded file checksum does not match announced checksum. Expected %v, got %v", b64_2, b64_1)
	// 			err = os.Remove(downloadedFilePath)
	// 			if err != nil {
	// 				log.Errorf("Failed to delete file: %s", err)
	// 			}
	// 			continue
	// 		}

	// 		fileInfo, err := os.Stat(downloadedFilePath)
	// 		if err != nil {
	// 			log.Errorf("Failed to get file info: %s", err)
	// 			continue
	// 		}

	// 		if uint64(fileInfo.Size()) != clientAnnouncement.FileSize {
	// 			log.Errorf("Downloaded file size does not match announced size")
	// 			err = os.Remove(downloadedFilePath)
	// 			if err != nil {
	// 				log.Errorf("Failed to delete file: %s", err)
	// 			}
	// 			continue
	// 		}

	// 		// Add the file to IPFS
	// 		fileImmutablePathCid, err := ipfs.AddFile(ctx, cfg, ipfsAPI, downloadedFilePath)
	// 		if err != nil {
	// 			log.Errorf("Failed to add file to IPFS: %s", err)
	// 			continue
	// 		}

	// 		// Pin the file
	// 		pinned, err := ipfs.PinFile(ctx, ipfsAPI, fileImmutablePathCid)
	// 		if err != nil {
	// 			log.Errorf("Failed to pin file: %s", err)
	// 		}
	// 		log.Debugf("File pinned: %s", pinned)

	// 		// From path.ImmutablePath to cid.Cid
	// 		fileRootCid := fileImmutablePathCid.RootCid()
	// 		if err != nil {
	// 			log.Errorf("Failed to parse file cid: %s", err)
	// 			continue
	// 		}

	// 		// Send to transaction channel
	// 		toTransactionChan <- map[string]interface{}{
	// 			"clientAnnouncement": clientAnnouncement,
	// 			"fileCid":            fileRootCid,
	// 		}

	// 		log.Infof("File %s downloaded successfully", fileImmutablePath)
	// 	}
	// }()

	// // Handle outgoing StorageNodeResponse messages
	// go func() {
	// 	for {
	// 		toTransaction := <-toTransactionChan
	// 		trx := transaction.NewAddFileTransaction(
	// 			toTransaction["clientAnnouncement"].(*transaction.ClientAnnouncement),
	// 			toTransaction["fileCid"].(cid.Cid), // Type assertion to convert to cid.Cid
	// 			false,
	// 			ecdsaKeyPair,
	// 			host.ID(),
	// 		)

	// 		// Send the transaction to the storage node response topic
	// 		transactionBytes, err := trx.Serialize()
	// 		if err != nil {
	// 			log.Errorf("Failed to serialize transaction: %s", err)
	// 			continue
	// 		}

	// 		err = storageNodeResponseTopic.Publish(ctx, transactionBytes)
	// 		if err != nil {
	// 			log.Errorf("Failed to publish transaction: %s", err)
	// 			continue
	// 		}

	// 		log.Infof("Transaction %s sent successfully", trx.TransactionID)
	// 	}
	// }()

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
