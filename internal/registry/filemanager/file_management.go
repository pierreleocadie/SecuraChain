package filemanager

import (
	"context"

	ipfsLog "github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/registry"
)

// searchMyFiles searches for files owned by a specific user (by his publicKey).
func searchMyFiles(log *ipfsLog.ZapEventLogger, config *config.Config, myPublicKey string) []registry.FileRegistry {
	log.Debugln("Searching for files of owner : ", myPublicKey)
	var myFiles []registry.FileRegistry

	// Get the indexing registry
	r, err := registry.LoadRegistryFile[registry.IndexingRegistry](log, config, config.IndexingRegistryPath)
	if err != nil {
		log.Errorln("Error loading the indexing registry : ", err)
	}

	for owner, filesRegistry := range r.IndexingFiles {
		if owner == myPublicKey {
			myFiles = append(myFiles, filesRegistry...)
			log.Debugln("Files found")
			return myFiles
		}
	}

	log.Debugln("No files found")
	return myFiles
}

// SendOwnersFiles sends the files owned by a specific specific user (by his publicKey).
func SendOwnersFiles(log *ipfsLog.ZapEventLogger, ctx context.Context, config *config.Config, myPublicKey string, owner *pubsub.Topic) bool {
	myFiles := searchMyFiles(log, config, myPublicKey)

	// myFilesBytes, err := registry.SerializeRegistry(log, myFiles)
	// if err != nil {
	// 	log.Errorln("Error serializing my files : ", err)
	// 	return false
	// }

	myFilesBytes, err := registry.SerializeRegistry(log, registry.RegistryMessage{OwnerPublicKey: myPublicKey, Registry: myFiles})
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
