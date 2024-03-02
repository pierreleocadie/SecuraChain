package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/path"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/fullnode"
	"github.com/pierreleocadie/SecuraChain/internal/ipfs"
)

func main() {
	ctx := context.Background()

	oldCid := path.ImmutablePath{}

	cfg, err := config.LoadConfig("./config-test.yml")
	if err != nil {
		fmt.Printf("error loading config file : %v", err)
	}

	ipfsAPI, nodeIpfs, err := ipfs.SpawnNode(ctx, cfg)
	if err != nil {
		fmt.Printf("error spawning IPFS node : %v", err)
	}

	newCidBlockChain, err := fullnode.PublishBlockchainToIPFS(ctx, cfg, nodeIpfs, ipfsAPI, oldCid)
	if err != nil {
		fmt.Printf("error publishing blockchain to IPFS : %v", err)
	}

	outputBasePath, err := os.MkdirTemp("", "example")
	if err != nil {
		fmt.Printf("could not create output dir (%v)", err)
	}

	outputPathDirectory := outputBasePath + strings.Split(newCidBlockChain.String(), "/")[2]

	rootNodeDirectory, err := ipfsAPI.Unixfs().Get(ctx, newCidBlockChain)
	if err != nil {
		fmt.Printf("error getting blockchain from IPFS : %v", err)
	}

	err = files.WriteTo(rootNodeDirectory, "./")
	if err != nil {
		fmt.Printf("error writing blockchain to file : %v", err)
	}

	fmt.Printf("Got directory back from IPFS (IPFS path: %s) and wrote it to %s\n", newCidBlockChain.String(), outputPathDirectory)

}
