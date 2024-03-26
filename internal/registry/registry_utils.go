package registry

import (
	"encoding/json"
	"os"
	"path/filepath"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/internal/config"
)

// SaveRegistryToFile saves any registry to a JSON file.
func SaveRegistryToFile(log *ipfsLog.ZapEventLogger, config *config.Config, filePath string, registry interface{}) error {
	data, err := json.Marshal(registry)
	if err != nil {
		log.Errorln("Error serializing registry")
		return err
	}

	return os.WriteFile(filepath.Clean(filePath), data, os.FileMode(config.FileRights))
}

// LoadRegistryFile loads the registry data from the specified file path and returns it.
// The registry data is deserialized into the provided generic type R.
// It returns the deserialized registry data and any error encountered during the process.
func LoadRegistryFile[R any](log *ipfsLog.ZapEventLogger, filePath string) (R, error) {
	var registry R
	data, err := os.ReadFile(filepath.Clean(filePath))
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
func DeserializeRegistry[R any](log *ipfsLog.ZapEventLogger, data []byte) (R, error) {
	var registry R
	if err := json.Unmarshal(data, &registry); err != nil {
		log.Errorln("Error deserializing registry", err)
		return registry, err
	}

	log.Debugln("Registry deserialized successfully")

	return registry, nil
}
