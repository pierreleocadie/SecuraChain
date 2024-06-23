package ipfs

import (
	"fmt"

	files "github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/path"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
)

// PublishBlock publishes a block to IPFS and adds its metadata to the registry.
func (ipfs *IPFSNode) PublishBlock(b block.Block) (path.ImmutablePath, peer.AddrInfo, error) {
	// Add the block to IPFS
	cidFile, err := ipfs.addBlock(b)
	if err != nil {
		return path.ImmutablePath{}, peer.AddrInfo{}, fmt.Errorf("could not add block to IPFS: %v", err)
	}

	// Pin the file on IPFS
	if err := ipfs.PinFile(cidFile); err != nil {
		return path.ImmutablePath{}, peer.AddrInfo{}, fmt.Errorf("could not pin block to IPFS: %v", err)
	}
	ipfs.log.Debugln("block pinned on IPFS")

	// Get the node ID
	nodeId := ipfs.Node.Peerstore.PeerInfo(ipfs.Node.Identity)

	return cidFile, nodeId, nil
}

// addBlock serializes the given block and adds it to IPFS.
func (ipfs *IPFSNode) addBlock(b block.Block) (path.ImmutablePath, error) {
	blockBytes, err := b.Serialize()
	if err != nil {
		return path.ImmutablePath{}, fmt.Errorf("could not serialize block: %s", err)
	}
	ipfs.log.Debugln("Block serialized")

	cidFile, err := ipfs.API.Unixfs().Add(ipfs.Ctx, files.NewBytesFile(blockBytes))
	if err != nil {
		return path.ImmutablePath{}, fmt.Errorf("could not add File: %s", err)
	}

	ipfs.log.Debugln("Block added to IPFS with CID: ", cidFile.String())
	return cidFile, nil
}
