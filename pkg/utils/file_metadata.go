package utils

import (
	"os"
	"path/filepath"
	"strconv"

	ipfsLog "github.com/ipfs/go-log/v2"
)

// FileInfo extracts and returns metadata about a given file.
// It provides the file's name, size, and type.
func FileInfo(log *ipfsLog.ZapEventLogger, filePath string) (string, string, string, error) {
	sanitizedFilePath, err := SanitizePath(filePath)
	if err != nil {
		log.Errorln("Error on sanitizing path %s", err)
	}

	fileInfo, err := os.Stat(sanitizedFilePath)
	if err != nil {
		log.Errorln("Error on getting file info %s", err)
	}

	fileSize := strconv.FormatInt(fileInfo.Size(), 10) + " bytes"
	fileExtension := filepath.Ext(filePath)
	fileName := fileInfo.Name()

	return fileName, fileSize, fileExtension, err
}
