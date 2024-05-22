package ipfs

import (
	"context"

	"github.com/ipfs/boxo/path"
	ipfsLog "github.com/ipfs/go-log/v2"
	icore "github.com/ipfs/kubo/core/coreiface"
)

func PinFile(log *ipfsLog.ZapEventLogger, ctx context.Context, ipfsAPI icore.CoreAPI, fileCid path.ImmutablePath) (bool, error) {
	if err := ipfsAPI.Pin().Add(ctx, fileCid); err != nil {
		log.Errorln("Could not pin file %v", err)
		return false, err
	}

	_, IsPinned, err := ipfsAPI.Pin().IsPinned(ctx, fileCid)
	if err != nil {
		log.Errorln("Could not check if file is pinned %v", err)
		return IsPinned, err
	}

	log.Infoln("File pinned: %v", IsPinned)
	return IsPinned, nil
}
