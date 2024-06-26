package blockchain

import (
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/fullnode"
)

type PostSyncState struct {
	name       string
	blockchain *Blockchain
}

func (s *PostSyncState) HandleBlock(block block.Block) {
	s.blockchain.pendingBlocks = append(s.blockchain.pendingBlocks, block)
	s.blockchain.log.Debugf("PostSyncState - Block %d received and added to the pending blocks - Pending blocks list length : %d", block.Height, len(s.blockchain.pendingBlocks))
}

func (s *PostSyncState) SyncBlockchain() {
	// No-op for this state
}

func (s *PostSyncState) PostSync() {
	s.blockchain.log.Debugln("State : PostSyncState")

	// 1. Sort the waiting list by height of the block
	sortedList := fullnode.SortBlockByHeight(s.blockchain.log, s.blockchain.pendingBlocks)

	for _, b := range sortedList {
		// 2 . Verify if the previous block is stored in the database
		if err := fullnode.PrevBlockStored(s.blockchain.log, b, s.blockchain.Database); err != nil {
			s.blockchain.log.Debugln("Error checking if previous block is stored : %s", err)

			s.blockchain.pendingBlocks = []block.Block{}
			s.blockchain.SetState(s.blockchain.SyncingState)
			return
		}

		// 3 . Validation of the block
		if b.IsGenesisBlock() {
			if err := s.blockchain.BlockValidator.Validate(b, block.Block{}); err != nil {
				s.blockchain.log.Debugln("Genesis block is invalid")
				continue
			}
			s.blockchain.log.Debugln("Genesis block is valid")
		} else {
			prevBlock, err := s.blockchain.Database.GetBlock(b.PrevBlock)
			if err != nil {
				s.blockchain.log.Debugln("Error getting the previous block : %s\n", err)
			}

			if err := s.blockchain.BlockValidator.Validate(b, prevBlock); err != nil {
				s.blockchain.log.Debugln("Block is invalid")
				continue
			}
			s.blockchain.log.Debugln(b.Height, " is valid")
		}

		// 4 . Add the block to the blockchain
		if err := s.blockchain.Database.AddBlock(b); err != nil {
			s.blockchain.log.Debugln("Error adding the block to the blockchain : %s\n", err)
			continue
		}

		if err := s.blockchain.Database.VerifyIntegrity(); err != nil {
			s.blockchain.pendingBlocks = []block.Block{}
			s.blockchain.SetState(s.blockchain.SyncingState)
			return
		}

		// 5 . Add the block transaction to the registry
		if err := s.blockchain.fileRegistry.UpdateFromBlock(b); err != nil {
			s.blockchain.log.Debugln("Error adding the block transactions to the registry")
		}

		// 6 . Send the block to IPFS
		fileCid, nodeIPFSAddrInfo, err := s.blockchain.ipfsNode.PublishBlock(b)
		if err != nil {
			s.blockchain.log.Debugln("Error publishing the block to IPFS")
		}

		// 7 . Update the block registry with the block
		if err := s.blockchain.blockRegistry.Add(b, fileCid, nodeIPFSAddrInfo); err != nil {
			s.blockchain.log.Errorln("Error adding the block metadata to the registry")
		}
	}
	s.blockchain.pendingBlocks = []block.Block{}
	s.blockchain.log.Debugln("Post-synchronization done")
	// 6 . Change the state of the node
	s.blockchain.SetState(s.blockchain.UpToDateState)
}

func (s *PostSyncState) GetCurrentStateName() string {
	return s.name
}
