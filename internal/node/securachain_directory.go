package node

import (
	"fmt"
	"os"
	"path/filepath"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/internal/config"
)

func CreateSecuraChainDataDirectory(log *ipfsLog.ZapEventLogger, config *config.Config) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Errorln("Error getting the user home directory : ", err)
		return "", fmt.Errorf("error getting the user home directory : %v", err)
	}

	defaultPath := filepath.Join(homeDir, config.SecurachainDataPath)
	if err := os.MkdirAll(defaultPath, os.FileMode(config.FileRights)); err != nil {
		log.Errorf("Error creating the SecuraChain data directory : %v", err)
		return "", fmt.Errorf("error creating the SecuraChain data directory : %v", err)
	}

	return defaultPath, nil
}
