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
	BootstrapPeers                        []string      `yaml:"bootstrapPeers"`
	ListeningPort                         uint          `yaml:"listeningPort"`
	DiscoveryRefreshInterval              time.Duration `yaml:"discoveryRefreshInterval"`
	AutorelayWithBackoffInterval          time.Duration `yaml:"autorelayWithBackoffInterval"`
	AutorelayWithMinCallbackInterval      time.Duration `yaml:"autorelayWithMinCallbackInterval"`
	KeepRelayConnectionAliveInterval      time.Duration `yaml:"keepRelayConnectionAliveInterval"`
	MaxRelayedConnectionDuration          time.Duration `yaml:"maxRelayedConnectionDuration"`
	MaxDataRelayed                        int64         `yaml:"maxDataRelayed"`
	RelayReservationTTL                   time.Duration `yaml:"relayReservationTTL"`
	MaxRelayReservations                  int           `yaml:"maxRelayReservations"`
	MaxRelayCircuits                      int           `yaml:"maxRelayCircuits"`
	MaxRelayedConnectionBufferSize        int           `yaml:"maxRelayedConnectionBufferSize"`
	MaxRelayReservationsPerPeer           int           `yaml:"maxRelayReservationsPerPeer"`
	MaxRelayReservationsPerIP             int           `yaml:"maxRelayReservationsPerIP"`
	MaxRelayReservationsPerASN            int           `yaml:"maxRelayReservationsPerASN"`
	IpfsProvidersDiscoveryRefreshInterval time.Duration `yaml:"ipfsProvidersDiscoveryRefreshInterval"`
	IP4tcp                                string        `yaml:"ip4tcp"`
	IP6tcp                                string        `yaml:"ip6tcp"`
	IP4quic                               string        `yaml:"ip4quic"`
	IP6quic                               string        `yaml:"ip6quic"`

	// PubSub
	RendezvousStringFlag               string `yaml:"rendezvousStringFlag"`
	ClientAnnouncementStringFlag       string `yaml:"clientAnnouncementStringFlag"`
	StorageNodeResponseStringFlag      string `yaml:"storageNodeResponseStringFlag"`
	KeepRelayConnectionAliveStringFlag string `yaml:"keepRelayConnectionAliveStringFlag"`
	BlockAnnouncementStringFlag        string `yaml:"blockAnnouncementStringFlag"`
	FullNodeAnnouncementStringFlag     string `yaml:"fullNodeAnnouncementStringFlag"`
	BlacklistStringFlag                string `yaml:"blacklistStringFlag"`
	AskingBlockchainStringFlag         string `yaml:"askingBlockchainStringFlag"`
	ReceiveBlockchainStringFlag        string `yaml:"receiveBlockchainStringFlag"`
	AskMyFilesStringFlag               string `yaml:"askMyFilesStringFlag"`
	SendFilesStringFlag                string `yaml:"sendFilesStringFlag"`

	// Embedded IPFS node
	FileRights               int    `yaml:"fileRights"`
	FileMetadataRegistryJSON string `yaml:"fileMetadataRegistryJson"`
	MemorySpace              uint   `yaml:"memorySpace"`
	BlockRegistryPath        string `yaml:"blockRegistryPath"`
	IndexingRegistryPath     string `yaml:"indexingRegistryPath"`
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
