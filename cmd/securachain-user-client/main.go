package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/google/uuid"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/network"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/internal/ipfs"
	"github.com/pierreleocadie/SecuraChain/internal/node"
	"github.com/pierreleocadie/SecuraChain/pkg/aes"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
	"github.com/pierreleocadie/SecuraChain/pkg/utils"
)

func main() {
	log := ipfsLog.Logger("user-client")
	ipfsLog.SetLogLevel("user-client", "INFO")

	var ecdsaKeyPair ecdsa.KeyPair
	var aesKey aes.Key
	var clientAnnouncementChan = make(chan *transaction.ClientAnnouncement)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	/*
	* IPFS NODE
	 */
	// Spawn an IPFS node
	ipfsApi, nodeIpfs, err := ipfs.SpawnNode(ctx)
	if err != nil {
		log.Fatalf("Failed to spawn IPFS node: %s", err)
	}

	log.Debugf("IPFS node spawned with PeerID: %s", nodeIpfs.Identity.String())

	/*
	* NODE LIBP2P
	 */
	// Initialize the node
	host := node.Initialize()
	defer host.Close()

	// Setup DHT discovery
	node.SetupDHTDiscovery(ctx, host, false)

	/*
	* PUBSUB
	 */
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		panic(err)
	}

	stayAliveTopic, err := ps.Join("stayAlive")
	if err != nil {
		panic(err)
	}

	subStayAlive, err := stayAliveTopic.Subscribe()
	if err != nil {
		panic(err)
	}

	// Handle incoming stay alive messages
	go func() {
		for {
			msg, err := subStayAlive.Next(ctx)
			if err != nil {
				log.Errorln("Error getting stay alive message : ", err)
			}
			log.Debugln("Received stay alive message from ", msg.GetFrom().String())
			uuidByte, err := uuid.FromBytes(msg.Data)
			if err != nil {
				log.Errorln("Error unmarshaling uuid : ", err)
				continue
			}
			log.Debugln("Received stay alive message : ", uuidByte.String())
		}
	}()

	// Publish stay alive messages
	go func() {
		for {
			uuidByte, err := uuid.New().MarshalBinary()
			if err != nil {
				log.Errorln("Error marshaling uuid : ", err)
				continue
			}
			err = stayAliveTopic.Publish(ctx, uuidByte)
			if err != nil {
				log.Errorln("Error publishing stay alive message : ", err)
			}
			log.Debugln("Published stay alive message")
			// Sleep for a random duration between 1 and 5 seconds
			time.Sleep(time.Duration(rand.Intn(10)+1) * time.Second)
		}
	}()

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

	// Handle publishing ClientAnnouncement messages
	go func() {
		for {
			clientAnnouncement := <-clientAnnouncementChan
			clientAnnouncementJson, err := clientAnnouncement.Serialize()
			if err != nil {
				log.Errorln("Error serializing ClientAnnouncement : ", err)
				continue
			}

			log.Debugln("Publishing ClientAnnouncement : ", string(clientAnnouncementJson))

			err = clientAnnouncementTopic.Publish(ctx, clientAnnouncementJson)
			if err != nil {
				log.Errorln("Error publishing ClientAnnouncement : ", err)
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
			log.Debugln("Received ClientAnnouncement message from ", msg.GetFrom().String())
		}
	}()

	// Join the topic StorageNodeResponseStringFlag
	storageNodeResponseTopic, err := ps.Join(config.StorageNodeResponseStringFlag)
	if err != nil {
		panic(err)
	}

	// Subscribe to StorageNodeResponseStringFlag topic
	subStorageNodeResponse, err := storageNodeResponseTopic.Subscribe()
	if err != nil {
		panic(err)
	}

	// Handle incoming NodeResponse messages
	go func() {
		for {
			msg, err := subStorageNodeResponse.Next(ctx)
			if err != nil {
				panic(err)
			}
			log.Debugln("Received StorageNodeResponse message from ", msg.GetFrom().String())
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
					log.Debugf("ECDSA key pair will be saved in : %v", u.Path)
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
					log.Debugf("AES key will be saved in : %v", u.Path)
				}
			}
		}, w)
		dialog.Show()
	})

	// Create a button to load an ECDSA key pair
	ecdsaButtonLoad := widget.NewButton("Load ECDSA Key Pair", func() {
		ecdsaKeyPairL, err := ecdsa.LoadKeys("ecdsaPrivateKey", "ecdsaPublicKey", ecdsaInput.Text)
		if err != nil {
			log.Errorln("Error loading ECDSA key pair : ", err)
			return
		}
		ecdsaKeyPair = ecdsaKeyPairL
		log.Debug("ECDSA key pair loaded")
	})

	// Create a button to generate a new ECDSA key pair
	ecdsaButton := widget.NewButton("Generate ECDSA Key Pair", func() {
		if ecdsaInput.Text == "" {
			log.Debug("Please select a directory to save the ECDSA key pair")
			return
		}

		ecdsaKeyPair, err := ecdsa.NewECDSAKeyPair()
		if err != nil {
			return
		}

		err = ecdsaKeyPair.SaveKeys("ecdsaPrivateKey", "ecdsaPublicKey", ecdsaInput.Text)
		if err != nil {
			log.Errorln("Error saving ECDSA key pair : ", err)
			return
		}
	})

	// Create a button to load an AES key
	aesButtonLoad := widget.NewButton("Load AES Key", func() {
		if aesInput.Text == "" {
			log.Debug("Please select a directory to save the AES key")
			return
		}

		aesKeyL, err := aes.LoadKey("aesKey", aesInput.Text)
		if err != nil {
			log.Errorln("Error loading AES key : ", err)
			return
		}
		aesKey = aesKeyL
		log.Debug("AES key loaded")
	})

	// Create a button to generate a new AES key
	aesButton := widget.NewButton("Generate AES Key", func() {
		if aesInput.Text == "" {
			log.Debug("Please select a directory to save the AES key")
			return
		}

		aesKey, err := aes.NewAESKey()
		if err != nil {
			log.Errorln("Error generating AES key : ", err)
			return
		}

		err = aesKey.SaveKey("aesKey", aesInput.Text)
		if err != nil {
			log.Errorln("Error saving AES key : ", err)
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
			log.Debug("AES key not loaded")
			return
		}

		if ecdsaKeyPair == nil {
			log.Debug("ECDSA key pair not loaded")
			return
		}

		// O. Get the original file name and extension
		if selectedFileLabel.Text == "" {
			log.Debug("Please select a file")
			return
		}

		filename, _, extension, err := utils.FileInfo(selectedFileLabel.Text)
		if err != nil {
			log.Errorln("Error getting file info : ", err)
			return
		}

		filename = strings.Split(filename, ".")[0]
		log.Debugln("filename : ", filename)
		log.Debugln("extension : ", extension)

		// 1. Encrypt the filename and file extension with AES
		encryptedFilename, err := aesKey.EncryptData([]byte(filename))
		if err != nil {
			log.Errorln("Error encrypting filename : ", err)
			return
		}

		encryptedExtension, err := aesKey.EncryptData([]byte(extension))
		if err != nil {
			log.Errorln("Error encrypting extension : ", err)
			return
		}

		// 2. Encrypt the file with AES
		encodedEncryptedFilename := base64.URLEncoding.EncodeToString(encryptedFilename)
		log.Debugln("Encoded encrypted filename : ", encodedEncryptedFilename)

		encodedEncryptedExtension := base64.URLEncoding.EncodeToString(encryptedExtension)
		log.Debugln("Encoded encrypted extension : ", encodedEncryptedExtension)

		encryptedFilePath := fmt.Sprintf("%v/%v", os.TempDir(), encodedEncryptedFilename)
		if extension != "" {
			encryptedFilePath = fmt.Sprintf("%v.%v", encryptedFilePath, encodedEncryptedExtension)
		}
		log.Debugln("Path for the encrypted file : ", encryptedFilePath)

		err = aesKey.EncryptFile(selectedFileLabel.Text, encryptedFilePath)
		if err != nil {
			log.Errorln("Error encrypting file : ", err)
			return
		}

		// 3. Compute the checksum of the encrypted file
		encryptedFileChecksum, err := utils.ComputeFileChecksum(encryptedFilePath)
		if err != nil {
			log.Errorln("Error computing checksum of encrypted file : ", err)
			return
		}
		log.Debugln("Encrypted file checksum : ", string(encryptedFileChecksum))

		// 4. Get the size of the encrypted file
		fileStat, err := os.Stat(encryptedFilePath)
		if err != nil {
			log.Debugln("Error getting size of encrypted file : ", err)
			return
		}
		fileSize := fileStat.Size()
		log.Debugln("Encrypted file size : ", fileSize)

		// 5. Add the encrypted file to IPFS
		encryptedFileCid, err := ipfs.AddFile(ctx, nodeIpfs, ipfsApi, encryptedFilePath)
		if err != nil {
			log.Errorln("Error adding encrypted file to IPFS : ", err)
			return
		}

		// 6. Pin the encrypted file
		isPinned, err := ipfs.PinFile(ctx, ipfsApi, encryptedFileCid)
		if err != nil {
			log.Errorln("Error pinning encrypted file : ", err)
			return
		}
		log.Debugln("Encrypted file pinned : ", isPinned)

		log.Debugln("Encrypted file immutable path : ", encryptedFileCid)

		// 7. Create a new ClientAnnouncement
		clientAnnouncement := transaction.NewClientAnnouncement(
			ecdsaKeyPair,
			encryptedFileCid.RootCid(),
			encryptedFilename,
			encryptedExtension,
			uint64(fileSize),
			encryptedFileChecksum,
		)
		clientAnnouncementChan <- clientAnnouncement
		clientAnnouncementJson, err := clientAnnouncement.Serialize()
		if err != nil {
			log.Errorln("Error serializing ClientAnnouncement : ", err)
			return
		}
		log.Debugln("ClientAnnouncement : ", string(clientAnnouncementJson))
		// // bis announce the file with a provider
		// provider.NewNoopProvider().Provide(encryptedFileCid.RootCid())

		//provider.NewNoopProvider().Reprovide(ctx)

		// err = p.Provide(encryptedFileCid.RootCid())

		// if err != nil {
		// 	log.Debugln("problem", err)
		// }

		// var x provider.Provide
		// x.Provide(ctx, encryptedFileCid.RootCid(), true)

		// provider.Online(x)
		log.Infoln(nodeIpfs.Provider.Stat())
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

	/*
	* DISPLAY PEER CONNECTEDNESS CHANGES
	 */
	// Subscribe to EvtPeerConnectednessChanged events
	subNet, err := host.EventBus().Subscribe(new(event.EvtPeerConnectednessChanged))
	if err != nil {
		log.Errorln("Failed to subscribe to EvtPeerConnectednessChanged: ", err)
	}
	defer subNet.Close()

	// Handle connection events in a separate goroutine
	go func() {
		for e := range subNet.Out() {
			evt := e.(event.EvtPeerConnectednessChanged)
			if evt.Connectedness == network.Connected {
				log.Debugln("Peer connected:", evt.Peer)
			} else if evt.Connectedness == network.NotConnected {
				log.Debugln("Peer disconnected: ", evt.Peer)
			}
		}
	}()

	w.SetContent(vBox)
	w.ShowAndRun()
}
