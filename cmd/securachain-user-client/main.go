package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	ipfsLog "github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/network"
	client "github.com/pierreleocadie/SecuraChain/internal/client"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/internal/ipfs"
	"github.com/pierreleocadie/SecuraChain/internal/node"
	fileregistry "github.com/pierreleocadie/SecuraChain/internal/registry/file_registry"
	"github.com/pierreleocadie/SecuraChain/pkg/aes"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

var yamlConfigFilePath = flag.String("config", "", "Path to the yaml config file")

func main() { //nolint: funlen, gocyclo
	log := ipfsLog.Logger("user-client")
	err := ipfsLog.SetLogLevel("user-client", "DEBUG")
	if err != nil {
		log.Errorln("Error setting log level : ", err)
	}

	var ecdsaKeyPair ecdsa.KeyPair
	var aesKey aes.Key
	var clientAnnouncementChan = make(chan *transaction.ClientAnnouncement)
	var askFilesListChan = make(chan []byte)

	a := app.New()
	w := a.NewWindow("SecuraChain User Client")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	flag.Parse()

	// Load the config file
	if *yamlConfigFilePath == "" {
		log.Panicln("Please provide a path to the yaml config file")
	}

	cfg, err := config.LoadConfig(*yamlConfigFilePath)
	if err != nil {
		log.Panicln("Error loading config file : ", err)
	}

	/*
	* IPFS NODE
	 */
	IPFSNode := ipfs.NewIPFSNode(ctx, log, cfg)

	log.Debugf("IPFS node spawned with PeerID: %s", IPFSNode.Node.Identity.String())

	/*
	* NODE LIBP2P
	 */
	// Initialize the node
	host := node.Initialize(log, *cfg)
	defer host.Close()

	// Setup DHT discovery
	_ = node.SetupDHTDiscovery(ctx, cfg, host, false, log)

	/*
	* PUBSUB
	 */
	ps, err := pubsub.NewGossipSub(ctx, host, pubsub.WithMaxMessageSize(int(cfg.MaxDataRelayed)))
	if err != nil {
		log.Panicf("Failed to create GossipSub: %s", err)
	}

	// KeepRelayConnectionAlive
	keepRelayConnectionAliveTopic, err := ps.Join(cfg.KeepRelayConnectionAliveStringFlag)
	if err != nil {
		log.Warnf("Failed to join KeepRelayConnectionAlive topic: %s", err)
	}

	// Subscribe to KeepRelayConnectionAlive topic
	subKeepRelayConnectionAlive, err := keepRelayConnectionAliveTopic.Subscribe()
	if err != nil {
		log.Warnf("Failed to subscribe to KeepRelayConnectionAlive topic: %s", err)
	}

	// NetworkVisualisation
	networkVisualisationTopic, err := ps.Join(cfg.NetworkVisualisationStringFlag)
	if err != nil {
		log.Warnf("Failed to join NetworkVisualisation topic: %s", err)
	}

	// Join the topic StorageNodeResponseStringFlag
	storageNodeResponseTopic, err := ps.Join(cfg.StorageNodeResponseStringFlag)
	if err != nil {
		log.Errorf("Failed to join StorageNodeResponse topic: %s", err)
	}

	// Subscribe to StorageNodeResponseStringFlag topic
	subStorageNodeResponse, err := storageNodeResponseTopic.Subscribe()
	if err != nil {
		log.Errorf("Failed to subscribe to StorageNodeResponse topic: %s", err)
	}

	// Join the topic clientAnnouncementStringFlag
	clientAnnouncementTopic, err := ps.Join(cfg.ClientAnnouncementStringFlag)
	if err != nil {
		log.Warnf("Failed to join clientAnnouncement topic: %s", err)
	}

	// Join the topic AskMyFilesStringFlag
	askMyFilesTopic, err := ps.Join(cfg.AskMyFilesStringFlag)
	if err != nil {
		panic(err)
	}

	// Join the topic SendFilesStringFlag
	sendFilesTopic, err := ps.Join(cfg.SendFilesStringFlag)
	if err != nil {
		log.Panicf("Failed to join SendFiles topic : %s\n", err)
	}

	subSendFiles, err := sendFilesTopic.Subscribe()
	if err != nil {
		log.Panicf("Failed to subscribe to SendFiles topic : %s\n", err)
	}

	pubsubHub := &node.PubSubHub{
		ClientAnnouncementTopic:       clientAnnouncementTopic,
		ClientAnnouncementSub:         nil,
		StorageNodeResponseTopic:      storageNodeResponseTopic,
		StorageNodeResponseSub:        subStorageNodeResponse,
		KeepRelayConnectionAliveTopic: keepRelayConnectionAliveTopic,
		KeepRelayConnectionAliveSub:   subKeepRelayConnectionAlive,
		BlockAnnouncementTopic:        nil,
		BlockAnnouncementSub:          nil,
		AskingBlockchainTopic:         nil,
		AskingBlockchainSub:           nil,
		ReceiveBlockchainTopic:        nil,
		ReceiveBlockchainSub:          nil,
		AskMyFilesTopic:               askMyFilesTopic,
		AskMyFilesSub:                 nil,
		SendMyFilesTopic:              sendFilesTopic,
		SendMyFilesSub:                subSendFiles,
		NetworkVisualisationTopic:     networkVisualisationTopic,
		NetworkVisualisationSub:       nil,
	}

	// KeepRelayConnectionAlive
	node.PubsubKeepRelayConnectionAlive(ctx, pubsubHub, host, cfg, log)

	// Handle publishing ClientAnnouncement messages
	go func() {
		for {
			clientAnnouncement := <-clientAnnouncementChan
			clientAnnouncementJSON, err := transaction.SerializeTransaction(clientAnnouncement)
			// clientAnnouncementPath := path.FromCid(clientAnnouncement.FileCid)
			if err != nil {
				log.Errorln("Error serializing ClientAnnouncement : ", err)
				continue
			}

			log.Debugln("Publishing ClientAnnouncement : ", string(clientAnnouncementJSON))
			err = clientAnnouncementTopic.Publish(ctx, clientAnnouncementJSON)
			if err != nil {
				log.Errorln("Error publishing ClientAnnouncement : ", err)
				continue
			}
		}
	}()

	// Handle incoming NodeResponse messages
	go func() {
		for {
			msg, err := subStorageNodeResponse.Next(ctx)
			if err != nil {
				log.Errorf("Failed to get next message from StorageNodeResponse topic: %s", err)
				continue
			}
			log.Debugln("Received StorageNodeResponse message from ", msg.GetFrom().String())
			log.Debugln("StorageNodeResponse: ", string(msg.Data))
			addFileTransaction, err := transaction.DeserializeTransaction(msg.Data)
			if err != nil {
				log.Errorln("Error deserializing AddFileTransaction : ", err)
				continue
			}
			ecdsaPubKeyByte, err := ecdsaKeyPair.PublicKeyToBytes()
			if err != nil {
				log.Errorln("Error getting public key : ", err)
				continue
			}
			if bytes.Equal(addFileTransaction.(*transaction.AddFileTransaction).OwnerAddress, ecdsaPubKeyByte) {
				log.Debugln("Owner of the file is the current user")
				filename, err := aesKey.DecryptData(addFileTransaction.(*transaction.AddFileTransaction).Filename)
				if err != nil {
					log.Errorln("Error decrypting filename : ", err)
					continue
				}
				fileExtension, err := aesKey.DecryptData(addFileTransaction.(*transaction.AddFileTransaction).Extension)
				if err != nil {
					log.Errorln("Error decrypting file extension : ", err)
					continue
				}
				filenameStr := fmt.Sprintf("Your file %s%s has been stored successfully by a storage node.", filename, fileExtension)
				dialog.ShowInformation("Storage Node Response", filenameStr, w)
			}
		}
	}()

	// Handle publishing AskMyFiles messages
	go func() {
		for {
			askMyFiles := <-askFilesListChan
			err := askMyFilesTopic.Publish(ctx, askMyFiles)
			if err != nil {
				log.Errorln("Error publishing AskMyFiles : ", err)
				continue
			}
		}
	}()

	// Handle incoming SendFiles messages
	go func() {
		for {
			msg, err := subSendFiles.Next(ctx)
			if err != nil {
				log.Errorf("Failed to get next message from SendFiles topic: %s", err)
				continue
			}

			log.Debugln("Received SendFiles message from ", msg.GetFrom().String())
			log.Debugln("SendFiles: ", string(msg.Data))

			fileRegistry := fileregistry.Message{}
			if err = json.Unmarshal(msg.Data, &fileRegistry); err != nil {
				log.Errorln("Error deserializing RegistryMessage : ", err)
				continue
			}

			ownerECDSAPubKeyBytes, err := ecdsaKeyPair.PublicKeyToBytes()
			if err != nil {
				log.Errorln("Error getting public key : ", err)
				continue
			}

			ownerECDSAPubKeyStr := fmt.Sprintf("%x", ownerECDSAPubKeyBytes)
			if fileRegistry.OwnerPublicKey == ownerECDSAPubKeyStr {
				log.Debugln("Owner of the files is the current user")

				fileListContainer := container.NewVBox()

				for _, fileRegistry := range fileRegistry.Registry {
					filename, err := aesKey.DecryptData(fileRegistry.Filename)
					if err != nil {
						log.Errorln("Error decrypting filename : ", err)
						continue
					}

					fileExtension, err := aesKey.DecryptData(fileRegistry.Extension)
					if err != nil {
						log.Errorln("Error decrypting file extension : ", err)
						continue
					}

					label := widget.NewLabel(string(filename) + string(fileExtension) + " - " + fileRegistry.FileCid.String())
					downloadButton := client.DownloadButtonWidget(log, ctx, cfg, IPFSNode.API, fileRegistry.FileCid, string(filename), string(fileExtension), aesKey, w)

					// Add the label and the button to a horizontal container, then to the vertical container
					fileEntry := container.NewHBox(label, downloadButton)
					fileListContainer.Add(fileEntry)
				}

				fileDialog := dialog.NewCustom("My files", "Close", fileListContainer, w)
				fileDialog.Show()
			}
		}
	}()

	/*
	* GUI FYNE
	 */
	w.Resize(fyne.NewSize(cfg.WindowWidth, cfg.WindowHeight))

	ecdsaInput := widget.NewLabel("")
	hBoxECDSA, ecdsaButtonLoad, ecdsaButton := client.CreateECDSAWidgets(w, ecdsaInput, log, &ecdsaKeyPair)

	aesInput := widget.NewLabel("")
	hBoxAES, aesButtonLoad, aesButton := client.CreateAESWidgets(w, aesInput, log, &aesKey)

	selectedFileLabel := widget.NewLabel("")
	hBoxSelectFile, _ := client.CreateFileSelectionWidgets(w, selectedFileLabel, log)

	askFilesListButton := client.AskFilesListButton(w, cfg, &ecdsaKeyPair, askFilesListChan, log)

	// create a new button to send a file over the network
	sendFileButton := client.SendFileButton(ctx,
		cfg,
		w,
		selectedFileLabel,
		&ecdsaKeyPair,
		&aesKey,
		IPFSNode,
		clientAnnouncementChan,
		log,
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
		askFilesListButton,
	)

	protocolUpdatedSub, err := host.EventBus().Subscribe(new(event.EvtPeerProtocolsUpdated))
	if err != nil {
		log.Errorf("Failed to subscribe to EvtPeerProtocolsUpdated: %s", err)
	}
	go func(sub event.Subscription) {
		for e := range sub.Out() {
			var updated bool
			for _, proto := range e.(event.EvtPeerProtocolsUpdated).Added {
				if proto == pubsub.GossipSubID_v11 || proto == pubsub.GossipSubID_v10 {
					updated = true
					break
				}
			}
			if updated {
				for _, c := range host.Network().ConnsToPeer(e.(event.EvtPeerProtocolsUpdated).Peer) {
					(*pubsub.PubSubNotif)(ps).Connected(host.Network(), c)
				}
			}
		}
	}(protocolUpdatedSub)

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
