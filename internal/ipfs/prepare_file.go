package ipfs

import (
	"os"

	"github.com/ipfs/boxo/files"
	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/pkg/utils"
)

// prepareFileForIPFS prepares a file to be added to IPFS by creating a UnixFS node from the given path.
// It retrieves file information and creates a serial file node for IPFS.
func PrepareFileForIPFS(log *ipfsLog.ZapEventLogger, path string) (files.Node, error) {
	sanitizedPath, err := utils.SanitizePath(log, path)
	if err != nil {
		return nil, err
	}

	stat, err := os.Stat(sanitizedPath)
	if err != nil {
		return nil, err
	}

	fileNode, err := files.NewSerialFile(path, false, stat)
	if err != nil {
		return nil, err
	}

	return fileNode, nil
}
