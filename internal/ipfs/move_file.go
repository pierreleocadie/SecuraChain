package ipfs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// MoveFile moves a file from the source path to the destination path.
// It returns an error if the move operation fails.
func MoveFile(source, destination string) error {
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

	// Sync the file content
	if err = outFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync file content: %w", err)
	}

	// Remove the source file
	if err = os.Remove(filepath.Clean(source)); err != nil {
		return fmt.Errorf("failed to remove source file: %w", err)
	}

	return nil
}
