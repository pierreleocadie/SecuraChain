package util

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const permission = 600

// FileMetaData represents the metadata associated with a file stored in the storage node.
type FileMetaData struct {
	Cid           string    `json:"cid"`           // Unique identifier of the file in IPFS.
	Timestamp     time.Time `json:"timestamp"`     // Timestamp of when the file was added.
	FileSize      string    `json:"fileSize"`      // Size of the file in bytes.
	Extension     string    `json:"extension"`     // L'extension du fichier.
	OriginalName  string    `json:"originalName"`  // Original name of the file.
	UserPublicKey string    `json:"userPublicKey"` // Public key of the user who added the file.
}

// CIDStorage represents a collection of file metadata records.
type CIDStorage struct {
	Files []FileMetaData `json:"files"`
}

func WriteAJSONFile(filePath string) (string, error) {
	file, err := os.Open(filepath.Clean(filePath))

	if err != nil {
		return "", err
	}
	defer file.Close()

	return filePath, err
}

// SaveToJSON saves the CID metadata records to a JSON file.
func SaveToJSON(filePath string, data CIDStorage) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Clean(filePath), jsonData, permission)
}

// LoadFromJSON loads the CID metadata records from a JSON file.
func LoadFromJSON(filePath string) (CIDStorage, error) {
	var data CIDStorage

	jsonData, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return data, err
	}

	err = json.Unmarshal(jsonData, &data) // Parse the JSON data into the CIDStorage struct.
	return data, err
}
