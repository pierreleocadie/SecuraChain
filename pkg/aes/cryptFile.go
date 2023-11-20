package aes

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// EncryptFile encrypts a file using AES encryption.
func EncryptFile(inputFilePath, outputFilePath string, key []byte) error {
	// Construct and clean the path
	filePath := path.Join(storagePath, filename+fileExtPEM)
	cleanFilePath := filepath.Clean(filePath)

	// Ensure that the cleaned path still has the intended base directory
	if !strings.HasPrefix(cleanFilePath, storagePath) {
		return nil, fmt.Errorf("potential path traversal attempt detected")
	}
	// Read file
	inputFile, err := os.Open((inputFilePath))
	if err != nil {
		return err
	}
	defer inputFile.Close()

	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	// Create an AES cipher block using the provided key
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	// Generate a random initialization vector (IV)
	initializationVector := make([]byte, aes.BlockSize)
	if _, err := rand.Read(initializationVector); err != nil {
		return err
	}

	// Write the IV to the output file (it will be needed for decryption)
	if _, err := outputFile.Write(initializationVector); err != nil {
		return err
	}

	// Create a cipher block mode for AES in CFB mode
	stream := cipher.NewCFBEncrypter(block, initializationVector)

	// Create a writer with the encryption stream
	writer := &cipher.StreamWriter{S: stream, W: outputFile}

	// Copy and encrypt the file content
	if _, err := io.Copy(writer, inputFile); err != nil {
		return err
	}

	return nil
}

// DecryptFile decrypts a file that was encrypted using AES encryption.
func DecryptFile(inputFilePath, outputFilePath string, key []byte) error {
	// Open the input and output files
	inputFile, err := os.Open(filepath.Clean(inputFilePath))
	if err != nil {
		return err
	}
	defer inputFile.Close()

	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	// Create an AES cipher block using the provided key
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	// Read the initialization vector (IV) from the input file
	initializationVector := make([]byte, aes.BlockSize)
	if _, err := inputFile.Read(initializationVector); err != nil {
		return err
	}

	// Create a cipher block mode for AES in CFB mode
	stream := cipher.NewCFBDecrypter(block, initializationVector)

	// Create a reader with the decryption stream
	reader := &cipher.StreamReader{S: stream, R: inputFile}

	// Copy and decrypt the file content
	if _, err := io.Copy(outputFile, reader); err != nil {
		return err
	}

	return nil
}
