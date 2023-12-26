package storage

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
)

// FileInfo extracts and returns metadata about a given file.
// It provides the file's name, size, and type.
func FileInfo(inputPathFile string) (string, string, string, error) {
	fileInfo, err := os.Stat(inputPathFile)

	if err != nil {
		log.Fatal(err)
	}

	fileSize := strconv.FormatInt(fileInfo.Size(), 10) + " octets"
	fileExtension := filepath.Ext(inputPathFile)
	fileName := fileInfo.Name()

	return fileName, fileSize, fileExtension, err
}
