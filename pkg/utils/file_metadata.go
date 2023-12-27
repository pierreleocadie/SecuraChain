package utils

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
)

// FileInfo extracts and returns metadata about a given file.
// It provides the file's name, size, and type.
func FileInfo(filePath string) (string, string, string, error) {
	sanitizedFilePath, err := SanitizePath(filePath)
	if err != nil {
		log.Printf("Error on sanitizing path %s", err)
	}

	fileInfo, err := os.Stat(sanitizedFilePath)
	if err != nil {
		log.Printf("Error on getting file info %s", err)
	}

	fileSize := strconv.FormatInt(fileInfo.Size(), 10) + " bytes"
	fileExtension := filepath.Ext(filePath)
	fileName := fileInfo.Name()

	return fileName, fileSize, fileExtension, err
}
