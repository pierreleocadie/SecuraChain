package block

import (
	"reflect"
	"testing"

	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

func TestBlock_Serialize(t *testing.T) {
	t.Parallel()

	minerKeyPair, _ := ecdsa.NewECDSAKeyPair() // Replace with actual key pair generation
	transactions := []transaction.Transaction{}

	for i := 0; i < 3; i++ {
		fakeAddTransaction, err := transaction.GenFakeAddTransaction()
		if err != nil {
			t.Errorf("Failed to generate fake transaction: %s", err)
		}
		fakeDeleteTransaction, err := transaction.GenFakeDeleteTransaction()
		if err != nil {
			t.Errorf("Failed to generate fake transaction: %s", err)
		}
		transactions = append(transactions, fakeAddTransaction, fakeDeleteTransaction)
	}

	block := NewBlock(transactions, nil, 1, minerKeyPair)
	serializedBlock, err := block.Serialize()
	if err != nil {
		t.Errorf("Failed to serialize block: %s", err)
	}

	deserializedBlock, err := DeserializeBlock(serializedBlock)
	if err != nil {
		t.Errorf("Failed to deserialize block: %s", err)
	}

	if !reflect.DeepEqual(block, deserializedBlock) {
		t.Errorf("Deserialized block does not match original block")
	}
}
