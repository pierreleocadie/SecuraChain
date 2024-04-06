package ecdsa

import (
	"crypto/sha256"
	"os"
	"testing"

	ipfsLog "github.com/ipfs/go-log/v2"
)

func TestECDSAKeyPairGeneration(t *testing.T) {
	t.Parallel()

	keyPair, err := NewECDSAKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate ECDSA key pair: %v", err)
	}

	if keyPair.PrivateKey() == nil {
		t.Error("Generated ECDSA key pair has nil private key")
	}

	if keyPair.PublicKey() == nil {
		t.Error("Generated ECDSA key pair has nil public key")
	}
}

func TestECDSASignAndVerify(t *testing.T) {
	t.Parallel()

	data := []byte("Hello, SecuraChain!")
	hash := sha256.Sum256(data)
	keyPair, _ := NewECDSAKeyPair()
	signature, err := keyPair.Sign(hash[:])

	if err != nil {
		t.Fatalf("Failed to sign data: %v", err)
	}

	if !VerifySignature(keyPair.PublicKey(), hash[:], signature) {
		t.Error("Failed to verify the signature of the data")
	}
}

func TestSaveAndLoadKeys(t *testing.T) {
	t.Parallel()

	log := ipfsLog.Logger("ecdsa_test")
	if err := ipfsLog.SetLogLevel("ecdsa_test", "DEBUG"); err != nil {
		log.Errorln("Failed to set log level : ", err)
	}

	// Paths for temporary testing
	privateKeyFile := "temp_private"
	publicKeyFile := "temp_public"
	storagePath := "../../temp"

	keyPair, _ := NewECDSAKeyPair()

	err := keyPair.SaveKeys(privateKeyFile, publicKeyFile, storagePath)

	if err != nil {
		t.Fatalf("Failed to save keys: %v", err)
	}

	loadedKeyPair, err := LoadKeys(log, privateKeyFile, publicKeyFile, storagePath)

	if err != nil {
		t.Fatalf("Failed to load keys: %v", err)
	}

	// Clean up temp files
	defer os.Remove(storagePath + "/" + privateKeyFile + ".pem")
	defer os.Remove(storagePath + "/" + publicKeyFile + ".pem")

	if loadedKeyPair.PrivateKey().D.Cmp(keyPair.PrivateKey().D) != 0 {
		t.Error("Loaded private key does not match saved private key")
	}

	if loadedKeyPair.PublicKey().X.Cmp(keyPair.PublicKey().X) != 0 || loadedKeyPair.PublicKey().Y.Cmp(keyPair.PublicKey().Y) != 0 {
		t.Error("Loaded public key does not match saved public key")
	}
}
