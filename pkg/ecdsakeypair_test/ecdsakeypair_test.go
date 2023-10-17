package ecdsakeypair_test

import (
	"crypto/sha256"
	"os"
	"testing"

	"github.com/pierreleocadie/SecuraChain/pkg/ecdsakeypair"
)

func TestECDSAKeyPairGeneration(t *testing.T) {
	t.Parallel()

	keyPair, err := ecdsakeypair.NewECDSAKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate ECDSA key pair: %v", err)
	}

	if keyPair.GetPrivateKey() == nil {
		t.Error("Generated ECDSA key pair has nil private key")
	}

	if keyPair.GetPublicKey() == nil {
		t.Error("Generated ECDSA key pair has nil public key")
	}
}

func TestECDSASignAndVerify(t *testing.T) {
	t.Parallel()

	data := []byte("Hello, SecuraChain!")
	hash := sha256.Sum256(data)
	keyPair, _ := ecdsakeypair.NewECDSAKeyPair()
	signature, err := keyPair.Sign(hash[:])

	if err != nil {
		t.Fatalf("Failed to sign data: %v", err)
	}

	if !keyPair.VerifySignature(hash[:], signature) {
		t.Error("Failed to verify the signature of the data")
	}
}

func TestSaveAndLoadKeys(t *testing.T) {
	t.Parallel()

	// Paths for temporary testing
	privateKeyFile := "temp_private"
	publicKeyFile := "temp_public"
	storagePath := "../../temp"

	keyPair, _ := ecdsakeypair.NewECDSAKeyPair()

	err := keyPair.SaveKeys(privateKeyFile, publicKeyFile, storagePath)

	if err != nil {
		t.Fatalf("Failed to save keys: %v", err)
	}

	loadedKeyPair, err := ecdsakeypair.LoadKeys(privateKeyFile, publicKeyFile, storagePath)

	if err != nil {
		t.Fatalf("Failed to load keys: %v", err)
	}

	// Clean up temp files
	os.Remove(privateKeyFile)
	os.Remove(publicKeyFile)

	if loadedKeyPair.GetPrivateKey().D.Cmp(keyPair.GetPrivateKey().D) != 0 {
		t.Error("Loaded private key does not match saved private key")
	}

	if loadedKeyPair.GetPublicKey().X.Cmp(keyPair.GetPublicKey().X) != 0 || loadedKeyPair.GetPublicKey().Y.Cmp(keyPair.GetPublicKey().Y) != 0 {
		t.Error("Loaded public key does not match saved public key")
	}
}
