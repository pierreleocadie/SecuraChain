package util

import (
	"io"
	"os"
)

func CopyFile(source, destination string) error {
	// Open the source file
	inFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer inFile.Close()

	// Create the destination file
	outFile, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// Copy the content
	_, err = io.Copy(outFile, inFile)
	if err != nil {
		return err
	}

	return outFile.Sync() // Verify that the data is written well on the disk
}
