package util

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CopyFile copies a file from the source path to the destination path.
// It returns an error if the copy operation fails.
func CopyFile(source, destination string) error {
	// Open the source file
	inFile, err := os.Open(filepath.Clean(source))
	if err != nil {
		return fmt.Errorf("failed to open source file %w", err)
	}
	defer inFile.Close()

	// Create the destination file
	outFile, err := os.Create(filepath.Clean(destination))
	if err != nil {
		return fmt.Errorf("failed to create destination file %w", err)
	}
	defer outFile.Close()

	// Copy the content
	_, err = io.Copy(outFile, inFile)
	if err != nil {
		return fmt.Errorf("failed to copy the file content %w", err)
	}

	// Verify that the data is written well on the dis
	if err = outFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync file content: %w", err)
	}

	return nil
}
