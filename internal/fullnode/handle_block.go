package fullnode

import (
	"context"
	"fmt"

	"github.com/ipfs/kubo/core"
	icore "github.com/ipfs/kubo/core/coreiface"
	"github.com/pierreleocadie/SecuraChain/internal/blockchaindb"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
)

func HandleIncomingBlock(ctx context.Context, config *config.Config, nodeIpfs *core.IpfsNode, ipfsApi icore.CoreAPI, b *block.Block, database *blockchaindb.BlockchainDB) (bool, error) {
	if block.IsGenesisBlock(b) {
		return handleGenesisBlock(ctx, config, nodeIpfs, ipfsApi, b, database)
	}
	return HandleNormalBlock(ctx, config, nodeIpfs, ipfsApi, b, database)
}

func handleGenesisBlock(ctx context.Context, config *config.Config, nodeIpfs *core.IpfsNode, ipfsApi icore.CoreAPI, b *block.Block, database *blockchaindb.BlockchainDB) (bool, error) {

	// Proceed to validate and add the block to the blockchain
	proceed, err := ProcessBlock(b, database)
	if err != nil {
		return false, fmt.Errorf("error processing block : %s", err)
	}

	fmt.Println("Block validate and added to the blockchain : ", proceed)

	// Send the block to IPFS
	if err := blockchaindb.PublishBlockToIPFS(ctx, config, nodeIpfs, ipfsApi, b); err != nil {
		return false, fmt.Errorf("error adding the block to IPFS : %s", err)

	}
	return proceed, nil
}

func HandleNormalBlock(ctx context.Context, config *config.Config, nodeIpfs *core.IpfsNode, ipfsApi icore.CoreAPI, b *block.Block, database *blockchaindb.BlockchainDB) (bool, error) {
	proceed := false

	// Verify if the previous block is stored in the database
	isPrevBlockStored, err := PrevBlockStored(b, database)
	if err != nil {
		return false, fmt.Errorf("error checking if previous block is stored : %s", err)
	}

	if isPrevBlockStored {
		// Proceed to validate and add the block to the blockchain
		proceed, err := ProcessBlock(b, database)
		if err != nil {
			return false, fmt.Errorf("error processing block : %s", err)
		}
		fmt.Println("Block validate and added to the blockchain : ", proceed)

		// Send the block to IPFS
		if err := blockchaindb.PublishBlockToIPFS(ctx, config, nodeIpfs, ipfsApi, b); err != nil {
			return false, fmt.Errorf("error adding the block to IPFS : %s", err)

		}
	}
	return proceed, nil
}
