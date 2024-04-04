package registry

import (
	"os"

	"github.com/ipfs/boxo/path"
	"github.com/ipfs/go-cid"
	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/pkg/utils"
)

// BlockData represents the data associated with a block stored with IPFS.
type BlockData struct {
	ID       uint32        `json:"ID"`
	Key      []byte        `json:"key"`
	BlockCid cid.Cid       `json:"cid"`
	Provider peer.AddrInfo `json:"provider"`
}

// BlockRegistry represents a collection of block records.
type BlockRegistry struct {
	Blocks []BlockData `json:"blocks"`
}

// AddBlockToRegistry adds a block and the data associated to the registry.
func AddBlockToRegistry(log *ipfsLog.ZapEventLogger, b *block.Block, config *config.Config, fileCid path.ImmutablePath, provider peer.AddrInfo) error {
	var blockRegistery BlockRegistry

	// Load existing registry if it exists
	blockRegistery, err := LoadRegistryFile[BlockRegistry](log, config, config.BlockRegistryPath)
	if err != nil && !os.IsNotExist(err) {
		log.Errorln("Error loading block registry:", err)
		return err
	}

	newData := BlockData{
		ID:       b.Header.Height,
		Key:      block.ComputeHash(b),
		BlockCid: fileCid.RootCid(),
		Provider: provider,
	}
	log.Debugln("New BlockData : ", newData)

	blockRegistery.Blocks = append(blockRegistery.Blocks, newData)

	// Save updated registry back to file
	log.Infoln("Block registry created or updated successfully")
	return SaveRegistryToFile(log, config, config.BlockRegistryPath, blockRegistery)
}

// ConvertToBlock reads the contents of the file at the given file path and converts it into a block.Block object.
func ConvertToBlock(log *ipfsLog.ZapEventLogger, filePath string) (*block.Block, error) {
	filePath, err := utils.SanitizePath(filePath)
	if err != nil {
		log.Errorf("Error sanitizing file patxh %v\n", err)
		return nil, err
	}
	data, err := os.ReadFile(filePath) //#nosec G304
	if err != nil {
		log.Errorf("Error reading file %v\n", err)
		return nil, err
	}

	log.Debugln("File converted to block successfully")
	return block.DeserializeBlock(data)
}
