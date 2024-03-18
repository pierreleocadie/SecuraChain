package blockchaindb

import (
	"context"
	"fmt"

	files "github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/path"
	"github.com/ipfs/kubo/core"
	icore "github.com/ipfs/kubo/core/coreiface"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/ipfs"
)

// PublishBlockchainToIPFS adds the local blockchain to IPFS for synchronization and re-synchronization of nodes in case of absence.
func PublishBlockToIPFS(ctx context.Context, config *config.Config, nodeIpfs *core.IpfsNode, ipfsApi icore.CoreAPI, b *block.Block) error {
	// Add the blockchain to IPFS
	fileImmutablePathCid, err := addBlockToIPFS(ctx, ipfsApi, b)
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

	nodeId := nodeIpfs.Peerstore.PeerInfo(nodeIpfs.Identity)

	fmt.Println("[PublishBlockTo IPFS] Block added : ", fileImmutablePathCid)
	// Save the file CID to the blockchain registry
	if err := AddBlockMetadataToRegistry(b, config, fileImmutablePathCid, nodeId); err != nil {
		fmt.Printf("Error adding the block metadata to the registry : %s\n", err)
		return err
	}

	return nil
}

// AddFileToIPFS adds a file to IPFS and returns its CID. It also collects and saves file metadata.
// The function handles the file addition process and records metadata such as file size, type, name, and user public key.
func addBlockToIPFS(ctx context.Context, ipfsAPI icore.CoreAPI, b *block.Block) (path.ImmutablePath, error) {
	blockBytes, err := b.Serialize()
	if err != nil {
		return path.ImmutablePath{}, fmt.Errorf("could not serialize block: %s", err)
	}

	peerCidFile, err := ipfsAPI.Unixfs().Add(ctx,
		files.NewBytesFile(blockBytes))
	if err != nil {
		return path.ImmutablePath{}, fmt.Errorf("could not add File: %s", err)
	}

	fmt.Printf("Added file to peer with CID %s\n", peerCidFile.String())

	return peerCidFile, nil

}
