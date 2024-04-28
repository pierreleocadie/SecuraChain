package client

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/path"
	"github.com/ipfs/go-cid"

	ipfsLog "github.com/ipfs/go-log/v2"
	icore "github.com/ipfs/kubo/core/coreiface"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/pkg/utils"
)

// DownloadFile downloads a file from IPFS using the provided CID and saves it to the download location of the machine.
func DownloadFile(log *ipfsLog.ZapEventLogger, ctx context.Context, config *config.Config, ipfsAPI icore.CoreAPI,
	cidFile cid.Cid, filename string, extension string) error {

	log.Debugf("Downloading file with CID: %s\n", cidFile.String())

	cidPath := path.FromCid(cidFile)
	log.Debugf("Converted CID to path.ImmutablePath: %s\n", cidPath.String())

	rootNodeFile, err := ipfsAPI.Unixfs().Get(ctx, cidPath)
	if err != nil {
		return fmt.Errorf("could not get file with CID: %s", err)
	}
	defer rootNodeFile.Close()
	log.Debugln("Got the file")

	if err := os.MkdirAll(utils.GetDownloadPath(log), os.FileMode(config.FileRights)); err != nil {
		return fmt.Errorf("error creating output directory: %v", err)
	}

	file := filename + extension
	log.Debugf("File name: %s\n", file)

	downloadFilePath := utils.GetDownloadPath(log)
	downloadFilePath = filepath.Join(downloadFilePath, file)
	log.Debugf("Writing file to %s\n", downloadFilePath)

	err = files.WriteTo(rootNodeFile, downloadFilePath)
	if err != nil {
		return fmt.Errorf("could not write out the fetched CID: %v", err)
	}

	log.Infof("File downloaded back from IPFS (IPFS path: %s) and wrote it to %s\n", cidFile.String(), downloadFilePath)

	return nil
}
