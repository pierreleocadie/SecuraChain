package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"path/filepath"

	"github.com/pierreleocadie/SecuraChain/internal/blockchain"
	"github.com/pierreleocadie/SecuraChain/internal/blockchaindb"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/consensus"
	"github.com/pierreleocadie/SecuraChain/internal/ipfs"
	"github.com/pierreleocadie/SecuraChain/internal/node"
	blockregistry "github.com/pierreleocadie/SecuraChain/internal/registry/block_registry"
	fileregistry "github.com/pierreleocadie/SecuraChain/internal/registry/file_registry"
	"github.com/pierreleocadie/SecuraChain/pkg/utils"

	ipfsLog "github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/network"
)

var (
	yamlConfigFilePath          = flag.String("config", "", "Path to the yaml config file")
	blockReceieved              = make(chan block.Block, 15)
	transactionValidatorFactory = consensus.DefaultTransactionValidatorFactory{}
	genesisValidator            = consensus.DefaultGenesisBlockValidator{}
)

func main() {
	log := ipfsLog.Logger("full-node")
	if err := ipfsLog.SetLogLevel("full-node", "DEBUG"); err != nil {
		log.Errorln("Failed to set log level : ", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	flag.Parse()

	// Load the config file
	cfg := node.LoadConfig(yamlConfigFilePath, log)

	// Create the SecuraChain data directory
	securaChainDataDirectory, err := node.CreateSecuraChainDataDirectory(log, cfg)
	if err != nil {
		log.Panicf("Error creating the SecuraChain data directory : %s\n", err)
	}

	// Initialize the block validator
	blockValidator := consensus.NewDefaultBlockValidator(genesisValidator, transactionValidatorFactory)

	/*
	* IPFS NODE
	 */
	IPFSNode := ipfs.NewIPFSNode(ctx, log, cfg)

	/*
	* IPFS MEMORY MANAGEMENT
	 */
	storageMax, err := IPFSNode.ChangeStorageMax(cfg.MemorySpace)
	if err != nil {
		log.Warnf("Failed to change storage max: %s", err)
	}
	if storageMax {
		log.Infof("Storage max changed: %v", storageMax)
	}

	freeMemorySpace, err := IPFSNode.FreeMemoryAvailable()
	if err != nil {
		log.Warnf("Failed to get free memory available: %s", err)
	}
	log.Infof("Free memory available: %v", freeMemorySpace)

	memoryUsedGB, err := IPFSNode.MemoryUsed()
	if err != nil {
		log.Warnf("Failed to get memory used: %s", err)
	}
	log.Infof("Memory used: %v", memoryUsedGB)

	/*
	* NODE LIBP2P
	 */
	// Initialize the node
	host := node.Initialize(log, *cfg)
	defer host.Close()
	log.Debugf("Storage node initizalized with PeerID: %s", host.ID().String())

	/*
	* DHT DISCOVERY
	 */
	node.SetupDHTDiscovery(ctx, cfg, host, false, log)

	/*
	* PUBSUB TOPICS AND SUBSCRIPTIONS
	 */
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		panic(err)
	}

	// KeepRelayConnectionAlive
	keepRelayConnectionAliveTopic, err := ps.Join(cfg.KeepRelayConnectionAliveStringFlag)
	if err != nil {
		log.Warnf("Failed to join KeepRelayConnectionAlive topic: %s", err)
	}

	// Subscribe to KeepRelayConnectionAlive topic
	subKeepRelayConnectionAlive, err := keepRelayConnectionAliveTopic.Subscribe()
	if err != nil {
		log.Warnf("Failed to subscribe to KeepRelayConnectionAlive topic: %s", err)
	}

	// NetworkVisualisation
	networkVisualisationTopic, err := ps.Join(cfg.NetworkVisualisationStringFlag)
	if err != nil {
		log.Warnf("Failed to join NetworkVisualisation topic: %s", err)
	}

	// Join the topic BlockAnnouncementStringFlag
	blockAnnouncementTopic, err := ps.Join(cfg.BlockAnnouncementStringFlag)
	if err != nil {
		log.Panicf("Failed to join block announcement topic : %s\n", err)
	}
	subBlockAnnouncement, err := blockAnnouncementTopic.Subscribe()
	if err != nil {
		log.Panicf("Failed to subscribe to block announcement topic : %s\n", err)
	}

	// Join the topic to ask for the json file of the blockchain
	askingBlockchainTopic, err := ps.Join(cfg.AskingBlockchainStringFlag)
	if err != nil {
		log.Panicf("Failed to join AskingBlockchain topic : %s\n", err)
	}

	// Subscribe to the topic to ask for the json file of the blockchain
	subAskingBlockchain, err := askingBlockchainTopic.Subscribe()
	if err != nil {
		log.Panicf("Failed to subscribe to AskingBlockchain topic : %s\n", err)
	}

	// Join the topic to receive the json file of the blockchain
	receiveBlockchainTopic, err := ps.Join(cfg.ReceiveBlockchainStringFlag)
	if err != nil {
		log.Panicf("Failed to join ReceiveBlockchain topic : %s\n", err)
	}

	// Subscribe to the topic to receive the json file of the blockchain
	subReceiveBlockchain, err := receiveBlockchainTopic.Subscribe()
	if err != nil {
		log.Panicf("Failed to subscribe to ReceiveBlockchain topic : %s\n", err)
	}

	// Join the topic to ask for my files
	askMyFilesTopic, err := ps.Join(cfg.AskMyFilesStringFlag)
	if err != nil {
		log.Panicf("Failed to join AskMyFiles topic : %s\n", err)
	}

	// Subscribe to the topic to ask for my files
	subAskMyFiles, err := askMyFilesTopic.Subscribe()
	if err != nil {
		log.Panicf("Failed to subscribe to AskMyFiles topic : %s\n", err)
	}

	// Join the topic to send the files of the owner
	sendFilesTopic, err := ps.Join(cfg.SendFilesStringFlag)
	if err != nil {
		log.Panicf("Failed to join SendFiles topic : %s\n", err)
	}

	pubsubHub := &node.PubSubHub{
		ClientAnnouncementTopic:       nil,
		ClientAnnouncementSub:         nil,
		StorageNodeResponseTopic:      nil,
		StorageNodeResponseSub:        nil,
		KeepRelayConnectionAliveTopic: keepRelayConnectionAliveTopic,
		KeepRelayConnectionAliveSub:   subKeepRelayConnectionAlive,
		BlockAnnouncementTopic:        blockAnnouncementTopic,
		BlockAnnouncementSub:          subBlockAnnouncement,
		AskingBlockchainTopic:         askingBlockchainTopic,
		AskingBlockchainSub:           subAskingBlockchain,
		ReceiveBlockchainTopic:        receiveBlockchainTopic,
		ReceiveBlockchainSub:          subReceiveBlockchain,
		AskMyFilesTopic:               askMyFilesTopic,
		AskMyFilesSub:                 subAskMyFiles,
		SendMyFilesTopic:              sendFilesTopic,
		SendMyFilesSub:                nil,
		NetworkVisualisationTopic:     networkVisualisationTopic,
		NetworkVisualisationSub:       nil,
	}

	/*
	* BLOCK REGISTRY
	 */
	blockRegistryPath := filepath.Join(securaChainDataDirectory, cfg.BlockRegistryPath)
	blockRegistryManager := blockregistry.NewDefaultBlockRegistryManager(log, cfg, blockRegistryPath)
	blockRegistry, err := blockregistry.NewDefaultBlockRegistry(log, cfg, blockRegistryManager)
	if err != nil {
		log.Warnln("Error creating or opening a block registry : %s\n", err)
	}

	/*
	 * FILE REGISTRY
	 */
	fileRegistryPath := filepath.Join(securaChainDataDirectory, cfg.FileRegistryPath)
	fileRegistryManager := fileregistry.NewDefaultFileRegistryManager(log, cfg, fileRegistryPath)
	fileRegistry, err := fileregistry.NewDefaultFileRegistry(log, cfg, fileRegistryManager)
	if err != nil {
		log.Warnln("Error creating or opening a file registry : %s\n", err)
	}

	/*
	 * BLOCKCHAIN DATABASE
	 */
	// Create the blockchain directory
	blockchainDBPath := filepath.Join(securaChainDataDirectory, "blockchain")
	blockchainDB, err := blockchaindb.NewPebbleDB(log, blockchainDBPath, blockValidator)
	if err != nil {
		log.Debugln("Error creating or opening a database : %s\n", err)
	}

	/*
	 * BLOCKCHAIN
	 */
	chain := blockchain.NewBlockchain(log, cfg, ctx, IPFSNode, pubsubHub, blockValidator, blockchainDB, blockRegistry, fileRegistry, nil)

	/*
	* SERVICES
	 */

	node.PubsubKeepRelayConnectionAlive(ctx, pubsubHub, host, cfg, log)

	// Service 1 : Receiption of blocks
	go func() {
		for {
			msg, err := subBlockAnnouncement.Next(ctx)
			if err != nil {
				log.Errorln("error getting block announcement message: ", err)
				continue
			}
			log.Debugln("Received block announcement message from ", msg.GetFrom().String())
			log.Debugln("Received block : ", string(msg.Data))
			blockAnnounced, err := block.DeserializeBlock(msg.Data)
			if err != nil {
				log.Errorln("Error deserializing the block : ", err)
				continue
			}
			blockReceieved <- blockAnnounced
		}
	}()

	// Service 2: Handle the block received
	go func() {
		for {
			block := <-blockReceieved
			chain.HandleBlock(block)
		}
	}()

	// Service 3 : synchronization
	go func() {
		for {
			chain.SyncBlockchain()
		}
	}()

	// Service 4 : Post-synchronization
	go func() {
		for {
			chain.PostSync()
		}
	}()

	// Service 5 : Sending registry of the blockchain
	go func() {
		for {
			msg, err := subAskingBlockchain.Next(ctx)
			if err != nil {
				log.Debugln("Error getting message from the network : ", err)
				break
			}

			if msg.GetFrom().String() == host.ID().String() {
				continue
			}
			log.Debugln("Blockchain asked by a peer ", msg.GetFrom().String())

			// Send the registry of the blockchain
			blockRegistryBytes, err := json.Marshal(blockRegistry)
			if err != nil {
				log.Errorln("Error serializing the registry of the blockchain : ", err)
				continue
			}
			if err := receiveBlockchainTopic.Publish(ctx, blockRegistryBytes); err != nil {
				log.Errorln("Error publishing the registry of the blockchain : ", err)
				continue
			}
		}
	}()

	// Service 6 : Sending the files of the address given
	go func() {
		for {
			msg, err := subAskMyFiles.Next(ctx)
			if err != nil {
				log.Debugln("Error getting message from the network : ", err)
				break
			}

			if msg.GetFrom().String() == host.ID().String() {
				continue
			}
			log.Debugln("Files asked by a peer ", msg.GetFrom().String())

			// Send the files of the owner
			ownerAddressStr := fmt.Sprintf("%x", msg.Data)
			ownerFiles := fileRegistry.Get(ownerAddressStr)
			messageToSend := fileregistry.Message{OwnerPublicKey: ownerAddressStr, Registry: ownerFiles}
			ownerFilesBytes, err := json.Marshal(messageToSend)
			if err != nil {
				log.Errorln("Error serializing my files : ", err)
				continue
			}
			if err := sendFilesTopic.Publish(ctx, ownerFilesBytes); err != nil {
				log.Errorln("Error publishing indexing registry : ", err)
				continue
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
