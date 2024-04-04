package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/internal/config"
)

type Registeries interface {
	IndexingRegistry | FileRegistry | BlockRegistry | BlockData | RegistryMessage
}

type RegistryMessage struct {
	OwnerPublicKey string
	Registry       []FileRegistry
}

// SaveRegistryToFile saves any registry to a JSON file.
func SaveRegistryToFile(log *ipfsLog.ZapEventLogger, config *config.Config, filename string, registry interface{}) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	defaultFilePath := filepath.Join(home, ".securachainData/")
	// Ensure the output directory exists or create it.
	if err := os.MkdirAll(defaultFilePath, os.FileMode(config.FileRights)); err != nil {
		return fmt.Errorf("error creating output directory: %v", err)
	}

	if err = os.Chdir(defaultFilePath); err != nil {
		return err
	}

	log.Debugln("Changing directory to", defaultFilePath)

	data, err := json.Marshal(registry)
	if err != nil {
		log.Errorln("Error serializing registry")
		return err
	}

	return os.WriteFile(filepath.Clean(filename), data, os.FileMode(config.FileRights))
}

// LoadRegistryFile loads the registry data from the specified file path and returns it.
// The registry data is deserialized into the provided generic type R.
// It returns the deserialized registry data and any error encountered during the process.
func LoadRegistryFile[R Registeries](log *ipfsLog.ZapEventLogger, config *config.Config, filename string) (R, error) {
	var registry R

	home, err := os.UserHomeDir()
	if err != nil {
		return registry, err
	}

	defaultFilePath := filepath.Join(home, ".securachainData/")
	// Ensure the output directory exists or create it.
	if err := os.MkdirAll(defaultFilePath, os.FileMode(config.FileRights)); err != nil {
		return registry, fmt.Errorf("error creating output directory: %v", err)
	}

	if err = os.Chdir(defaultFilePath); err != nil {
		return registry, err
	}

	log.Debugln("Changing directory to", defaultFilePath)

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
func SerializeRegistry(log *ipfsLog.ZapEventLogger, registry interface{}) ([]byte, error) {
	log.Debugln("Serializing registry")
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
