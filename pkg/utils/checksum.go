package utils

import (
	"crypto/sha256"
	"os"

	ipfsLog "github.com/ipfs/go-log/v2"
)

func ComputeFileChecksum(log *ipfsLog.ZapEventLogger, filePath string) ([]byte, error) {
	filePath, err := SanitizePath(log, filePath)
	if err != nil {
		log.Errorln("Error sanitizing file path")
		return nil, err
	}

	fileBytes, err := os.ReadFile(filePath) // #nosec G304
	if err != nil {
		log.Errorln("Error reading file")
		return nil, err
	}

	checksum := sha256.Sum256(fileBytes)

	log.Debugln("Checksum computed successfully")
	return checksum[:], nil
}
