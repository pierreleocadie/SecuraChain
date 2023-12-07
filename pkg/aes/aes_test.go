package aes

import (
	"bytes"
	"encoding/base64"
	"os"
	"testing"
)

const (
	testStoragePath = "../../temp/"
	testFileName    = "test_key"
	testInputFile   = "test_input.txt"
	testOutputFile  = "test_output.txt"
	testDecryptFile = "test_decrypt.txt"
	testContent     = "This is a test content for encryption and decryption."
)

func TestNewAESKey(t *testing.T) {
	t.Parallel()

	key, err := NewAESKey()
	if err != nil {
		t.Fatalf("NewAESKey failed: %v", err)
	}

	if len(key.Key()) != KeySize {
		t.Fatalf("Key size is incorrect, got: %d, want: %d", len(key.Key()), KeySize)
	}
}

func TestAESKeyString(t *testing.T) {
	t.Parallel()

	key, _ := NewAESKey()
	encoded := key.String()
	raw, _ := base64.StdEncoding.DecodeString(encoded)

	if !bytes.Equal(raw, key.Key()) {
		t.Fatalf("Key string encoding/decoding failed")
	}
}

func TestSaveAndLoadKey(t *testing.T) {
	t.Parallel()

	key, _ := NewAESKey()

	// Save the key to a file.
	err := key.SaveKey(testFileName, testStoragePath)
	if err != nil {
		t.Fatalf("SaveKey failed: %v", err)
	}
	defer os.Remove(testStoragePath + testFileName + fileExtPEM)

	// Load the key from the file.
	loadedKey, err := LoadKey(testFileName, testStoragePath)
	if err != nil {
		t.Fatalf("LoadKey failed: %v", err)
	}

	if !bytes.Equal(key.Key(), loadedKey.Key()) {
		t.Fatalf("Keys do not match after save and load")
	}
}

func TestEncryptDecryptFile(t *testing.T) {
	t.Parallel()

	key, _ := NewAESKey()

	// Create a test file for encryption.
	err := os.WriteFile(testInputFile, []byte(testContent), 0600)
	if err != nil {
		t.Fatalf("Could not create test file: %v", err)
	}
	defer os.Remove(testInputFile)

	// Encrypt the test file.
	err = key.EncryptFile(testInputFile, testOutputFile)
	if err != nil {
		t.Fatalf("EncryptFile failed: %v", err)
	}
	defer os.Remove(testOutputFile)

	// Decrypt the test file.
	err = key.DecryptFile(testOutputFile, testDecryptFile)
	if err != nil {
		t.Fatalf("DecryptFile failed: %v", err)
	}
	defer os.Remove(testDecryptFile)

	// Read the decrypted content.
	decryptedContent, err := os.ReadFile(testDecryptFile)
	if err != nil {
		t.Fatalf("Could not read decrypted file: %v", err)
	}

	if string(decryptedContent) != testContent {
		t.Fatalf("Decrypted content does not match original content")
	}
}

func TestEncryptDecryptData(t *testing.T) {
	t.Parallel()

	key, _ := NewAESKey()

	// Encrypt the test content.
	encryptedContent, err := key.EncryptData([]byte(testContent))
	if err != nil {
		t.Fatalf("EncryptData failed: %v", err)
	}

	// Decrypt the test content.
	decryptedContent, err := key.DecryptData(encryptedContent)
	if err != nil {
		t.Fatalf("DecryptData failed: %v", err)
	}

	if string(decryptedContent) != testContent {
		t.Fatalf("Decrypted content does not match original content")
	}
}
