package ipfs

import (
	"context"

	"github.com/ipfs/boxo/path"
	ipfsLog "github.com/ipfs/go-log/v2"
	icore "github.com/ipfs/kubo/core/coreiface"
)

func UnpinFile(log *ipfsLog.ZapEventLogger, ctx context.Context, ipfsAPI icore.CoreAPI, fileCid path.ImmutablePath) (bool, error) {
	if err := ipfsAPI.Pin().Rm(ctx, fileCid); err != nil {
		log.Errorln("Could not unpin file %v", err)
		return false, err
	}

	_, IsUnpinned, err := ipfsAPI.Pin().IsPinned(ctx, fileCid)
	if err != nil {
		log.Errorln("Could not check if file is unpinned %v", err)
		return IsUnpinned, err
	}

	log.Debugln("File unpinned: %v", IsUnpinned)
	return IsUnpinned, nil
}
