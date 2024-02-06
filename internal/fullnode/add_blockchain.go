package fullnode

import (
	"context"
	"fmt"

	"github.com/ipfs/boxo/path"
	"github.com/ipfs/kubo/core"
	icore "github.com/ipfs/kubo/core/coreiface"
	"github.com/pierreleocadie/SecuraChain/internal/ipfs"
)

// AddBlockchainToIPFS adds the local blockchain to IPFS for synchronization and re-synchronization of nodes in case of absence.
func AddBlockchainToIPFS(ctx context.Context, nodeIpfs *core.IpfsNode, ipfsApi icore.CoreAPI, oldCid path.ImmutablePath) (path.ImmutablePath, error) {
	// Add the blockchain to IPFS
	fileImmutablePathCid, err := ipfs.AddFile(ctx, nodeIpfs, ipfsApi, "./blockchain")
	if err != nil {
		fmt.Printf("Error adding the blockhain to IPFS : %s\n ", err)
		return path.ImmutablePath{}, err
	}

	// Pin the file on IPFS
	pinned, err := ipfs.PinFile(ctx, ipfsApi, fileImmutablePathCid)
	if err != nil {
		fmt.Printf("Error pinning the blockchain to IPFS : %s\n", err)
		return path.ImmutablePath{}, err
	}
	fmt.Println("Blockchain pinned on IPFS : ", pinned)

	// Unpin the old CID if defined
	if oldCid.RootCid().Defined() {
		_, err := ipfs.UnpinFile(ctx, ipfsApi, oldCid)
		if err != nil {
			fmt.Printf("Error unpinning the old blockchain from IPFS : %s\n", err)
			// Not returning the error on unpin to allow process continuation
		}
	}
	// Return the new CID for the blockchain
	return fileImmutablePathCid, nil
}
