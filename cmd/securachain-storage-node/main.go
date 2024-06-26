package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"

	"os"
	"path/filepath"

	"github.com/pierreleocadie/SecuraChain/internal/blockchain"
	"github.com/pierreleocadie/SecuraChain/internal/blockchaindb"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/consensus"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/internal/ipfs"
	"github.com/pierreleocadie/SecuraChain/internal/node"
	blockregistry "github.com/pierreleocadie/SecuraChain/internal/registry/block_registry"
	fileregistry "github.com/pierreleocadie/SecuraChain/internal/registry/file_registry"
	"github.com/pierreleocadie/SecuraChain/pkg/utils"

	"github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/path"
	"github.com/ipfs/go-cid"
	ipfsLog "github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/network"
)

var (
	yamlConfigFilePath          = flag.String("config", "", "Path to the yaml config file")
	generateKeys                = flag.Bool("genKeys", false, "Generate new ECDSA and AES keys to the paths specified in the config file")
	transactionValidatorFactory = consensus.DefaultTransactionValidatorFactory{}
	genesisValidator            = consensus.DefaultGenesisBlockValidator{}
	blockReceieved              = make(chan block.Block, 15)
	toTransactionChan           = make(chan map[string]interface{})
)

func main() { //nolint: funlen, gocyclo
	log := ipfsLog.Logger("storage-node")
	err := ipfsLog.SetLogLevel("storage-node", "DEBUG")
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

	ecdsaKeyPair, _ := node.LoadKeys(cfg, log)

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
	host := node.Initialize(log, *cfg)
	defer host.Close()
	log.Debugf("Storage node initialized with PeerID: %s", host.ID().String())

	/*
	* DHT DISCOVERY
	 */
	node.SetupDHTDiscovery(ctx, cfg, host, false, log)

	/*
	* PUBSUB TOPICS AND SUBSCRIPTIONS
	 */
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		log.Panicf("Failed to create new pubsub: %s", err)
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

	// Join the topic clientAnnouncementStringFlag
	clientAnnouncementTopic, err := ps.Join(cfg.ClientAnnouncementStringFlag)
	if err != nil {
		log.Panicf("Failed to join clientAnnouncementStringFlag topic: %s", err)
	}

	// Subscribe to clientAnnouncementStringFlag topic
	subClientAnnouncement, err := clientAnnouncementTopic.Subscribe()
	if err != nil {
		log.Panicf("Failed to subscribe to clientAnnouncementStringFlag topic: %s", err)
	}

	// Join the topic StorageNodeResponseStringFlag
	storageNodeResponseTopic, err := ps.Join(cfg.StorageNodeResponseStringFlag)
	if err != nil {
		log.Panicf("Failed to join StorageNodeResponseStringFlag topic: %s", err)
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
		ClientAnnouncementTopic:       clientAnnouncementTopic,
		ClientAnnouncementSub:         subClientAnnouncement,
		StorageNodeResponseTopic:      storageNodeResponseTopic,
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

	// Handle incoming ClientAnnouncement messages
	go func() {
		for {
			msg, err := subClientAnnouncement.Next(ctx)
			if err != nil {
				log.Errorf("Failed to get next message from clientAnnouncementStringFlag topic: %s", err)
			}
			log.Debugln("Received ClientAnnouncement message from ", msg.GetFrom().String())
			log.Debugln("Client Announcement: ", string(msg.Data))
			trxClientAnnouncement, err := transaction.DeserializeTransaction(msg.Data)
			if err != nil {
				log.Errorf("Failed to deserialize ClientAnnouncement: %s", err)
				continue
			}

			if _, ok := trxClientAnnouncement.(*transaction.ClientAnnouncement); !ok {
				log.Errorf("Received transaction is not a ClientAnnouncement")
				continue
			}

			// Verify the ClientAnnouncement
			if err := trxClientAnnouncement.Verify(trxClientAnnouncement, trxClientAnnouncement.(transaction.ClientAnnouncement).OwnerSignature, trxClientAnnouncement.(transaction.ClientAnnouncement).OwnerAddress); err != nil {
				log.Errorf("Failed to verify ClientAnnouncement: %s", err)
				continue
			}

			cA, ok := trxClientAnnouncement.(*transaction.ClientAnnouncement)
			if !ok {
				log.Errorf("Failed to convert transaction to ClientAnnouncement")
				continue
			}
			clientAnnouncement := *cA

			// Check if the file size is within the limit
			if clientAnnouncement.FileSize > cfg.FileSizeLimit {
				log.Debugf("File size exceeds the limit")
				continue
			}

			// Check if we have enough space to store the file
			freeMemorySpace, err := IPFSNode.FreeMemoryAvailable()
			if err != nil {
				log.Warnf("Failed to get free memory available: %s", err)
			}
			log.Debugf("Free memory available: %v", freeMemorySpace)

			if clientAnnouncement.FileSize > freeMemorySpace {
				log.Debugf("Not enough free memory available to store file")
				continue
			}

			// Download the file
			fileImmutablePath := path.FromCid(clientAnnouncement.FileCid)

			err = IPFSNode.API.Swarm().Connect(ctx, clientAnnouncement.IPFSClientNodeAddrInfo)
			if err != nil {
				log.Errorf("Failed to connect to provider: %s", err)
				continue
			}

			log.Debugf("Downloading file %s", fileImmutablePath)
			rootNodeFile, err := IPFSNode.GetFile(fileImmutablePath)
			if err != nil {
				log.Errorf("Failed to download file: %s", err)
				continue
			}

			// Move the file to the downloads storage
			home, err := os.UserHomeDir()
			if err != nil {
				log.Errorf("Failed to get user home directory: %s", err)
				continue
			}

			downloadsStoragePath := filepath.Join(home, ".IPFS_Downloads/")

			// Ensure the output directory exists or create it.
			if err := os.MkdirAll(downloadsStoragePath, os.FileMode(cfg.FileRights)); err != nil {
				log.Errorf("Error creating output directory: %v", err)
			}

			downloadedFilePath := filepath.Join(downloadsStoragePath, filepath.Base(fileImmutablePath.String()))

			log.Debugln("Writing file to %s\n", downloadedFilePath)

			if err := files.WriteTo(rootNodeFile, downloadedFilePath); err != nil {
				log.Errorf("Failed to write out the fetched CID: %v", err)
				continue
			}

			// Verify that the file we downloaded is the same as the one announced
			downloadedFilePath = filepath.Join(home, ".IPFS_Downloads", clientAnnouncement.FileCid.String())
			checksum, err := utils.ComputeFileChecksum(downloadedFilePath)
			if err != nil {
				log.Errorf("Failed to compute checksum of downloaded file: %s", err)
				continue
			}

			if !bytes.Equal(checksum, clientAnnouncement.Checksum) {
				b64_1 := base64.URLEncoding.EncodeToString(checksum)
				b64_2 := base64.URLEncoding.EncodeToString(clientAnnouncement.Checksum)
				log.Errorf("Downloaded file checksum does not match announced checksum. Expected %v, got %v", b64_2, b64_1)
				err = os.Remove(downloadedFilePath)
				if err != nil {
					log.Errorf("Failed to delete file: %s", err)
				}
				continue
			}

			fileInfo, err := os.Stat(downloadedFilePath)
			if err != nil {
				log.Errorf("Failed to get file info: %s", err)
				continue
			}

			if uint64(fileInfo.Size()) != clientAnnouncement.FileSize {
				log.Errorf("Downloaded file size does not match announced size")
				err = os.Remove(downloadedFilePath)
				if err != nil {
					log.Errorf("Failed to delete file: %s", err)
				}
				continue
			}

			// Add the file to IPFS
			fileImmutablePathCid, err := IPFSNode.AddFile(downloadedFilePath)
			if err != nil {
				log.Errorf("Failed to add file to IPFS: %s", err)
				continue
			}

			// Move the file to the local storage
			localStoragePath := filepath.Join(home, ".IPFS_Local_Storage/")
			if err := os.MkdirAll(localStoragePath, os.FileMode(cfg.FileRights)); err != nil {
				log.Error("error creating output directory : %v", err)
			}

			outputFilePath := filepath.Join(localStoragePath, filepath.Base(downloadedFilePath))

			if err := ipfs.MoveFile(downloadedFilePath, outputFilePath); err != nil {
				log.Errorf("Failed to move file to local storage: %s", err)
				continue
			}

			// restore file permissions
			if err := os.Chmod(outputFilePath, os.FileMode(cfg.FileRights)); err != nil {
				log.Error("error restoring file permissions: %v", err)
			}

			// Pin the file
			if err := IPFSNode.PinFile(fileImmutablePathCid); err != nil {
				log.Errorf("Failed to pin file: %s", err)
				continue
			}

			// From path.ImmutablePath to cid.Cid
			fileRootCid := fileImmutablePathCid.RootCid()
			if err != nil {
				log.Errorf("Failed to parse file cid: %s", err)
				continue
			}

			// Send to transaction channel
			toTransactionChan <- map[string]interface{}{
				"clientAnnouncement": clientAnnouncement,
				"fileCid":            fileRootCid,
			}

			log.Infof("File %s downloaded successfully", fileImmutablePath)
		}
	}()

	// Handle outgoing StorageNodeResponse messages
	go func() {
		for {
			toTransaction := <-toTransactionChan
			trx := transaction.NewAddFileTransaction(
				toTransaction["clientAnnouncement"].(transaction.ClientAnnouncement),
				toTransaction["fileCid"].(cid.Cid), // Type assertion to convert to cid.Cid
				ecdsaKeyPair,
				host.ID(),
				IPFSNode.Node.Peerstore.PeerInfo(IPFSNode.Node.Identity),
			)

			// Send the transaction to the storage node response topic
			transactionBytes, err := transaction.SerializeTransaction(trx)
			if err != nil {
				log.Errorf("Failed to serialize transaction: %s", err)
				continue
			}

			err = storageNodeResponseTopic.Publish(ctx, transactionBytes)
			if err != nil {
				log.Errorf("Failed to publish transaction: %s", err)
				continue
			}

			log.Infof("Transaction %s sent successfully", trx.TransactionID)
		}
	}()

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
