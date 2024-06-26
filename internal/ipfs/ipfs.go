package ipfs

import (
	"context"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/ipfs/kubo/core"
	icore "github.com/ipfs/kubo/core/coreiface"
	"github.com/pierreleocadie/SecuraChain/internal/config"
)

type IPFSNode struct {
	Ctx  context.Context
	Node *core.IpfsNode
	API  icore.CoreAPI
	log  *ipfsLog.ZapEventLogger
	cfg  *config.Config
}

func NewIPFSNode(ctx context.Context, log *ipfsLog.ZapEventLogger, cfg *config.Config) *IPFSNode {
	api, node, err := SpawnNode(ctx, cfg)
	if err != nil {
		log.Fatalf("Could not spawn IPFS node: %v", err)
	}
	return &IPFSNode{
		Ctx:  ctx,
		Node: node,
		API:  api,
		log:  log,
		cfg:  cfg,
	}
}
