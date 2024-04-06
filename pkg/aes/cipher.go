package aes

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"os"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/pkg/utils"
)

// EncryptFile encrypts a file using AES encryption in CFB mode.
// It takes as input the paths for the input and output files.
// Returns an error if encryption fails.
func (aesKey *aesKey) EncryptFile(log *ipfsLog.ZapEventLogger, inputFilePath, outputFilePath string) error {
	// Sanitize the input file path.
	cleanInputFilePath, err := utils.SanitizePath(log, inputFilePath)
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
	cleanOutputFilePath, err := utils.SanitizePath(log, outputFilePath)
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
func (aesKey *aesKey) DecryptFile(log *ipfsLog.ZapEventLogger, inputFilePath, outputFilePath string) error {
	// Sanitize the input file path.
	cleanInputFilePath, err := utils.SanitizePath(log, inputFilePath)
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
	cleanOutputFilePath, err := utils.SanitizePath(log, outputFilePath)
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

// EncryptData encrypts data using AES encryption in CFB mode.
// It takes as input the data to encrypt.
// Returns the encrypted data and an error if encryption fails.
func (aesKey *aesKey) EncryptData(data []byte) ([]byte, error) {
	// Initialize AES cipher block.
	block, err := aes.NewCipher(aesKey.key)
	if err != nil {
		return nil, fmt.Errorf("could not create AES cipher block: %v", err)
	}

	// Generate a random Initialization Vector (IV).
	initializationVector := make([]byte, aes.BlockSize)
	if _, err := rand.Read(initializationVector); err != nil {
		return nil, fmt.Errorf("could not generate random initialization vector: %v", err)
	}

	// Initialize AES cipher in CFB mode.
	stream := cipher.NewCFBEncrypter(block, initializationVector)

	// Encrypt the data.
	encryptedData := make([]byte, len(data))
	stream.XORKeyStream(encryptedData, data)

	// Prepend the IV to the encrypted data.
	encryptedData = append(initializationVector, encryptedData...)

	return encryptedData, nil
}

// DecryptData decrypts data that was encrypted using AES encryption in CFB mode.
// It takes as input the data to decrypt.
// Returns the decrypted data and an error if decryption fails.
func (aesKey *aesKey) DecryptData(data []byte) ([]byte, error) {
	// Initialize AES cipher block.
	block, err := aes.NewCipher(aesKey.key)
	if err != nil {
		return nil, fmt.Errorf("could not create AES cipher block: %v", err)
	}

	// Read the Initialization Vector (IV) from the encrypted data.
	initializationVector := data[:aes.BlockSize]

	// Initialize AES cipher in CFB mode for decryption.
	stream := cipher.NewCFBDecrypter(block, initializationVector)

	// Decrypt the data.
	decryptedData := make([]byte, len(data)-aes.BlockSize)
	stream.XORKeyStream(decryptedData, data[aes.BlockSize:])

	return decryptedData, nil
}
