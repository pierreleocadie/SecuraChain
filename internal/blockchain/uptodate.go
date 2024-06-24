package blockchain

import (
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/consensus"
	"github.com/pierreleocadie/SecuraChain/internal/fullnode"
)

type UpToDateState struct {
	name       string
	blockchain *Blockchain
}

func (s *UpToDateState) HandleBlock(b block.Block) {
	s.blockchain.log.Debugln("State : UpToDateState")

	// 1 . Validation of the block
	if b.IsGenesisBlock() {
		s.blockchain.log.Debugln("Genesis block")
		if err := s.blockchain.BlockValidator.Validate(b, block.Block{}); err != nil {
			s.blockchain.log.Debugln("Genesis block is invalid")
		}
		s.blockchain.log.Debugln("Genesis block is valid")
	} else {
		if err := fullnode.PrevBlockStored(s.blockchain.log, b, s.blockchain.Database); err != nil {
			s.blockchain.log.Debugln("Error checking if previous block is stored : %s", err)

			s.blockchain.pendingBlocks = append(s.blockchain.pendingBlocks, b)
			s.blockchain.SetState(s.blockchain.SyncingState)
			s.blockchain.NotifyObservers()
			return
		}

		prevBlock, err := s.blockchain.Database.GetBlock(b.PrevBlock)
		if err != nil {
			s.blockchain.log.Debugln("Error getting the previous block : %s\n", err)
		}

		if err := s.blockchain.BlockValidator.Validate(b, prevBlock); err != nil {
			s.blockchain.log.Debugln("Block is invalid")
			return
		}
	}

	// 2 . Add the block to the blockchain
	if err := s.blockchain.Database.AddBlock(b); err != nil {
		s.blockchain.log.Debugln("Error adding the block to the blockchain : %s\n", err)
		return
	}

	// 3 . Verify the integrity of the blockchain
	if err := s.blockchain.Database.VerifyIntegrity(); err != nil {
		s.blockchain.SetState(s.blockchain.SyncingState)
		s.blockchain.NotifyObservers()
		return
	}

	// 4 . Update the file registry with the block transactions
	if err := s.blockchain.fileRegistry.UpdateFromBlock(b); err != nil {
		s.blockchain.log.Debugln("Error adding the block transactions to the registry")
	}

	// 5 . Send the block to IPFS
	fileCid, nodeIPFSAddrInfo, err := s.blockchain.ipfsNode.PublishBlock(b)
	if err != nil {
		s.blockchain.log.Debugln("Error publishing the block to IPFS")
	}

	// 6 . Update the block registry with the block
	if err := s.blockchain.blockRegistry.Add(b, fileCid, nodeIPFSAddrInfo); err != nil {
		s.blockchain.log.Errorln("Error adding the block metadata to the registry")
		return
	}

	// 7 . Stop the mining process if the block is received from the network while mining
	// have the same block height or greater than the block being mined
	if s.blockchain.StopMiningChan != nil {
		// Among the observers, get the observer that is a Miner struct
		// for _, observer := range s.blockchain.observers {
		// 	if miner, ok := observer.(miningnode.Miner); ok {
		// 		if b.Height >= miner.GetCurrentBlock().Height {
		// 			s.blockchain.StopMiningChan <- miningnode.StopMiningSignal{Stop: true, BlockReceived: b}
		// 		}
		// 	}
		// }
		s.blockchain.StopMiningChan <- consensus.StopMiningSignal{Stop: true, BlockReceived: b}
	}
}

func (s *UpToDateState) SyncBlockchain() {
	// No-op for this state
}

func (s *UpToDateState) PostSync() {
	// No-op for this state
}

func (s *UpToDateState) GetCurrentStateName() string {
	return s.name
}
