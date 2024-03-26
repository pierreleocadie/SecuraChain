package fullnode

import (
	"bytes"
	"context"

	ipfsLog "github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/registry"
)

func SearchMyFiles(log *ipfsLog.ZapEventLogger, config *config.Config, myAddress []byte) []registry.FileRegistry {
	log.Debugln("Searching for files of owner : ", myAddress)
	var myFiles []registry.FileRegistry

	// Get the indexing registry
	r, err := registry.LoadRegistryFile[registry.IndexingRegistry](log, config.IndexingRegistryPath)
	if err != nil {
		log.Errorln("Error loading the indexing registry : ", err)
	}

	for _, ownerFiles := range r.IndexingFiles {
		if bytes.Equal(ownerFiles.OwnerAddress, myAddress) {
			myFiles = append(myFiles, ownerFiles.Files...)
			log.Debugln("Files found")
			return myFiles
		}
	}

	log.Debugln("No files found")
	return myFiles
}

// SendOwnersFiles sends the files owned by a specific address to the given owner topic.
func SendOwnersFiles(log *ipfsLog.ZapEventLogger, ctx context.Context, config *config.Config, myAddress []byte, owner *pubsub.Topic) bool {
	myFiles := SearchMyFiles(log, config, myAddress)

	myFilesBytes, err := registry.SerializeRegistry(log, myFiles)
	if err != nil {
		log.Errorln("Error serializing my files : ", err)
		return false
	}

	if err := owner.Publish(ctx, myFilesBytes); err != nil {
		log.Errorln("Error publishing indexing registry : ", err)
		return false
	}

	log.Debugln("Files of owner sent")
	return true
}
