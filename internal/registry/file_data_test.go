package registry

import (
	"os"
	"testing"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/stretchr/testify/require"

	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
)

func TestAddFileToRegistryWhenOwnerDontExists(t *testing.T) {
	log := ipfsLog.Logger("test")

	cfg, err := config.LoadConfig("/Users/jordandohou/Desktop/SecuraChain/cmd/securachain-full-node/config.yml")
	if err != nil {
		log.Errorln("Error loading config file : ", err)
		os.Exit(1)
	}

	// Generate a fake AddFileTransaction.
	addFileTransac, err := transaction.GenFakeAddTransaction()
	if err != nil {
		log.Errorln("Error generating fake AddFileTransaction : ", err)
	}

	// Call the function under test.
	err = AddFileToRegistry(log, cfg, addFileTransac)
	if err != nil {
		log.Errorln("Error adding file to registry : ", err)
	}

	// Generate a fake AddFileTransaction with the same Owner.
	addFileTransacWithSameOwner1, err := transaction.GenFakeAddTransactionWithSameOwner()
	if err != nil {
		log.Errorln("Error generating fake AddFileTransaction : ", err)
	}

	err = AddFileToRegistry(log, cfg, addFileTransacWithSameOwner1)
	if err != nil {
		log.Errorln("Error adding file to registry : ", err)
	}

	// Generate a fake AddFileTransaction with the same Owner.
	addFileTransacWithSameOwner2, err := transaction.GenFakeAddTransactionWithSameOwner()
	if err != nil {
		log.Errorln("Error generating fake AddFileTransaction : ", err)
	}

	err = AddFileToRegistry(log, cfg, addFileTransacWithSameOwner2)
	if err != nil {
		log.Errorln("Error adding file to registry : ", err)
	}

	// Generate a fake AddFileTransaction with the same Owner.
	addFileTransacWithSameOwner3, err := transaction.GenFakeAddTransactionWithSameOwner()
	if err != nil {
		log.Errorln("Error generating fake AddFileTransaction : ", err)
	}

	err = AddFileToRegistry(log, cfg, addFileTransacWithSameOwner3)
	if err != nil {
		log.Errorln("Error adding file to registry : ", err)
	}

	// Assert that no error occurred.
	require.NoError(t, err)

}
