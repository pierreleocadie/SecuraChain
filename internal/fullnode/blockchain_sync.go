package fullnode

import (
	"context"
	"fmt"

	"github.com/ipfs/boxo/path"
	ipfsLog "github.com/ipfs/go-log/v2"
	icore "github.com/ipfs/kubo/core/coreiface"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pierreleocadie/SecuraChain/internal/blockchaindb"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/ipfs"
	"github.com/pierreleocadie/SecuraChain/internal/registry"
)

// AskForBlockchainRegistry sends a request for the blockchain registry over the network.
func AskForBlockchainRegistry(log *ipfsLog.ZapEventLogger, ctx context.Context, askBlockchain *pubsub.Topic, recBlockchain *pubsub.Subscription) ([]byte, string, error) {
	log.Debugln("Requesting blockchain from the network")
	if err := askBlockchain.Publish(ctx, []byte("I need to synchronize. Who can help me ?")); err != nil {
		log.Errorln("Error publishing blockchain request : ", err)
		return nil, "", err
	}

	registryBytes := make(chan []byte)
	senderID := make(chan string)
	go func() {
		for {
			msg, err := recBlockchain.Next(ctx)
			if err != nil {
				log.Errorln("Error receiving message from network: ", err)
				break
			}
			if msg != nil {
				log.Debugln("Registry received from : ", senderID)
				log.Debugln("Registry received : ", string(msg.Data))
				registryBytes <- msg.Data
				senderID <- msg.GetFrom().String()
				break
			}
		}
	}()

	return <-registryBytes, <-senderID, nil
}

// DownloadMissingBlocks attemps to download blocks that are missing in the local blockchain from IPFS.
func DownloadMissingBlocks(log *ipfsLog.ZapEventLogger, ctx context.Context, ipfsAPI icore.CoreAPI, registryBytes []byte, db *blockchaindb.BlockchainDB) (bool, []*block.Block, error) {
	var missingBlocks []*block.Block

	r, err := registry.DeserializeRegistry[registry.BlockRegistry](log, registryBytes)
	if err != nil {
		log.Errorln("Error converting bytes to block registry : ", err)
		return false, nil, fmt.Errorf("error converting bytes to block registry : %s", err)
	}
	log.Debugln("Registry converted to BlockRegistry : ", r)

	for _, blockData := range r.Blocks {
		if existingBlock, err := db.GetBlock(log, blockData.Key); err == nil && existingBlock != nil {
			log.Errorln("Block already exists in the blockchain : ", blockData.Key)
			continue
		}

		blockPath := path.FromCid(blockData.BlockCid)
		log.Debugln("Converted block CID to pathImmutable : ", blockPath.String())

		if err := ipfsAPI.Swarm().Connect(ctx, blockData.Provider); err != nil {
			log.Errorln("failed to connect to provider: %s", err)
			continue
		}
		log.Debugln("Connected to provider %s", blockData.Provider.ID)

		downloadBlock, err := ipfs.GetBlock(log, ctx, ipfsAPI, blockPath)
		if err != nil {
			log.Errorln("Error downloading block from IPFS : ", err)
			return false, nil, fmt.Errorf("error downloading block from IPFS : %s", err)
		}
		log.Debugln("Block downloaded from IPFS : ", downloadBlock)

		missingBlocks = append(missingBlocks, downloadBlock)
	}

	log.Debugln("Number of missing blocks donwnloaded : ", len(missingBlocks))
	return true, missingBlocks, nil
}

// SendBlocksRegistryToNetwork sends the registry of the blockchain to the network.
func SendBlocksRegistryToNetwork(log *ipfsLog.ZapEventLogger, ctx context.Context, config *config.Config, network *pubsub.Topic) bool {
	// Get the registry of the blockchain
	r, err := registry.LoadRegistryFile[registry.BlockRegistry](log, config, config.BlockRegistryPath)
	if err != nil {
		log.Errorln("Error loading the registry of the blockchain : ", err)
		return false
	}

	registryBytes, err := registry.SerializeRegistry(log, r)
	if err != nil {
		log.Errorln("Error serializing the registry of the blockchain : ", err)
		return false
	}
	if err := network.Publish(ctx, registryBytes); err != nil {
		log.Errorln("Error publishing the registry of the blockchain : ", err)
		return false
	}

	log.Debugln("Blocks registry sent to the network")
	return true
}
