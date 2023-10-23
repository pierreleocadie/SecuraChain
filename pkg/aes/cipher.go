package aes

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"os"

	"github.com/pierreleocadie/SecuraChain/pkg/utils"
)

// EncryptFile encrypts a file using AES encryption in CFB mode.
// It takes as input the paths for the input and output files.
// Returns an error if encryption fails.
func (aesKey *aesKey) EncryptFile(inputFilePath, outputFilePath string) error {
	// Sanitize the input file path.
	cleanInputFilePath, err := utils.SanitizePath(inputFilePath)
	if err != nil {
		return fmt.Errorf("could not sanitize input file path: %v", err)
	}

	// Open the input file.
	inputFile, err := os.Open(cleanInputFilePath) // #nosec G304
	if err != nil {
		return fmt.Errorf("could not open input file: %v", err)
	}
	defer inputFile.Close()

	// Sanitize the output file path.
	cleanOutputFilePath, err := utils.SanitizePath(outputFilePath)
	if err != nil {
		return fmt.Errorf("could not sanitize output file path: %v", err)
	}

	// Create the output file.
	outputFile, err := os.Create(cleanOutputFilePath) // #nosec G304
	if err != nil {
		return fmt.Errorf("could not create output file: %v", err)
	}
	defer outputFile.Close()

	// Initialize AES cipher block.
	block, err := aes.NewCipher(aesKey.key)
	if err != nil {
		return fmt.Errorf("could not create AES cipher block: %v", err)
	}

	// Generate a random Initialization Vector (IV).
	initializationVector := make([]byte, aes.BlockSize)
	if _, err := rand.Read(initializationVector); err != nil {
		return fmt.Errorf("could not generate random initialization vector: %v", err)
	}

	// Write the IV to the output file for future decryption.
	if _, err := outputFile.Write(initializationVector); err != nil {
		return fmt.Errorf("could not write initialization vector to output file: %v", err)
	}

	// Initialize AES cipher in CFB mode.
	stream := cipher.NewCFBEncrypter(block, initializationVector)

	// Initialize the writer for encrypted data.
	writer := &cipher.StreamWriter{S: stream, W: outputFile}

	// Perform the encryption.
	if _, err := io.Copy(writer, inputFile); err != nil {
		return fmt.Errorf("could not copy and encrypt file content: %v", err)
	}

	return nil
}

// DecryptFile decrypts a file that was encrypted using AES encryption in CFB mode.
// It takes as input the paths for the input and output files.
// Returns an error if decryption fails.
func (aesKey *aesKey) DecryptFile(inputFilePath, outputFilePath string) error {
	// Sanitize the input file path.
	cleanInputFilePath, err := utils.SanitizePath(inputFilePath)
	if err != nil {
		return fmt.Errorf("could not sanitize input file path: %v", err)
	}

	// Open the encrypted file.
	inputFile, err := os.Open(cleanInputFilePath) // #nosec G304
	if err != nil {
		return fmt.Errorf("could not open input file: %v", err)
	}
	defer inputFile.Close()

	// Sanitize the output file path.
	cleanOutputFilePath, err := utils.SanitizePath(outputFilePath)
	if err != nil {
		return fmt.Errorf("could not sanitize output file path: %v", err)
	}

	// Create the output file for decrypted data.
	outputFile, err := os.Create(cleanOutputFilePath) // #nosec G304
	if err != nil {
		return fmt.Errorf("could not create output file: %v", err)
	}
	defer outputFile.Close()

	// Initialize AES cipher block.
	block, err := aes.NewCipher(aesKey.key)
	if err != nil {
		return fmt.Errorf("could not create AES cipher block: %v", err)
	}

	// Read the Initialization Vector (IV) from the encrypted file.
	initializationVector := make([]byte, aes.BlockSize)
	if _, err := inputFile.Read(initializationVector); err != nil {
		return fmt.Errorf("could not read initialization vector from input file: %v", err)
	}

	// Initialize AES cipher in CFB mode for decryption.
	stream := cipher.NewCFBDecrypter(block, initializationVector)

	// Initialize the reader for decrypted data.
	reader := &cipher.StreamReader{S: stream, R: inputFile}

	// Perform the decryption.
	if _, err := io.Copy(outputFile, reader); err != nil {
		return fmt.Errorf("could not copy and decrypt file content: %v", err)
	}

	return nil
}
