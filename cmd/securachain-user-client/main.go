package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/network"

	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/internal/node"
	"github.com/pierreleocadie/SecuraChain/pkg/aes"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
	"github.com/pierreleocadie/SecuraChain/pkg/utils"
)

func main() {
	var ecdsaKeyPair ecdsa.KeyPair
	var aesKey aes.Key
	var clientAnnouncementChan = make(chan *transaction.ClientAnnouncement)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	/*
	* NODE LIBP2P
	 */
	// Initialize the node
	host := node.Initialize()
	defer host.Close()

	// Setup DHT discovery
	node.SetupDHTDiscovery(ctx, host, false)

	/*
	* DISPLAY PEER CONNECTEDNESS CHANGES
	 */
	// Subscribe to EvtPeerConnectednessChanged events
	subNet, err := host.EventBus().Subscribe(new(event.EvtPeerConnectednessChanged))
	if err != nil {
		log.Println("Failed to subscribe to EvtPeerConnectednessChanged: ", err)
	}
	defer subNet.Close()

	// Handle connection events in a separate goroutine
	go func() {
		for e := range subNet.Out() {
			evt := e.(event.EvtPeerConnectednessChanged)
			if evt.Connectedness == network.Connected {
				log.Println("Peer connected:", evt.Peer)
			} else if evt.Connectedness == network.NotConnected {
				log.Println("Peer disconnected: ", evt.Peer)
			}
		}
	}()

	/*
	* PUBSUB
	 */
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		panic(err)
	}

	// Join the topic clientAnnouncementStringFlag
	clientAnnouncementTopic, err := ps.Join(config.ClientAnnouncementStringFlag)
	if err != nil {
		panic(err)
	}

	// Subscribe to clientAnnouncementStringFlag topic
	subClientAnnouncement, err := clientAnnouncementTopic.Subscribe()
	if err != nil {
		panic(err)
	}

	// Join the topic storageNodeResponseStringFlag
	storageNodeResponseTopic, err := ps.Join(config.StorageNodeResponseStringFlag)
	if err != nil {
		panic(err)
	}

	// Subscribe to storageNodeResponseStringFlag topic
	subStorageNodeResponse, err := storageNodeResponseTopic.Subscribe()
	if err != nil {
		panic(err)
	}

	// handle incoming StorageNodeResponse messages
	go func() {
		for {
			msg, err := subStorageNodeResponse.Next(ctx)
			if err != nil {
				panic(err)
			}

			log.Println("Received StorageNodeResponse message : ", string(msg.Data))
		}
	}()

	// Handle publishing ClientAnnouncement messages
	go func() {
		for {
			clientAnnouncement := <-clientAnnouncementChan
			clientAnnouncementJson, err := clientAnnouncement.Serialize()
			if err != nil {
				log.Println("Error serializing ClientAnnouncement : ", err)
				continue
			}

			log.Println("Publishing ClientAnnouncement : ", string(clientAnnouncementJson))

			err = clientAnnouncementTopic.Publish(ctx, clientAnnouncementJson)
			if err != nil {
				log.Println("Error publishing ClientAnnouncement : ", err)
				continue
			}
		}
	}()

	// Handle incoming ClientAnnouncement messages
	go func() {
		for {
			msg, err := subClientAnnouncement.Next(ctx)
			if err != nil {
				panic(err)
			}
			log.Println("Received ClientAnnouncement message from ", msg.GetFrom().String())
		}
	}()

	/*
	* GUI FYNE
	 */
	a := app.New()
	w := a.NewWindow("SecuraChain User Client")
	w.Resize(fyne.NewSize(800, 600))

	// Create an input to set file path with a browse button to save the ECDSA key pair
	ecdsaInput := widget.NewLabel("")
	ecdsaBrowseButton := widget.NewButton("Browse", func() {
		dialog := dialog.NewFolderOpen(func(dir fyne.ListableURI, err error) {
			if err == nil && (dir != nil) {
				u, er := url.Parse(dir.String())
				if er == nil {
					ecdsaInput.SetText(u.Path)
					log.Printf("ECDSA key pair will be saved in : %v", u.Path)
				}
			}
		}, w)
		dialog.Show()
	})

	// Create an input to set file path with a browse button to save the AES key
	aesInput := widget.NewLabel("")
	aesBrowseButton := widget.NewButton("Browse", func() {
		dialog := dialog.NewFolderOpen(func(dir fyne.ListableURI, err error) {
			if err == nil && (dir != nil) {
				u, er := url.Parse(dir.String())
				if er == nil {
					aesInput.SetText(u.Path)
					log.Printf("AES key will be saved in : %v", u.Path)
				}
			}
		}, w)
		dialog.Show()
	})

	// Create a button to load an ECDSA key pair
	ecdsaButtonLoad := widget.NewButton("Load ECDSA Key Pair", func() {
		ecdsaKeyPairL, err := ecdsa.LoadKeys("ecdsaPrivateKey", "ecdsaPublicKey", ecdsaInput.Text)
		if err != nil {
			log.Println("Error loading ECDSA key pair : ", err)
			return
		}
		ecdsaKeyPair = ecdsaKeyPairL
		log.Println(ecdsaKeyPair)
	})

	// Create a button to generate a new ECDSA key pair
	ecdsaButton := widget.NewButton("Generate ECDSA Key Pair", func() {
		if ecdsaInput.Text == "" {
			log.Println("Please select a directory to save the ECDSA key pair")
			return
		}

		ecdsaKeyPair, err := ecdsa.NewECDSAKeyPair()
		if err != nil {
			return
		}

		err = ecdsaKeyPair.SaveKeys("ecdsaPrivateKey", "ecdsaPublicKey", ecdsaInput.Text)
		if err != nil {
			log.Println("Error saving ECDSA key pair : ", err)
			return
		}
	})

	// Create a button to load an AES key
	aesButtonLoad := widget.NewButton("Load AES Key", func() {
		if aesInput.Text == "" {
			log.Println("Please select a directory to save the AES key")
			return
		}

		aesKeyL, err := aes.LoadKey("aesKey", aesInput.Text)
		if err != nil {
			log.Println("Error loading AES key : ", err)
			return
		}
		aesKey = aesKeyL
		log.Println(aesKey)
	})

	// Create a button to generate a new AES key
	aesButton := widget.NewButton("Generate AES Key", func() {
		if aesInput.Text == "" {
			log.Println("Please select a directory to save the AES key")
			return
		}

		aesKey, err := aes.NewAESKey()
		if err != nil {
			log.Println("Error generating AES key : ", err)
			return
		}

		err = aesKey.SaveKey("aesKey", aesInput.Text)
		if err != nil {
			log.Println("Error saving AES key : ", err)
			return
		}
	})

	// Create a new button to select a file
	selectedFileLabel := widget.NewLabel("")
	selectFileButton := widget.NewButton("Select File", func() {
		dialog := dialog.NewFileOpen(func(file fyne.URIReadCloser, err error) {
			if err == nil && (file != nil) {
				selectedFileLabel.SetText(file.URI().Path())
			}
		}, w)
		dialog.Show()
	})

	// create a new button to send a file over the network
	sendFileButton := widget.NewButton("Send File", func() {
		// -1. Check if the ECDSA key pair and the AES key are loaded
		if aesKey == nil {
			log.Println("AES key not loaded")
			return
		}

		if ecdsaKeyPair == nil {
			log.Println("ECDSA key pair not loaded")
			return
		}

		// O. Get the original file name and extension
		if selectedFileLabel.Text == "" {
			log.Println("Please select a file")
			return
		}

		filename := strings.Split(selectedFileLabel.Text, "/")[len(strings.Split(selectedFileLabel.Text, "/"))-1]
		extensionSplit := strings.Split(filename, ".")
		extension := extensionSplit[len(strings.Split(filename, "."))-1]
		if len(extensionSplit) == 1 {
			filename = extension
			extension = ""
		}
		log.Println("filename : ", filename)
		log.Println("extension : ", extension)

		// 1. Encrypt the filename and file extension with AES
		encryptedFilename, err := aesKey.EncryptData([]byte(filename))
		if err != nil {
			log.Println("Error encrypting filename : ", err)
			return
		}

		encryptedExtension, err := aesKey.EncryptData([]byte(extension))
		if err != nil {
			log.Println("Error encrypting extension : ", err)
			return
		}

		if extension == "" {
			encryptedExtension = []byte("")
		}

		// 2. Encrypt the file with AES
		encodedEncryptedFilename := base64.URLEncoding.EncodeToString(encryptedFilename)
		log.Println("Encoded encrypted filename : ", encodedEncryptedFilename)

		encodedEncryptedExtension := base64.URLEncoding.EncodeToString(encryptedExtension)
		log.Println("Encoded encrypted extension : ", encodedEncryptedExtension)

		encryptedFilePath := fmt.Sprintf("%v/%v", os.TempDir(), encodedEncryptedFilename)
		if extension != "" {
			encryptedFilePath = fmt.Sprintf("%v.%v", encryptedFilePath, encodedEncryptedExtension)
		}
		log.Println("Path for the encrypted file : ", encryptedFilePath)

		err = aesKey.EncryptFile(selectedFileLabel.Text, encryptedFilePath)
		if err != nil {
			log.Println("Error encrypting file : ", err)
			return
		}

		// 3. Compute the checksum of the encrypted file
		encryptedFileChecksum, err := utils.ComputeFileChecksum(encryptedFilePath)
		if err != nil {
			log.Println("Error computing checksum of encrypted file : ", err)
			return
		}
		log.Println("Encrypted file checksum : ", string(encryptedFileChecksum))

		// 4. Get the size of the encrypted file
		fileStat, err := os.Stat(encryptedFilePath)
		if err != nil {
			log.Println("Error getting size of encrypted file : ", err)
			return
		}
		fileSize := fileStat.Size()
		log.Println("Encrypted file size : ", fileSize)

		// 7. Create a new ClientAnnouncement
		clientAnnouncement := transaction.NewClientAnnouncement(
			ecdsaKeyPair,
			encryptedFilename,
			encryptedExtension,
			uint64(fileSize),
			encryptedFileChecksum,
		)
		clientAnnouncementChan <- clientAnnouncement
		// clientAnnouncementJson, err := clientAnnouncement.Serialize()
		// if err != nil {
		// 	log.Println("Error serializing ClientAnnouncement : ", err)
		// 	return
		// }
		// log.Println("ClientAnnouncement : ", string(clientAnnouncementJson))
	})

	hBoxECDSA := container.New(
		layout.NewHBoxLayout(),
		ecdsaInput,
		layout.NewSpacer(),
		ecdsaBrowseButton,
	)
	hBoxAES := container.New(
		layout.NewHBoxLayout(),
		aesInput,
		layout.NewSpacer(),
		aesBrowseButton,
	)
	hBoxSelectFile := container.New(
		layout.NewHBoxLayout(),
		selectFileButton,
		selectedFileLabel,
	)
	vBox := container.New(
		layout.NewVBoxLayout(),
		widget.NewLabel("ECDSA Key Pair:"),
		hBoxECDSA,
		ecdsaButtonLoad,
		ecdsaButton,
		widget.NewLabel("AES Key:"),
		hBoxAES,
		aesButtonLoad,
		aesButton,
		widget.NewLabel("Select File:"),
		hBoxSelectFile,
		sendFileButton,
	)

	w.SetContent(vBox)
	w.ShowAndRun()
}
