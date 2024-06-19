package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/internal/config"
)

type RegistryManager interface {
	Save(registry Registry) error
	Load(registry Registry) (Registry, error)
}

type DefaultRegistryManager struct {
	filename string
	log      *ipfsLog.ZapEventLogger
	config   *config.Config
}

func NewDefaultRegistryManager(log *ipfsLog.ZapEventLogger, config *config.Config, filename string) *DefaultRegistryManager {
	return &DefaultRegistryManager{
		filename: filename,
		log:      log,
		config:   config,
	}
}

func (drm DefaultRegistryManager) Save(registry Registry) error {
	data, err := SerializeRegistry(registry)
	if err != nil {
		return fmt.Errorf("error serializing registry: %v", err)
	}

	drm.log.Debugln("Saving registry to file:", drm.filename)
	return os.WriteFile(filepath.Clean(drm.filename), data, os.FileMode(drm.config.FileRights))
}

func (drm DefaultRegistryManager) Load(registry Registry) (Registry, error) {
	data, err := os.ReadFile(filepath.Clean(drm.filename))
	if err != nil {
		drm.log.Errorln("Error reading file", err)
		return registry, err
	}

	drm.log.Debugln("Registry loaded successfully")

	if err := json.Unmarshal(data, &registry); err != nil {
		drm.log.Errorln("Error deserializing registry", err)
		return registry, err
	}

	drm.log.Debugln("Registry deserialized successfully")

	return registry, nil
}
