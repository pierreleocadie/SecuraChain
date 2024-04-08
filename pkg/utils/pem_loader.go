// Package utils provides utility functions for file operations and other
// common tasks in the SecuraChain project.
package utils

import (
	"encoding/pem"
	"fmt"
	"os"
	"path"
)

// fileExtPEM is the file extension for PEM files.
const fileExtPEM = ".pem"

// LoadKeyFromFile reads a key from a PEM-formatted file and returns its bytes.
// The function takes the filename without extension and the storage path as arguments.
// It returns the raw key bytes or an error if the file cannot be read or parsed.
func LoadKeyFromFile(filename, storagePath string) ([]byte, error) {
	// Construct and clean the file path.
	// Joining filename and storagePath to create the complete path to the key file.
	filePath := path.Join(storagePath, filename+fileExtPEM)

	// Sanitize the constructed file path.
	cleanFilePath, err := SanitizePath(filePath)
	if err != nil {
		return nil, err
	}

	// Read the PEM file bytes.
	// os.ReadFile reads the file named by cleanFilePath and returns the contents.
	fileBytes, err := os.ReadFile(cleanFilePath) // #nosec G304
	if err != nil {
		return nil, err
	}

	// Decode the PEM file to get the raw key bytes.
	// The function pem.Decode parses the PEM encoded fileBytes and returns a PEM block
	// containing the key's DER encoded bytes and any extra bytes following the PEM block.
	block, extraBytes := pem.Decode(fileBytes)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from %s", filename)
	}
	if len(extraBytes) > 0 {
		return nil, fmt.Errorf("unexpected extra bytes in PEM file %s", filename)
	}

	// Return the raw key bytes.
	return block.Bytes, nil
}
