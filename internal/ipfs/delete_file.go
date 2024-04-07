package ipfs

import (
	"os"
	"path/filepath"

	ipfsLog "github.com/ipfs/go-log/v2"
)

// This function allows to delete a file from IPFS, and from the storer node (precisely unpin it)
func DeleteFile(log *ipfsLog.ZapEventLogger, fileName string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Errorln("Error getting user home directory")
		return err
	}

	// if err != nil {
	// 	log.Fatal(err)
	// }

	outputBasePath := filepath.Join(home, ".IPFS_Local_Storage/")
	outputBaseFile := filepath.Join(outputBasePath, fileName)
	if err := os.Remove(outputBaseFile); err != nil {
		log.Errorln("Error deleting file from local storage")
		return err
	}

	log.Debugln("File deleted from", outputBaseFile)
	return nil
}
