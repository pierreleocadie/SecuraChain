package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/pierreleocadie/SecuraChain/internal"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fullNode, err := NewFullNode(ctx)
	if err != nil {
		log.Fatal("Failed to create a full node:", err)
	}

	// Subscribe to the miner's block topic
	if err := fullNode.SubscribeToMinerTopic("NewBlock"); err != nil {
		log.Fatal("Failed to subscribe to the miner's block topic:", err)
	}

	//create the database for transaction
	dbPath := "../../internal/securachain_db_fullnode"
	CreateDB(dbPath)

	// Run the full node
	go fullNode.Run(ctx)

	// Simulate the program running for some time
	select {
	case <-time.After(60 * time.Second):
		log.Println("Exiting the program after 60 seconds.")
		cancel()
		os.Exit(0)
	}
}

func NewFullNode(ctx context.Context) {
	panic("unimplemented")
}

func CreateDB(dbPath string) {
	db, err := internal.NewPebbleTransactionDB(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
}

//Enregistrer la transition
