package internal

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
)

func FileInfo(inputPathFile string) (string, string, string, error) {
	fileInfo, err := os.Stat(inputPathFile)

	if err != nil {
		log.Fatal(err)
	}

	fileSize := strconv.FormatInt(fileInfo.Size(), 10) + " octets"
	fileType := filepath.Ext(inputPathFile)
	fileName := fileInfo.Name()

	return fileName, fileSize, fileType, err

}
