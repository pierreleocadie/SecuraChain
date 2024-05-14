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
	"github.com/pierreleocadie/SecuraChain/pkg/aes"
	"github.com/pierreleocadie/SecuraChain/pkg/utils"
)

// DownloadFile downloads a file from IPFS using the provided CID and saves it to the download location of the machine.
func DownloadFile(log *ipfsLog.ZapEventLogger, ctx context.Context, config *config.Config, ipfsAPI icore.CoreAPI,
	cidFile cid.Cid, filename string, extension string, aesKey aes.Key) error {

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

	downloadFilePath := utils.GetDownloadPath(log)

	file := filename + extension
	log.Debugf("File name: %s\n", file)
	downloadFilePathDefault := filepath.Join(downloadFilePath, file)

	fileIPFS := filename + "_IPFS" + extension
	log.Debugf("File name IPFS: %s\n", fileIPFS)
	downloadFilePathIPFS := filepath.Join(downloadFilePath, fileIPFS)
	log.Debugf("Writing file to %s\n", downloadFilePathIPFS)

	err = files.WriteTo(rootNodeFile, downloadFilePathIPFS)
	if err != nil {
		return fmt.Errorf("could not write out the fetched CID: %v", err)
	}

	//Decrypt the file with the AES key
	if err := aesKey.DecryptFile(downloadFilePathIPFS, downloadFilePathDefault); err != nil {
		return fmt.Errorf("could not decrypt the file with aes key: %v", err)
	}

	if err := os.Remove(downloadFilePathIPFS); err != nil {
		return fmt.Errorf("could not remove the inyermediate file %s : %v", downloadFilePathIPFS, err)
	}

	log.Infof("File decrypted and saved to %s\n", downloadFilePathDefault)

	return nil
}
