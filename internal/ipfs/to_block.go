package ipfs

import (
	"os"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/pkg/utils"
)

// ConvertToBlock reads the contents of the file at the given file path and converts it into a block.Block object.
func ConvertBytesToBlock(log *ipfsLog.ZapEventLogger, filePath string) (block.Block, error) {
	filePath, err := utils.SanitizePath(filePath)
	if err != nil {
		log.Errorf("Error sanitizing file patxh %v\n", err)
		return block.Block{}, err
	}

	data, err := os.ReadFile(filePath) //#nosec G304
	if err != nil {
		log.Errorf("Error reading file %v\n", err)
		return block.Block{}, err
	}

	log.Debugln("File converted to block successfully")
	return block.DeserializeBlock(data)
}
