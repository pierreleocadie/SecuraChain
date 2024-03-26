package indexing

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
	require.NoError(t, err)

	// Call the function under test.
	err = AddFileToRegistry(log, cfg, addFileTransac)

	// Generate a fake AddFileTransaction with the same Owner.
	addFileTransacWithSameOwner1, err := transaction.GenFakeAddTransactionWithSameOwner()
	require.NoError(t, err)

	err = AddFileToRegistry(log, cfg, addFileTransacWithSameOwner1)

	// Generate a fake AddFileTransaction with the same Owner.
	addFileTransacWithSameOwner2, err := transaction.GenFakeAddTransactionWithSameOwner()
	require.NoError(t, err)

	err = AddFileToRegistry(log, cfg, addFileTransacWithSameOwner2)

	// Generate a fake AddFileTransaction with the same Owner.
	addFileTransacWithSameOwner3, err := transaction.GenFakeAddTransactionWithSameOwner()
	require.NoError(t, err)

	err = AddFileToRegistry(log, cfg, addFileTransacWithSameOwner3)

	// Assert that no error occurred.
	require.NoError(t, err)

}
