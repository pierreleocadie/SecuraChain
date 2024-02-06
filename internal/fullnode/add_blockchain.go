package fullnode

import (
	"context"
	"fmt"

	"github.com/ipfs/boxo/path"
	"github.com/ipfs/kubo/core"
	icore "github.com/ipfs/kubo/core/coreiface"
	"github.com/pierreleocadie/SecuraChain/internal/ipfs"
)

func AddBlockchainToIPFS(ctx context.Context, nodeIpfs *core.IpfsNode, ipfsApi icore.CoreAPI, oldCid path.ImmutablePath) (path.ImmutablePath, error) {
	/*
	* Fifth step : SYNCHRONIZATION AND RE-SYNCHRONIZATION of nodes in case of absence
	 */

	// Add the blockchain to IPFS
	fileImmutablePathCid, err := ipfs.AddFile(ctx, nodeIpfs, ipfsApi, "./blockchain")
	if err != nil {
		fmt.Println("Error adding the blockhain to IPFS : ", err)
	}

	// Pin the file on IPFS
	pinned, err := ipfs.PinFile(ctx, ipfsApi, fileImmutablePathCid)
	if err != nil {
		fmt.Println("Error pinning the blockchain to IPFS : ", err)
	}
	fmt.Println("Blockchain pinned on IPFS : ", pinned)

	if oldCid.RootCid().Defined() {
		_, err := ipfs.UnpinFile(ctx, ipfsApi, oldCid)
		if err != nil {
			fmt.Println("Error unpinning the blockchain to IPFS : ", err)
		}
	}
	oldCid = fileImmutablePathCid

	return oldCid, nil
}
