package ipfs

import (
	"context"
	"fmt"

	files "github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/path"
	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/ipfs/kubo/core"
	icore "github.com/ipfs/kubo/core/coreiface"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/registry"
)

// PublishBlock publishes a block to IPFS and adds its metadata to the registry.
func PublishBlock(log *ipfsLog.ZapEventLogger, ctx context.Context, config *config.Config, nodeIpfs *core.IpfsNode, ipfsApi icore.CoreAPI, b *block.Block) bool {
	// Add the block to IPFS
	cidFile, err := addBlock(log, ctx, ipfsApi, b)
	if err != nil {
		log.Errorln("Error adding the block to IPFS")
		return false
	}

	// Pin the file on IPFS
	_, err = PinFile(log, ctx, ipfsApi, cidFile)
	if err != nil {
		log.Errorln("Error pinning the block to IPFS")
		return false
	}
	log.Debugln("block pinned on IPFS")

	// Get the node ID
	nodeId := nodeIpfs.Peerstore.PeerInfo(nodeIpfs.Identity)

	// Save the block metadata to the registry
	if err := registry.AddBlockToRegistry(log, b, config, cidFile, nodeId); err != nil {
		log.Errorln("Error adding the block metadata to the registry")
		return false
	}

	log.Debugln("block metadata added to the registry")
	return true
}

// addBlock serializes the given block and adds it to IPFS.
func addBlock(log *ipfsLog.ZapEventLogger, ctx context.Context, ipfsAPI icore.CoreAPI, b *block.Block) (path.ImmutablePath, error) {
	blockBytes, err := b.Serialize(log)
	if err != nil {
		log.Errorln("Error serializing block: ", err)
		return path.ImmutablePath{}, fmt.Errorf("could not serialize block: %s", err)
	}
	log.Debugln("Block serialized")

	cidFile, err := ipfsAPI.Unixfs().Add(ctx,
		files.NewBytesFile(blockBytes))
	if err != nil {
		log.Errorln("Error adding File: ", err)
		return path.ImmutablePath{}, fmt.Errorf("could not add File: %s", err)
	}

	log.Debugln("Block added to IPFS with CID: ", cidFile.String())
	return cidFile, nil
}
