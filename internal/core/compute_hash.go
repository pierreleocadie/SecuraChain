package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
)

// IntToHex converts an int64 to a byte array
func IntToHex(num int64) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	if err != nil {
		panic(err)
	}
	return buff.Bytes()
}

// calculateHash calculates the hash of a block
func ComputeHash(block *Block) []byte {
	headerBytes := bytes.Join(
		[][]byte{
			IntToHex(int64(block.Header.Version)),
			block.Header.PrevBlock,
			block.Header.MerkleRoot,
			IntToHex(block.Header.Timestamp),
			IntToHex(int64(block.Header.TargetBits)),
			IntToHex(int64(block.Header.Height)),
			IntToHex(int64(block.Header.Nonce)),
			block.Header.MinerAddr,
		},
		[]byte{},
	)
	hash := sha256.Sum256(headerBytes)
	return hash[:]
}
