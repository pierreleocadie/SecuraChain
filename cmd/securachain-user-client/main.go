package main

import (
	"context"
	"flag"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/ipfs/boxo/path"
	ipfsLog "github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/network"
	client "github.com/pierreleocadie/SecuraChain/internal/client"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/internal/ipfs"
	netwrk "github.com/pierreleocadie/SecuraChain/internal/network"
	"github.com/pierreleocadie/SecuraChain/internal/node"
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
	// Spawn an IPFS node
	ipfsAPI, nodeIpfs, err := ipfs.SpawnNode(ctx, cfg)
	if err != nil {
		log.Panicf("Failed to spawn IPFS node: %s", err)
	}
	dhtAPI := ipfsAPI.Dht()

	log.Debugf("IPFS node spawned with PeerID: %s", nodeIpfs.Identity.String())

	/*
	* NODE LIBP2P
	 */
	// Initialize the node
	host := node.Initialize(log, *cfg)
	defer host.Close()

	// Setup DHT discovery
	_ = node.SetupDHTDiscovery(ctx, cfg, host, false)

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

	// Handle incoming KeepRelayConnectionAlive messages
	go func() {
		for {
			msg, err := subKeepRelayConnectionAlive.Next(ctx)
			if err != nil {
				log.Errorf("Failed to get next message from KeepRelayConnectionAlive topic: %s", err)
				continue
			}
			log.Debugf("Received KeepRelayConnectionAlive message from %s", msg.GetFrom().String())
			log.Debugf("KeepRelayConnectionAlive: %s", string(msg.Data))
		}
	}()

	// Handle outgoing KeepRelayConnectionAlive messages
	go func() {
		for {
			time.Sleep(cfg.KeepRelayConnectionAliveInterval)
			err := keepRelayConnectionAliveTopic.Publish(ctx, netwrk.GeneratePacket(host.ID()))
			if err != nil {
				log.Errorf("Failed to publish KeepRelayConnectionAlive message: %s", err)
				continue
			}
			log.Debugf("KeepRelayConnectionAlive message sent successfully")
		}
	}()

	// Join the topic clientAnnouncementStringFlag
	clientAnnouncementTopic, err := ps.Join(cfg.ClientAnnouncementStringFlag)
	if err != nil {
		log.Warnf("Failed to join clientAnnouncement topic: %s", err)
	}

	// Subscribe to clientAnnouncementStringFlag topic
	subClientAnnouncement, err := clientAnnouncementTopic.Subscribe()
	if err != nil {
		log.Warnf("Failed to subscribe to clientAnnouncement topic: %s", err)
	}

	// Handle publishing ClientAnnouncement messages
	go func() {
		for {
			clientAnnouncement := <-clientAnnouncementChan
			clientAnnouncementJSON, err := clientAnnouncement.Serialize()
			clientAnnouncementPath := path.FromCid(clientAnnouncement.FileCid)
			if err != nil {
				log.Errorln("Error serializing ClientAnnouncement : ", err)
				continue
			}

			log.Debugln("Publishing ClientAnnouncement : ", string(clientAnnouncementJSON))
			for {
				providerStats, err := nodeIpfs.Provider.Stat()
				if err != nil {
					log.Errorln("Error getting provider stats : ", err)
					continue
				}
				log.Debugln("Total provides : ", providerStats)
				if providerStats.TotalProvides >= 1 {
					break
				}
				time.Sleep(cfg.IpfsProvidersDiscoveryRefreshInterval)
			}

			// Find providers for the file
			providers, err := dhtAPI.FindProviders(ctx, clientAnnouncementPath)
			if err != nil {
				log.Errorln("Error finding providers : ", err)
				continue
			}
			for provider := range providers {
				log.Debugln("Found provider : ", provider.ID.String())
			}

			err = clientAnnouncementTopic.Publish(ctx, clientAnnouncementJSON)
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
				log.Errorln("Error getting next message from clientAnnouncement topic : ", err)
				continue
			}
			if msg.GetFrom().String() != host.ID().String() {
				log.Debugln("Received ClientAnnouncement message from ", msg.GetFrom().String())
			}
		}
	}()

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

	// Handle incoming NodeResponse messages
	go func() {
		for {
			msg, err := subStorageNodeResponse.Next(ctx)
			if err != nil {
				log.Errorf("Failed to get next message from StorageNodeResponse topic: %s", err)
			}
			log.Debugln("Received StorageNodeResponse message from ", msg.GetFrom().String())
			log.Debugln("StorageNodeResponse: ", string(msg.Data))
		}
	}()

	/*
	* GUI FYNE
	 */
	a := app.New()
	w := a.NewWindow("SecuraChain User Client")
	w.Resize(fyne.NewSize(cfg.WindowWidth, cfg.WindowHeight))

	ecdsaInput := widget.NewLabel("")
	hBoxECDSA, ecdsaButtonLoad, ecdsaButton := client.CreateECDSAWidgets(w, ecdsaInput, log, &ecdsaKeyPair)

	aesInput := widget.NewLabel("")
	hBoxAES, aesButtonLoad, aesButton := client.CreateAESWidgets(w, aesInput, log, &aesKey)

	selectedFileLabel := widget.NewLabel("")
	hBoxSelectFile, _ := client.CreateFileSelectionWidgets(w, selectedFileLabel, log)

	// create a new button to send a file over the network
	sendFileButton := client.SendFileButton(ctx,
		cfg,
		selectedFileLabel,
		&ecdsaKeyPair,
		&aesKey,
		nodeIpfs,
		ipfsAPI,
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
