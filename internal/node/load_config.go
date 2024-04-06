package node

import (
	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/internal/config"
)

func LoadConfig(yamlConfigFilePath *string, log *ipfsLog.ZapEventLogger) *config.Config {
	// Load the config file
	if *yamlConfigFilePath == "" {
		log.Panicln("Please provide a path to the yaml config file")
	}

	cfg, err := config.LoadConfig(log, *yamlConfigFilePath)
	if err != nil {
		log.Panicln("Error loading config file : ", err)
	}

	return cfg
}
