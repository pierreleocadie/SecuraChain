package block

import (
	"reflect"
	"testing"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

func TestBlock_Serialize(t *testing.T) {
	t.Parallel()

	log := ipfsLog.Logger("full-node")
	if err := ipfsLog.SetLogLevel("full-node", "DEBUG"); err != nil {
		log.Errorln("Failed to set log level : ", err)
	}

	minerKeyPair, _ := ecdsa.NewECDSAKeyPair() // Replace with actual key pair generation
	transactions := []transaction.Transaction{}

	for i := 0; i < 3; i++ {
		fakeAddTransaction, err := transaction.GenFakeAddTransaction(log)
		if err != nil {
			t.Errorf("Failed to generate fake transaction: %s", err)
		}
		fakeDeleteTransaction, err := transaction.GenFakeDeleteTransaction(log)
		if err != nil {
			t.Errorf("Failed to generate fake transaction: %s", err)
		}
		transactions = append(transactions, fakeAddTransaction, fakeDeleteTransaction)
	}

	block := NewBlock(log, transactions, nil, 1, minerKeyPair)
	serializedBlock, err := block.Serialize(log)
	if err != nil {
		t.Errorf("Failed to serialize block: %s", err)
	}

	deserializedBlock, err := DeserializeBlock(log, serializedBlock)
	if err != nil {
		t.Errorf("Failed to deserialize block: %s", err)
	}

	if !reflect.DeepEqual(block, deserializedBlock) {
		t.Errorf("Deserialized block does not match original block")
	}
}
