package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"slices"
	"time"

	"os"
	"path/filepath"

	"github.com/pierreleocadie/SecuraChain/internal/blockchaindb"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/consensus"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/internal/fullnode"
	"github.com/pierreleocadie/SecuraChain/internal/ipfs"
	"github.com/pierreleocadie/SecuraChain/internal/node"
	"github.com/pierreleocadie/SecuraChain/internal/visualisation"
	"github.com/pierreleocadie/SecuraChain/pkg/utils"

	"github.com/ipfs/boxo/path"
	"github.com/ipfs/go-cid"
	ipfsLog "github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/network"
)

var (
	yamlConfigFilePath     = flag.String("config", "", "Path to the yaml config file")
	generateKeys           = flag.Bool("genKeys", false, "Generate new ECDSA and AES keys to the paths specified in the config file")
	requiresSync           = false
	requiresPostSync       = false
	blockProcessingEnabled = true
	pendingBlocks          = []*block.Block{}
	blockReceieved         = make(chan *block.Block, 15)
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

	/*
	* IPFS NODE
	 */
	ipfsAPI, nodeIpfs := node.InitializeIPFSNode(ctx, cfg, log)

	/*
	* IPFS MEMORY MANAGEMENT
	 */
	storageMax, err := ipfs.ChangeStorageMax(nodeIpfs, cfg.MemorySpace)
	if err != nil {
		log.Warnf("Failed to change storage max: %s", err)
	}
	if storageMax {
		log.Infof("Storage max changed: %v", storageMax)
	}

	freeMemorySpace, err := ipfs.FreeMemoryAvailable(ctx, nodeIpfs)
	if err != nil {
		log.Warnf("Failed to get free memory available: %s", err)
	}
	log.Infof("Free memory available: %v", freeMemorySpace)

	memoryUsedGB, err := ipfs.MemoryUsed(ctx, nodeIpfs)
	if err != nil {
		log.Warnf("Failed to get memory used: %s", err)
	}
	log.Infof("Memory used: %v", memoryUsedGB)

	/*
	* BLOCKCHAIN DATABASE
	 */
	// Create the blockchain directory
	blockchainPath := filepath.Join(securaChainDataDirectory, "blockchain")
	blockchain, err := blockchaindb.NewBlockchainDB(log, blockchainPath)
	if err != nil {
		log.Debugln("Error creating or opening a database : %s\n", err)
	}

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
	* PUBSUB
	 */
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		log.Panicf("Failed to create new pubsub: %s", err)
	}

	// When a file is fully downloaded without errors, into this chan
	toTransactionChan := make(chan map[string]interface{})

	// KeepRelayConnectionAlive
	node.PubsubKeepRelayConnectionAlive(ctx, ps, host, cfg, log)

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

	// Subscribe to StorageNodeResponseStringFlag topic
	// subStorageNodeResponse, err := storageNodeResponseTopic.Subscribe()
	// if err != nil {
	// 	log.Panicf("Failed to subscribe to StorageNodeResponseStringFlag topic: %s", err)
	// }

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

	// Handle incoming ClientAnnouncement messages
	go func() {
		for {
			msg, err := subClientAnnouncement.Next(ctx)
			if err != nil {
				log.Errorf("Failed to get next message from clientAnnouncementStringFlag topic: %s", err)
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

			// Check if the file size is within the limit
			if clientAnnouncement.FileSize > cfg.FileSizeLimit {
				log.Debugf("File size exceeds the limit")
				continue
			}

			// Check if we have enough space to store the file
			freeMemorySpace, err := ipfs.FreeMemoryAvailable(ctx, nodeIpfs)
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

			err = ipfsAPI.Swarm().Connect(ctx, clientAnnouncement.IPFSClientNodeAddrInfo)
			if err != nil {
				log.Errorf("Failed to connect to provider: %s", err)
				continue
			}

			log.Debugf("Downloading file %s", fileImmutablePath)
			err = ipfs.GetFile(ctx, cfg, ipfsAPI, fileImmutablePath)
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
			fileImmutablePathCid, err := ipfs.AddFile(ctx, cfg, ipfsAPI, downloadedFilePath)
			if err != nil {
				log.Errorf("Failed to add file to IPFS: %s", err)
				continue
			}

			// Pin the file
			pinned, err := ipfs.PinFile(ctx, ipfsAPI, fileImmutablePathCid)
			if err != nil {
				log.Errorf("Failed to pin file: %s", err)
			}
			log.Debugf("File pinned: %s", pinned)

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
				toTransaction["clientAnnouncement"].(*transaction.ClientAnnouncement),
				toTransaction["fileCid"].(cid.Cid), // Type assertion to convert to cid.Cid
				false,
				ecdsaKeyPair,
				host.ID(),
				nodeIpfs.Peerstore.PeerInfo(nodeIpfs.Identity),
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
			blockAnnounced, err := fullnode.ReceiveBlock(log, ctx, subBlockAnnouncement)
			if err != nil {
				log.Debugln("Error waiting for the next block : %s\n", err)
				continue
			}

			// If the node is synchronizing with the network
			if requiresSync {
				pendingBlocks = append(pendingBlocks, blockAnnounced)
			} else {
				blockReceieved <- blockAnnounced
			}
		}
	}()

	// Service 2: Handle the block received
	go func() {
		for {
			if !blockProcessingEnabled {
				continue
			}
			select {
			case <-ctx.Done():
				return
			case bReceive := <-blockReceieved:
				log.Debugln("Normal block processing")
				log.Debugln("State : block processing enabled ", blockProcessingEnabled)

				// 1 . Validation of the block
				if block.IsGenesisBlock(bReceive) {
					log.Debugln("Genesis block")
					if !consensus.ValidateBlock(bReceive, nil) {
						log.Debugln("Genesis block is invalid")
						continue
					}
					log.Debugln("Genesis block is valid")
				} else {
					isPrevBlockStored, err := fullnode.PrevBlockStored(log, bReceive, blockchain)
					if err != nil {
						log.Debugln("error checking if previous block is stored : %s", err)
					}

					if !isPrevBlockStored {
						pendingBlocks = append(pendingBlocks, bReceive)
						blockProcessingEnabled = false
						requiresSync = true // This will call the process of synchronizing the blockchain with the network
						continue
					}

					prevBlock, err := blockchain.GetBlock(log, bReceive.PrevBlock)
					if err != nil {
						log.Debugln("Error getting the previous block : %s\n", err)
					}

					if !consensus.ValidateBlock(bReceive, prevBlock) {
						log.Debugln("Block is invalid")
						continue
					}
				}

				// 2 . Add the block to the blockchain
				if err := blockchaindb.AddBlockToBlockchain(log, bReceive, blockchain); err != nil {
					log.Debugln("Error adding the block to the blockchain : %s\n", err)
					continue
				}

				// 3 . Verify the integrity of the blockchain
				if !blockchain.VerifyIntegrity(log) {
					blockProcessingEnabled = false
					requiresSync = true // This will call the process of synchronizing the blockchain with the network
					continue
				}

				// 4 . Add the block transaction to the registry
				if !fullnode.AddBlockTransactionToRegistry(log, cfg, bReceive) {
					log.Debugln("Error adding the block transactions to the registry")
				}

				// 5 . Send the block to IPFS
				if !ipfs.PublishBlock(log, ctx, cfg, nodeIpfs, ipfsAPI, bReceive) {
					log.Debugln("Error publishing the block to IPFS")
				}
			}
		}
	}()

	// Service 3 : Syncronization
	go func() {
		// create a list of blacklisted nodes
		blackListNode := []string{}
		for {
			if !requiresSync {
				continue
			}
			log.Debugln("Synchronizing the blockchain with the network")
			log.Debugln("State : requires sync ", requiresSync)

			//1 . Ask for a registry of the blockchain
			registryBytes, senderID, err := fullnode.AskForBlockchainRegistry(log, ctx, askingBlockchainTopic, subReceiveBlockchain)
			if err != nil {
				log.Debugln("Error asking the blockchain registry : %s\n", err)
				continue
			}

			// 1.1 Check if the sender is blacklisted
			if slices.Contains(blackListNode, senderID) {
				log.Debugln("Node blacklisted")
				continue
			}

			log.Debugln("Node not blacklisted")

			// 1.2 black list the sender
			blackListNode = append(blackListNode, senderID)
			log.Debugln("Node added to the black list")

			// 2 . Dowlnoad the missing blocks
			downloaded, listOfMissingBlocks, err := fullnode.DownloadMissingBlocks(log, ctx, ipfsAPI, registryBytes, blockchain)
			if err != nil {
				log.Debugln("Error downloading missing blocks : %s\n", err)
			}

			if !downloaded {
				log.Debugln("Blocks not downloaded")
				continue
			}

			// 3 . Valid the downloaded blocks
			for _, b := range listOfMissingBlocks {
				if block.IsGenesisBlock(b) {
					if !consensus.ValidateBlock(b, nil) {
						log.Debugln("Genesis block is invalid")
						continue
					}
					log.Debugln("Genesis block is valid")
				} else {
					prevBlock, err := blockchain.GetBlock(log, b.PrevBlock)
					if err != nil {
						log.Debugln("Error getting the previous block : %s\n", err)
					}
					if !consensus.ValidateBlock(b, prevBlock) {
						log.Debugln("Block is invalid")
						continue
					}
					log.Debugln(b.Height, " is valid")
				}

				// 4 . Add the block to the blockchain
				if err := blockchaindb.AddBlockToBlockchain(log, b, blockchain); err != nil {
					log.Debugln("Error adding the block to the blockchain : %s\n", err)
					continue
				}

				// 5 . Verify the integrity of the blockchain
				if !blockchain.VerifyIntegrity(log) {
					log.Debugln("Blockchain is not verified")
					continue
				}

				// 6 . Add the block transaction to the registry
				if !fullnode.AddBlockTransactionToRegistry(log, cfg, b) {
					log.Debugln("Error adding the block transactions to the registry")
				}

				// 7 . Send the block to IPFS
				if !ipfs.PublishBlock(log, ctx, cfg, nodeIpfs, ipfsAPI, b) {
					log.Debugln("Error publishing the block to IPFS")
				}
			}

			// 7 . Clear the pending blocks
			blackListNode = []string{}

			// 8 . Change the state of the node
			requiresSync = false
			requiresPostSync = true
			log.Debugln("Blockchain synchronized with the network")
		}
	}()

	// Service 4 : Post-syncronization
	go func() {
		for {
			if !requiresPostSync {
				continue
			}

			log.Debugln("Post-syncronization of the blockchain with the network")
			log.Debugln("State : Post-syncronization ", requiresPostSync)

			// 1. Sort the waiting list by height of the block
			sortedList := fullnode.SortBlockByHeight(log, pendingBlocks)

			for _, b := range sortedList {
				// 2 . Verify if the previous block is stored in the database
				isPrevBlockStored, err := fullnode.PrevBlockStored(log, b, blockchain)
				if err != nil {
					log.Debugln("Error checking if previous block is stored : %s", err)
				}

				if !isPrevBlockStored {
					pendingBlocks = []*block.Block{}
					requiresPostSync = false
					requiresSync = true // This will call the process of synchronizing the blockchain with the network
					continue
				}

				// 3 . Validation of the block
				if block.IsGenesisBlock(b) {
					if !consensus.ValidateBlock(b, nil) {
						log.Debugln("Genesis block is invalid")
						continue
					}
					log.Debugln("Genesis block is valid")
				} else {
					prevBlock, err := blockchain.GetBlock(log, b.PrevBlock)
					if err != nil {
						log.Debugln("Error getting the previous block : %s\n", err)
					}

					if !consensus.ValidateBlock(b, prevBlock) {
						log.Debugln("Block is invalid")
						continue
					}
					log.Debugln(b.Height, " is valid")
				}

				// 4 . Add the block to the blockchain
				if err := blockchaindb.AddBlockToBlockchain(log, b, blockchain); err != nil {
					log.Debugln("Error adding the block to the blockchain : %s\n", err)
					continue
				}

				if !blockchain.VerifyIntegrity(log) {
					pendingBlocks = []*block.Block{}
					requiresPostSync = false
					requiresSync = true // This will call the process of synchronizing the blockchain with the network
					continue
				}

				// 5 . Add the block transaction to the registry
				if !fullnode.AddBlockTransactionToRegistry(log, cfg, b) {
					log.Debugln("Error adding the block transactions to the registry")
				}

				// 6 . Send the block to IPFS
				if !ipfs.PublishBlock(log, ctx, cfg, nodeIpfs, ipfsAPI, b) {
					log.Debugln("Error publishing the block to IPFS")
				}
			}
			// 6 . Change the state of the node
			requiresPostSync = false
			blockProcessingEnabled = true
			log.Debugln("Post-syncronization done")
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
			if !fullnode.SendBlocksRegistryToNetwork(log, ctx, cfg, receiveBlockchainTopic) {
				log.Debugln("Error sending the registry of the blockchain")
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
			if !fullnode.SendOwnersFiles(log, ctx, cfg, ownerAddressStr, sendFilesTopic) {
				log.Debugln("Error sending the files of the owner")
				continue
			}
		}
	}()

	go func() {
		for {
			// Sleep for random time between 10 seconds and 1min
			randomInt := rand.Intn(60-10+1) + 10
			time.Sleep(time.Duration(randomInt) * time.Second)
			data := &visualisation.Data{
				PeerID:   host.ID().String(),
				NodeType: "StorageNode",
				ConnectedPeers: func() []string {
					peers := make([]string, 0)
					for _, peer := range host.Network().Peers() {
						// check the connectedness of the peer
						if host.Network().Connectedness(peer) != network.Connected {
							continue
						}
						peers = append(peers, peer.String())
					}
					return peers
				}(),
				TopicsList: ps.GetTopics(),
				KeepRelayConnectionAlive: func() []string {
					peers := make([]string, 0)
					for _, peer := range ps.ListPeers("KeepRelayConnectionAlive") {
						// check the connectedness of the peer
						if host.Network().Connectedness(peer) != network.Connected {
							continue
						}
						peers = append(peers, peer.String())
					}
					return peers
				}(),
				BlockAnnouncement: func() []string {
					peers := make([]string, 0)
					for _, peer := range ps.ListPeers("BlockAnnouncement") {
						// check the connectedness of the peer
						if host.Network().Connectedness(peer) != network.Connected {
							continue
						}
						peers = append(peers, peer.String())
					}
					return peers
				}(),
				AskingBlockchain: func() []string {
					peers := make([]string, 0)
					for _, peer := range ps.ListPeers("AskingBlockchain") {
						// check the connectedness of the peer
						if host.Network().Connectedness(peer) != network.Connected {
							continue
						}
						peers = append(peers, peer.String())
					}
					return peers
				}(),
				ReceiveBlockchain: func() []string {
					peers := make([]string, 0)
					for _, peer := range ps.ListPeers("ReceiveBlockchain") {
						// check the connectedness of the peer
						if host.Network().Connectedness(peer) != network.Connected {
							continue
						}
						peers = append(peers, peer.String())
					}
					return peers
				}(),
				ClientAnnouncement: func() []string {
					peers := make([]string, 0)
					for _, peer := range ps.ListPeers("ClientAnnouncement") {
						// check the connectedness of the peer
						if host.Network().Connectedness(peer) != network.Connected {
							continue
						}
						peers = append(peers, peer.String())
					}
					return peers
				}(),
				StorageNodeResponse: func() []string {
					peers := make([]string, 0)
					for _, peer := range ps.ListPeers("StorageNodeResponse") {
						// check the connectedness of the peer
						if host.Network().Connectedness(peer) != network.Connected {
							continue
						}
						peers = append(peers, peer.String())
					}
					return peers
				}(),
				FullNodeAnnouncement: func() []string {
					peers := make([]string, 0)
					for _, peer := range ps.ListPeers("FullNodeAnnouncement") {
						// check the connectedness of the peer
						if host.Network().Connectedness(peer) != network.Connected {
							continue
						}
						peers = append(peers, peer.String())
					}
					return peers
				}(),
				AskMyFilesList: func() []string {
					peers := make([]string, 0)
					for _, peer := range ps.ListPeers("AskMyFilesList") {
						// check the connectedness of the peer
						if host.Network().Connectedness(peer) != network.Connected {
							continue
						}
						peers = append(peers, peer.String())
					}
					return peers
				}(),
				ReceiveMyFilesList: func() []string {
					peers := make([]string, 0)
					for _, peer := range ps.ListPeers("ReceiveMyFilesList") {
						// check the connectedness of the peer
						if host.Network().Connectedness(peer) != network.Connected {
							continue
						}
						peers = append(peers, peer.String())
					}
					return peers
				}(),
			}
			dataBytes, err := json.Marshal(data)
			if err != nil {
				log.Errorf("Failed to marshal NetworkVisualisation message: %s", err)
				continue
			}
			err = networkVisualisationTopic.Publish(ctx, dataBytes)
			if err != nil {
				log.Errorf("Failed to publish NetworkVisualisation message: %s", err)
				continue
			}
			log.Debugf("NetworkVisualisation message sent successfully")
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
