package blockchaindb

import (
	"context"
	"fmt"

	"github.com/ipfs/boxo/path"
	ipfsLog "github.com/ipfs/go-log/v2"
	icore "github.com/ipfs/kubo/core/coreiface"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
)

// AskForBlockchainRegistry sends a request for the blockchain registry over the network.
func AskForBlockchainRegistry(log *ipfsLog.ZapEventLogger, ctx context.Context, askBlockchain *pubsub.Topic, recBlockchain *pubsub.Subscription) ([]byte, string, error) {
	log.Debugln("Requesting blockchain from the network")
	if err := askBlockchain.Publish(ctx, []byte("I need to synchronize. Who can help me ?")); err != nil {
		return nil, "", fmt.Errorf("error publishing blockchain request %s", err)
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
				log.Debugln("Blockchain received from the network")
				registryBytes <- msg.Data
				senderID <- msg.GetFrom().String()
				break
			}
		}
	}()

	return <-registryBytes, <-senderID, nil
}

// DownloadMissingBlocks attemps to download blocks that are missing in the local blockchain from IPFS.
func DownloadMissingBlocks(log *ipfsLog.ZapEventLogger, ctx context.Context, ipfsAPI icore.CoreAPI, registryBytes []byte, db *BlockchainDB) (bool, []*block.Block, error) {
	var missingBlocks []*block.Block

	registry, err := ConvertByteToBlockRegistry(registryBytes)
	if err != nil {
		return false, nil, fmt.Errorf("error converting bytes to block registry : %s", err)
	}
	log.Debugln("Registry converted to BlockRegistry : ", registry)

	for _, blockData := range registry.Blocks {
		if existingBlock, err := db.GetBlock(blockData.Key); err == nil && existingBlock != nil {
			log.Debugln("Block already present in the blockchain")
			continue
		}

		blockPath := path.FromCid(blockData.BlockCid)
		log.Debugln("Converted block CID to path : ", blockPath.String())

		if err := ipfsAPI.Swarm().Connect(ctx, blockData.Provider); err != nil {
			log.Errorln("failed to connect to provider: %s", err)
			continue
		}
		log.Debugln("Connected to provider %s", blockData.Provider.ID)

		downloadBlock, err := GetBlockFromIPFS(ctx, ipfsAPI, blockPath)
		if err != nil {
			return false, nil, fmt.Errorf("error downloading block from IPFS : %s", err)
		}
		log.Debugln("Block downloaded from IPFS : ", downloadBlock)

		missingBlocks = append(missingBlocks, downloadBlock)
	}

	log.Debugln("Number of missing blocks donwnloaded : ", len(missingBlocks))
	return true, missingBlocks, nil
}
