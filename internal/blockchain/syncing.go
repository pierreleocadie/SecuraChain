package blockchain

import (
	"slices"

	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/fullnode"
	blockregistry "github.com/pierreleocadie/SecuraChain/internal/registry/block_registry"
)

type SyncingState struct {
	name       string
	blockchain *Blockchain
}

func (s *SyncingState) HandleBlock(block block.Block) {
	s.blockchain.pendingBlocks = append(s.blockchain.pendingBlocks, block)
	s.blockchain.log.Debugf("SyncingState - Block %d received and added to the pending blocks - Pending blocks list length : %d", block.Height, len(s.blockchain.pendingBlocks))
}

func (s *SyncingState) SyncBlockchain() {
	s.blockchain.log.Debugln("State : SyncingState")

	//1 . Ask for a registry of the blockchain
	registryBytes, senderID, err := blockregistry.AskForBlockchainRegistry(s.blockchain.log, s.blockchain.Ctx, s.blockchain.pubsubHub.AskingBlockchainTopic, s.blockchain.pubsubHub.ReceiveBlockchainSub)
	if err != nil {
		s.blockchain.log.Debugln("Error asking the blockchain registry : %s\n", err)
		return
	}

	// 1.1 Check if the sender is blacklisted
	if slices.Contains(s.blockchain.nodeBlacklist, senderID) {
		s.blockchain.log.Debugln("Node blacklisted")
		return
	}

	s.blockchain.log.Debugln("Node not blacklisted")

	// 1.2 black list the sender
	s.blockchain.nodeBlacklist = append(s.blockchain.nodeBlacklist, senderID)
	s.blockchain.log.Debugln("Node added to the black list")

	// 1.3 Convert the bytes to a block registry
	r, err := blockregistry.DeserializeBlockRegistry[*blockregistry.DefaultBlockRegistry](registryBytes)
	if err != nil {
		s.blockchain.log.Errorln("Error converting bytes to block registry : ", err)
	}
	s.blockchain.log.Debugln("Registry converted to BlockRegistry : ", r)

	// 2 . Get the missing blocks
	missingBlocks := fullnode.GetMissingBlocks(s.blockchain.log, r, s.blockchain.Database)

	// 2bis . Dowlnoad the missing blocks
	downloadedBlocks, err := fullnode.DownloadMissingBlocks(s.blockchain.log, s.blockchain.Ctx, s.blockchain.ipfsNode, missingBlocks)
	if err != nil {
		s.blockchain.log.Debugln("Error downloading missing blocks : %s\n", err)
		return
	}

	// 3 . Valid the downloaded blocks
	for _, b := range downloadedBlocks {
		if b.IsGenesisBlock() {
			if err := s.blockchain.BlockValidator.Validate(b, block.Block{}); err != nil {
				s.blockchain.log.Debugln("Genesis block is invalid")
				return
			}
			s.blockchain.log.Debugln("Genesis block is valid")
		} else {
			prevBlock, err := s.blockchain.Database.GetBlock(b.PrevBlock)
			if err != nil {
				s.blockchain.log.Debugln("Error getting the previous block : %s\n", err)
			}
			if err := s.blockchain.BlockValidator.Validate(b, prevBlock); err != nil {
				s.blockchain.log.Debugln("Block is invalid")
				return
			}
			s.blockchain.log.Debugln(b.Height, " is valid")
		}

		// 4 . Add the block to the blockchain
		if err := s.blockchain.Database.AddBlock(b); err != nil {
			s.blockchain.log.Debugln("Error adding the block to the blockchain : %s\n", err)
			return
		}

		// 5 . Verify the integrity of the blockchain
		if err := s.blockchain.Database.VerifyIntegrity(); err != nil {
			s.blockchain.log.Debugln("Blockchain is not verified")
			return
		}

		// 6 . Add the block transaction to the registry
		if err := s.blockchain.fileRegistry.UpdateFromBlock(b); err != nil {
			s.blockchain.log.Debugln("Error adding the block transactions to the registry")
		}

		// 7 . Send the block to IPFS
		fileCid, nodeIPFSAddrInfo, err := s.blockchain.ipfsNode.PublishBlock(b)
		if err != nil {
			s.blockchain.log.Debugln("Error publishing the block to IPFS")
		}

		// 8 . Update the block registry with the block
		if err := s.blockchain.blockRegistry.Add(b, fileCid, nodeIPFSAddrInfo); err != nil {
			s.blockchain.log.Errorln("Error adding the block metadata to the registry")
			return
		}
	}

	// 7 . Clear the black list
	s.blockchain.nodeBlacklist = []string{}

	s.blockchain.log.Debugln("Blockchain synchronized with the network")

	// 8 . Change the state of the node
	s.blockchain.SetState(s.blockchain.PostSyncState)
}

func (s *SyncingState) PostSync() {
	// No-op for this state
}

func (s *SyncingState) GetCurrentStateName() string {
	return s.name
}
