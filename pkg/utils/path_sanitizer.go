// Package utils provides utility functions for file operations and other
// common tasks in the SecuraChain project.package utils
package utils

import (
	"fmt"
	"path/filepath"

	ipfsLog "github.com/ipfs/go-log/v2"
)

// SanitizePath sanitizes a given file path to prevent potential security vulnerabilities.
// This function takes a file path as input and returns its absolute and cleaned version.
// The function ensures the path is absolute and eliminates any ".." or similar patterns that could be a security risk.
//
// Parameters:
// - path: The file path that needs to be sanitized.
//
// Returns:
// - A cleaned, absolute path as a string.
// - An error if the operation fails, for instance, if it fails to convert the path to an absolute path.
// Package utils provides utility functions for file operations and other
func SanitizePath(log *ipfsLog.ZapEventLogger, path string) (string, error) {
	// Make the path absolute
	absPath, err := filepath.Abs(path)
	if err != nil {
		log.Errorln("Error getting absolute path")
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	cleanPath := filepath.Clean(absPath)

	log.Debugln("Path sanitized successfully")
	return cleanPath, nil
}
