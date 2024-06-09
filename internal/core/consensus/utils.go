package consensus

import (
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/peer"
)

// isValidTimestampOrder checks if the provided timestamps are in the correct chronological order
func isValidTimestampOrder(timestamps ...int64) bool {
	for i := 0; i < len(timestamps)-1; i++ {
		if timestamps[i] >= timestamps[i+1] {
			return false
		}
	}
	return true
}

func isValidCID(c cid.Cid) bool {
	// Check if the CID is defined. An undefined CID is invalid.
	if !c.Defined() {
		return false
	}

	if c.Version() != 0 && c.Version() != 1 {
		return false
	}

	return true
}

func isValidPeerID(p peer.ID) bool {
	// Check if the peer.ID is valid
	if _, err := peer.Decode(p.String()); err != nil {
		return false
	}

	return true
}
