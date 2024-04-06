package ecdsa

import (
	"crypto/ecdsa"
	"crypto/rand"

	ipfsLog "github.com/ipfs/go-log/v2"
)

// Sign signs the given hash of the data using the private key of the ecdsaKeyPair.
// It returns the signature in ASN.1 format.
//
// Parameters:
//   - hash: The cryptographic hash of the data to be signed.
//
// Returns:
//   - A byte slice representing the ASN.1 signature.
//   - An error if signing fails.
func (keyPair *ecdsaKeyPair) Sign(log *ipfsLog.ZapEventLogger, hash []byte) ([]byte, error) {
	log.Debugln("Signing data")
	return ecdsa.SignASN1(rand.Reader, keyPair.privateKey, hash)
}

// VerifySignature verifies the given signature against the hash of the data using the public key of the ecdsaKeyPair.
//
// Parameters:
//   - hash: The cryptographic hash of the original data that was signed.
//   - signature: The ASN.1 signature to verify.
//
// Returns:
//   - A boolean indicating whether the verification was successful.
func VerifySignature(log *ipfsLog.ZapEventLogger, publicKey *ecdsa.PublicKey, hash, signature []byte) bool {
	log.Debugln("Verifying signature")
	return ecdsa.VerifyASN1(publicKey, hash, signature)
}
