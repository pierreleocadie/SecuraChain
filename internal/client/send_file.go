package client

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	iface "github.com/ipfs/boxo/coreiface"
	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/ipfs/kubo/core"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/internal/ipfs"
	"github.com/pierreleocadie/SecuraChain/pkg/aes"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
	"github.com/pierreleocadie/SecuraChain/pkg/utils"
)

func SendFile(ctx context.Context, selectedFile string,
	ecdsaKeyPair *ecdsa.KeyPair, aesKey *aes.Key,
	nodeIpfs *core.IpfsNode, ipfsApi iface.CoreAPI,
	clientAnnouncementChan chan *transaction.ClientAnnouncement,
	log *ipfsLog.ZapEventLogger) error {

	// -1. Check if the ECDSA key pair and the AES key are loaded
	err := checkKeys(ecdsaKeyPair, aesKey)
	if err != nil {
		log.Errorf("failed to check keys: %s", err)
		return fmt.Errorf("failed to check keys: %s", err)
	}

	// O. Get the original file name and extension
	filename, extension, err := getSelectedFileInfo(selectedFile)
	if err != nil {
		log.Errorf("failed to get selected file info: %s", err)
		return fmt.Errorf("failed to get selected file info: %s", err)
	}

	// 1. Encrypt the filename and file extension with AES
	encryptedFilename, encryptedExtension, err := encryptFilenameAndExtension(filename, extension, *aesKey)
	if err != nil {
		log.Errorf("failed to encrypt filename and extension: %s", err)
		return fmt.Errorf("failed to encrypt filename and extension: %s", err)
	}

	// 2. Encrypt the file with AES
	encryptedFilePath, err := encryptFile(selectedFile, *aesKey, encryptedFilename, encryptedExtension, extension)
	if err != nil {
		log.Errorf("failed to encrypt file: %s", err)
		return fmt.Errorf("failed to encrypt file: %s", err)
	}

	// 3. Compute the checksum of the encrypted file
	encryptedFileChecksum, err := utils.ComputeFileChecksum(encryptedFilePath)
	if err != nil {
		log.Errorf("failed to compute checksum of encrypted file: %s", err)
		return fmt.Errorf("failed to compute checksum of encrypted file: %s", err)
	}

	// 4. Get the size of the encrypted file
	fileStat, err := os.Stat(encryptedFilePath)
	if err != nil {
		log.Errorf("failed to get file stat: %s", err)
		return fmt.Errorf("failed to get file stat: %s", err)
	}
	fileSize := fileStat.Size()

	// 5. Add the encrypted file to IPFS
	encryptedFileCid, err := ipfs.AddFile(ctx, nodeIpfs, ipfsApi, encryptedFilePath)
	if err != nil {
		log.Errorf("failed to add file to IPFS: %s", err)
		return fmt.Errorf("failed to add file to IPFS: %s", err)
	}

	// 6. Create a new ClientAnnouncement
	clientAnnouncement := transaction.NewClientAnnouncement(
		*ecdsaKeyPair,
		encryptedFileCid.RootCid(),
		encryptedFilename,
		encryptedExtension,
		uint64(fileSize),
		encryptedFileChecksum,
	)

	// 7. Send the ClientAnnouncement to the channel
	clientAnnouncementChan <- clientAnnouncement

	return nil
}

func checkKeys(ecdsaKeyPair *ecdsa.KeyPair, aesKey *aes.Key) error {
	if ecdsaKeyPair == nil || aesKey == nil {
		return fmt.Errorf("please load the keys ECDSA key pair and AES key before sending a file")
	}
	return nil
}

func getSelectedFileInfo(selectedFile string) (string, string, error) {
	if selectedFile == "" {
		return "", "", fmt.Errorf("please select a file")
	}

	filename, _, extension, err := utils.FileInfo(selectedFile)
	if err != nil {
		return "", "", fmt.Errorf("failed to get file info: %s", err)
	}

	filename = strings.Split(filename, ".")[0]
	return filename, extension, nil
}

func encryptFilenameAndExtension(filename, extension string, aesKey aes.Key) ([]byte, []byte, error) {
	encryptedFilename, err := aesKey.EncryptData([]byte(filename))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to encrypt filename: %s", err)
	}

	encryptedExtension, err := aesKey.EncryptData([]byte(extension))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to encrypt extension: %s", err)
	}

	return encryptedFilename, encryptedExtension, nil
}

func encryptFile(selectedFile string, aesKey aes.Key, encryptedFilename []byte, encryptedExtension []byte, extension string) (string, error) {
	encodedEncryptedFilename := base64.URLEncoding.EncodeToString(encryptedFilename)
	encodedEncryptedExtension := base64.URLEncoding.EncodeToString(encryptedExtension)

	encryptedFilePath := fmt.Sprintf("%v/%v", os.TempDir(), encodedEncryptedFilename)
	if extension != "" {
		encryptedFilePath = fmt.Sprintf("%v.%v", encryptedFilePath, encodedEncryptedExtension)
	}

	err := aesKey.EncryptFile(selectedFile, encryptedFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt file: %s", err)
	}

	return encryptedFilePath, nil
}
