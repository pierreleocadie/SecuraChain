// Package util provides utility functions for managing file metadata and CID records.
package util

import (
	"encoding/json"
	"os"
	"time"
)

// FileMetaData represents the metadata associated with a file stored in the storage node.
type FileMetaData struct {
	Cid           string    `json:"cid"`           // Unique identifier of the file in IPFS.
	Timestamp     time.Time `json:"timestamp"`     // Timestamp of when the file was added.
	FileSize      string    `json:"fileSize"`      // Size of the file in bytes.
	FileType      string    `json:"fileType"`      // File type (extension).
	OriginalName  string    `json:"originalName"`  // Original name of the file.
	UserPublicKey string    `json:"userPublicKey"` // Public key of the user who added the file.
}

// CIDStorage represents a collection of file metadata records.
type CIDStorage struct {
	Files []FileMetaData `json:"files"`
}

func WriteAJSONFile(filePath string) (string, error) {
	file, err := os.Open(filePath)

	if err != nil {
		return "", err
	}
	defer file.Close()

	return filePath, err
}

// SaveToJSON saves the CID metadata records to a JSON file.
// filePath: Path to the JSON file where data will be saved.
// data: CIDStorage object containing the metadata records.
// Returns an error if the saving process fails.
func SaveToJSON(filePath string, data CIDStorage) error {

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, jsonData, 0644)
}

// LoadFromJSON loads the CID metadata records from a JSON file.
// filePath: Path to the JSON file to load the data from.
// Returns a CIDStorage object containing the metadata records and an error if the loading process fails.
func LoadFromJSON(filePath string) (CIDStorage, error) {
	var data CIDStorage

	jsonData, err := os.ReadFile(filePath)
	if err != nil {
		return data, err
	}

	err = json.Unmarshal(jsonData, &data) // Parse the JSON data into the CIDStorage struct.
	return data, err
}
