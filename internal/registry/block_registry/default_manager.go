package blockregistry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/internal/config"
)

type DefaultBlockRegistryManager struct {
	filename string
	log      *ipfsLog.ZapEventLogger
	config   *config.Config
}

func NewDefaultBlockRegistryManager(log *ipfsLog.ZapEventLogger, config *config.Config, filename string) *DefaultBlockRegistryManager {
	return &DefaultBlockRegistryManager{
		filename: filename,
		log:      log,
		config:   config,
	}
}

func (drm DefaultBlockRegistryManager) Save(registry BlockRegistry) error {
	data, err := json.Marshal(registry)
	if err != nil {
		return fmt.Errorf("error serializing registry: %v", err)
	}

	drm.log.Debugln("Saving registry to file:", drm.filename)
	return os.WriteFile(filepath.Clean(drm.filename), data, os.FileMode(drm.config.FileRights))
}

func (drm DefaultBlockRegistryManager) Load() (BlockRegistry, error) {
	var registry DefaultBlockRegistry
	data, err := os.ReadFile(filepath.Clean(drm.filename))
	if err != nil {
		drm.log.Errorln("Error reading file", err)
		return &registry, err
	}

	drm.log.Debugln("Registry loaded successfully")

	if err := json.Unmarshal(data, &registry); err != nil {
		drm.log.Errorln("Error deserializing registry", err)
		return &registry, err
	}

	drm.log.Debugln("Registry deserialized successfully")

	return &registry, nil
}
