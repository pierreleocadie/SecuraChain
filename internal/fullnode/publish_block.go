package fullnode

import (
	"context"
	"fmt"

	"github.com/ipfs/kubo/core"
	icore "github.com/ipfs/kubo/core/coreiface"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/ipfs"
	"github.com/pierreleocadie/SecuraChain/internal/pebble"
)

// PublishBlockchainToIPFS adds the local blockchain to IPFS for synchronization and re-synchronization of nodes in case of absence.
func PublishBlockToIPFS(ctx context.Context, config *config.Config, nodeIpfs *core.IpfsNode, ipfsApi icore.CoreAPI, b *block.Block) error {
	// Add the blockchain to IPFS
	fileImmutablePathCid, err := ipfs.AddBlockToIPFS(ctx, config, ipfsApi, b)
	if err != nil {
		fmt.Printf("Error adding the block to IPFS : %s\n ", err)
		return err
	}

	// Pin the file on IPFS
	pinned, err := ipfs.PinFile(ctx, ipfsApi, fileImmutablePathCid)
	if err != nil {
		fmt.Printf("Error pinning the block to IPFS : %s\n", err)
		return err
	}
	fmt.Println("block pinned on IPFS : ", pinned)

	// Save the file CID to the blockchain registry
	if err := pebble.AddBlockMetadataToRegistry(b, config, fileImmutablePathCid); err != nil {
		fmt.Printf("Error adding the block metadata to the registry : %s\n", err)
		return err
	}

	return nil
}
