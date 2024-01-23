package config

import (
	"os"
	"time"

	"github.com/pierreleocadie/SecuraChain/pkg/utils"
	"gopkg.in/yaml.v3"
)

type Config struct {
	// SecuraChain node
	UserAgent       string `yaml:"userAgent"`
	ProtocolVersion string `yaml:"protocolVersion"`

	// Connection manager
	LowWater  int `yaml:"lowWater"`
	HighWater int `yaml:"highWater"`

	// Network
	BootstrapPeers           []string      `yaml:"bootstrapPeers"`
	ListeningPort            uint          `yaml:"listeningPort"`
	DiscoveryRefreshInterval time.Duration `yaml:"discoveryRefreshInterval"`
	Ip4tcp                   string        `yaml:"ip4tcp"`
	Ip6tcp                   string        `yaml:"ip6tcp"`
	Ip4quic                  string        `yaml:"ip4quic"`
	Ip6quic                  string        `yaml:"ip6quic"`

	// PubSub
	RendezvousStringFlag           string `yaml:"rendezvousStringFlag"`
	ClientAnnouncementStringFlag   string `yaml:"clientAnnouncementStringFlag"`
	StorageNodeResponseStringFlag  string `yaml:"storageNodeResponseStringFlag"`
	BlockAnnouncementStringFlag    string `yaml:"blockAnnouncementStringFlag"`
	FullNodeAnnouncementStringFlag string `yaml:"fullNodeAnnouncementStringFlag"`

	// Embeded IPFS node
	FileRights               int    `yaml:"fileRights"`
	FileMetadataRegistryJson string `yaml:"fileMetadataRegistryJson"`
}

// Function to load the yaml config file
func LoadConfig(yamlConfigFilePath string) (*Config, error) {
	// Load the config file
	config := Config{}
	sanitizedPath, err := utils.SanitizePath(yamlConfigFilePath)
	if err != nil {
		return nil, err
	}
	configBytes, err := os.ReadFile(sanitizedPath)
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
