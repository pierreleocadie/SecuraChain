package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/internal/config"
)

type RegistryMessage struct {
	OwnerPublicKey string
	Registry       []FileRegistry
}

// getFileName returns the appropriate file name based on the type of registry.
func getFileName(config *config.Config, registry interface{}) (string, error) {
	switch registry.(type) {
	case IndexingRegistry:
		return config.IndexingRegistryPath, nil
	case BlockRegistry:
		return config.BlockRegistryPath, nil
	default:
		return "", fmt.Errorf("unsupported registry type")
	}
}

// SaveRegistryToFile saves any registry to a JSON file.
func SaveRegistryToFile(log *ipfsLog.ZapEventLogger, config *config.Config, registry interface{}) error {
	data, err := SerializeRegistry(registry)
	if err != nil {
		log.Errorln("Error serializing registry", err)
		return err
	}

	filename, err := getFileName(config, registry)
	if err != nil {
		log.Errorln("Error getting file name", err)
		return err
	}

	log.Debugln("Saving registry to file:", filename)
	return os.WriteFile(filepath.Clean(filename), data, os.FileMode(config.FileRights))
}

// LoadRegistryFile loads the registry data from the specified file path and returns it.
// The registry data is deserialized into the provided generic type R.
// It returns the deserialized registry data and any error encountered during the process.
func LoadRegistryFile[R Registeries](log *ipfsLog.ZapEventLogger, config *config.Config, filename string) (R, error) {
	var registry R

	data, err := os.ReadFile(filepath.Clean(filename))
	if err != nil {
		log.Errorln("Error reading file", err)
		return registry, err
	}

	log.Debugln("Registry loaded successfully")

	if err := json.Unmarshal(data, &registry); err != nil {
		log.Errorln("Error deserializing registry", err)
		return registry, err
	}

	log.Debugln("Registry deserialized successfully")

	return registry, nil
}

// SerializeRegistry serializes any registry into a byte slice using JSON encoding.
func SerializeRegistry(registry interface{}) ([]byte, error) {
	return json.Marshal(registry)
}

// DeserializeRegistry converts a byte slice to a registry struct.
func DeserializeRegistry[R Registeries](log *ipfsLog.ZapEventLogger, data []byte) (R, error) {
	var registry R
	if err := json.Unmarshal(data, &registry); err != nil {
		log.Errorln("Error deserializing registry", err)
		return registry, err
	}

	log.Debugln("Registry deserialized successfully")
	return registry, nil
}
