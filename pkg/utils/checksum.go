package utils

import (
	"crypto/sha256"
	"os"
)

func ComputeFileChecksum(filePath string) ([]byte, error) {
	filePath, err := SanitizePath(filePath)
	if err != nil {
		return nil, err
	}

	fileBytes, err := os.ReadFile(filePath) // #nosec G304
	if err != nil {
		return nil, err
	}

	checksum := sha256.Sum256(fileBytes)

	return checksum[:], nil
}
