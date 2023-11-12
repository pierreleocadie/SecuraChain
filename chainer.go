package main

import (
	"encoding/json"
	"log"
)

type Chain struct {
	ChainVersion int64   `json:"chainVersion"`
	Chain        []Block `json:"chain"`
}

func addBlockToChain(block []byte, chain *Chain) {
	// Convert the block to a struct
	blockStruct := Block{}
	err := json.Unmarshal(block, &blockStruct)
	if err != nil {
		log.Printf("Error unmarshalling block: %v", err)
		return
	}
	// Append the block to the chain
	chain.Chain = append(chain.Chain, blockStruct)
	// Increment the chain version
	chain.ChainVersion++
}
