package block

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"

	ipfsLog "github.com/ipfs/go-log/v2"
)

// IntToHex converts an int64 to a byte array
func IntToHex(log *ipfsLog.ZapEventLogger, num int64) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	if err != nil {
		log.Errorln("Error converting int to hex")
		panic(err)
	}

	log.Debugln("IntToHex:", buff.Bytes())
	return buff.Bytes()
}

// calculateHash calculates the hash of a block
func ComputeHash(log *ipfsLog.ZapEventLogger, block *Block) []byte {
	headerBytes := bytes.Join(
		[][]byte{
			IntToHex(log, int64(block.Header.Version)),
			block.Header.PrevBlock,
			block.Header.MerkleRoot,
			IntToHex(log, block.Header.Timestamp),
			IntToHex(log, int64(block.Header.TargetBits)),
			IntToHex(log, int64(block.Header.Height)),
			IntToHex(log, int64(block.Header.Nonce)),
			block.Header.MinerAddr,
		},
		[]byte{},
	)
	hash := sha256.Sum256(headerBytes)

	log.Debugln("Block hash:", hash)
	return hash[:]
}
