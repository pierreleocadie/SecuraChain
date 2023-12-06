package internal

import (
	"encoding/json"
	"os"
	"time"
)

type FileMetaData struct {
	Cid           string    `json:"cid"`           // Id unique du fichier dans IPFS
	Timestamp     time.Time `json:"timestamp"`     // Tumestamp de l'ajout du fichier
	FileSize      string    `json:"fileSize"`      // Taille du fichier en octet
	FileType      string    `json:"fileType"`      // Type de fichier (extension)
	OriginalName  string    `json:"originalName"`  // Nom original du fichier
	UserPublicKey string    `json:"userPublicKey"` // Clé publique de l'utilisateur ayant ajouté le fichier
}

type CIDStorage struct {
	Files []FileMetaData `json:"files"`
}

func SaveToJSON(filePath string, data CIDStorage) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, jsonData, 0644)
}

func LoadFromJSON(filePath string) (CIDStorage, error) {
	var data CIDStorage

	jsonData, err := os.ReadFile(filePath)
	if err != nil {
		return data, err
	}

	err = json.Unmarshal(jsonData, &data)
	return data, err
}
