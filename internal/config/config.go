package config

import (
	"os"
	"time"

	"github.com/pierreleocadie/SecuraChain/pkg/utils"
	"gopkg.in/yaml.v3"
)

type Config struct {
	// GUI
	WindowWidth  float32 `yaml:"windowWidth"`
	WindowHeight float32 `yaml:"windowHeight"`

	// SecuraChain node
	UserAgent       string `yaml:"userAgent"`
	ProtocolVersion string `yaml:"protocolVersion"`

	// ECDSA and AES keys path
	ECDSAKeyPairPath string `yaml:"ecdsaKeyPairPath"`
	AESKeyPath       string `yaml:"aesKeyPath"`

	// Connection manager
	LowWater  int `yaml:"lowWater"`
	HighWater int `yaml:"highWater"`

	// Network
	BootstrapPeers           []string      `yaml:"bootstrapPeers"`
	ListeningPort            uint          `yaml:"listeningPort"`
	DiscoveryRefreshInterval time.Duration `yaml:"discoveryRefreshInterval"`
	IP4tcp                   string        `yaml:"ip4tcp"`
	IP6tcp                   string        `yaml:"ip6tcp"`
	IP4quic                  string        `yaml:"ip4quic"`
	IP6quic                  string        `yaml:"ip6quic"`

	// PubSub
	RendezvousStringFlag          string `yaml:"rendezvousStringFlag"`
	ClientAnnouncementStringFlag  string `yaml:"clientAnnouncementStringFlag"`
	StorageNodeResponseStringFlag string `yaml:"storageNodeResponseStringFlag"`

	// Embedded IPFS node
	FileRights               int    `yaml:"fileRights"`
	FileMetadataRegistryJSON string `yaml:"fileMetadataRegistryJson"`
	MemorySpace              uint   `yaml:"memorySpace"`
}

// Function to load the yaml config file
func LoadConfig(yamlConfigFilePath string) (*Config, error) {
	// Load the config file
	config := Config{}
	sanitizedPath, err := utils.SanitizePath(yamlConfigFilePath)
	if err != nil {
		return nil, err
	}

	configBytes, err := os.ReadFile(sanitizedPath) // #nosec G304
	if err != nil {
		return nil, err
	}

	// Unmarshal the config file
	err = yaml.Unmarshal(configBytes, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
