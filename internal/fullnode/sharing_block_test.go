package fullnode_test

import (
	"bytes"
	"context"
	"os"
	"testing"

	ipfsLog "github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/internal/fullnode"
	"github.com/pierreleocadie/SecuraChain/internal/node"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

func TestPublishBlockToNetwork(t *testing.T) {
	t.Parallel()

	/*
	* CONFIGURATION
	 */

	log := ipfsLog.Logger("full-node-test")
	ctx := context.Background()

	cfg, err := config.LoadConfig("./config-test.yml")
	if err != nil {
		t.Errorf("Error loading config file : %v", err)
		os.Exit(1)
	}

	// Initialize the node
	host := node.Initialize(log, *cfg)
	defer host.Close()

	// Create a new pubsub instance
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		t.Errorf("Error creating pubsub: %v", err)
	}
	blockAnnouncementTopic, err := ps.Join("block-announcement")
	if err != nil {
		t.Errorf("Error joining topic: %v", err)
	}
	sub, err := blockAnnouncementTopic.Subscribe()
	if err != nil {
		t.Errorf("Error subscribing to topic: %v", err)
	}

	/*
	* FIRST BLOCK
	 */

	minerKeyPair, _ := ecdsa.NewECDSAKeyPair()  // Replace with actual key pair generation
	transactions := []transaction.Transaction{} // Empty transaction list for simplicity

	// Create a genesis block
	genesisBlock := block.NewBlock(transactions, nil, 1, minerKeyPair)

	// Check if the returned block is not nil
	if genesisBlock == nil {
		t.Errorf("NewBlock returned nil")
	}

	// Call the PublishBlockToNetwork function
	success, err := fullnode.PublishBlockToNetwork(ctx, genesisBlock, blockAnnouncementTopic)
	if err != nil {
		t.Errorf("Error publishing block to the network: %v", err)
	}

	// Start a goroutine to read messages from the topic
	message := make(chan []byte)
	go func() {
		for {
			msg, err := sub.Next(ctx)
			if err != nil {
				t.Errorf("Error reading message: %v", err)
			}
			message <- msg.Data
			// uncomment the following line to test that the message is not the same as the block
			// message <- append([]byte("toto"), msg.Data...)

		}
	}()

	response := <-message

	// Check if the block was successfully published
	if !success {
		t.Errorf("Block was not successfully published")
	}

	// Check if the message received is the same as the block
	blockBytes, err := genesisBlock.Serialize()
	if err != nil {
		t.Errorf("Error serializing block: %v", err)
	}
	if !bytes.Equal(response, blockBytes) {
		t.Errorf("Received message does not match the block")
	}
}
