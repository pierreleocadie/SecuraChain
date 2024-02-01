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
	"github.com/libp2p/go-libp2p/core/peerstore"
	relayClient "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	"github.com/multiformats/go-multiaddr"
	client "github.com/pierreleocadie/SecuraChain/internal/client"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/internal/discovery"
	"github.com/pierreleocadie/SecuraChain/internal/ipfs"
	"github.com/pierreleocadie/SecuraChain/internal/node"
	"github.com/pierreleocadie/SecuraChain/pkg/aes"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

var yamlConfigFilePath = flag.String("config", "", "Path to the yaml config file")

func main() {
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
	dhtApi := ipfsAPI.Dht()

	log.Debugf("IPFS node spawned with PeerID: %s", nodeIpfs.Identity.String())

	/*
	* NODE LIBP2P
	 */
	// Initialize the node
	host := node.Initialize(*cfg)
	defer host.Close()

	// Setup DHT discovery
	node.SetupDHTDiscovery(ctx, cfg, host, false)

	/*
	* RELAY SERVICE
	 */
	// Check if the node is behind NAT
	behindNAT := discovery.NATDiscovery(log)

	// If the node is behind NAT, search for a node that supports relay
	// TODO: Optimize this code
	if behindNAT {
		go func() {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					for _, p := range host.Network().Peers() {
						peerProtocols, err := host.Peerstore().GetProtocols(p)
						if err != nil {
							log.Errorln("Error getting peer protocols : ", err)
							continue
						}
						for _, protocol := range peerProtocols {
							if protocol == "/libp2p/circuit/relay/0.2.0" {
								log.Debugln("Found relay node : ", p.String())
								// Reserve with the relay node
								_, err := relayClient.Reserve(ctx, host, host.Peerstore().PeerInfo(p))
								if err != nil {
									log.Errorln("Error reserving with relay node : ", err)
									continue
								}
								// Remove all current host addresses to avoid dialing directly that would bypass the relay and fail
								host.Peerstore().ClearAddrs(host.ID())
								log.Debugf("Removed all host addresses")
								log.Debugf("Host addresses : %v", host.Addrs())
								// Add a new address using the relay node for the host
								relayAddr, err := multiaddr.NewMultiaddr("/p2p/" + p.String() + "/p2p-circuit/p2p/" + host.ID().String())
								if err != nil {
									log.Errorln("Error creating relay address : ", err)
									continue
								}
								host.Peerstore().AddAddr(host.ID(), relayAddr, peerstore.PermanentAddrTTL)
								log.Debugln("Added relay address : ", relayAddr.String())
								log.Debugf("Host addresses : %v", host.Addrs())
								break
							}
						}
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	} else {
		log.Debugln("Node is not behind NAT")
		// Start the relay service
		_, err = relay.New(host)
		if err != nil {
			log.Errorln("Error instantiating relay service : ", err)
		}
	}

	/*
	* PUBSUB
	 */
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		panic(err)
	}

	// Join the topic clientAnnouncementStringFlag
	clientAnnouncementTopic, err := ps.Join(cfg.ClientAnnouncementStringFlag)
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
				log.Debugf("Waiting for %d providers to be available", providerStats.TotalProvides)
				time.Sleep(10 * time.Second)
			}

			// Find providers for the file
			providers, err := dhtApi.FindProviders(ctx, clientAnnouncementPath)
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
			log.Debugln("Published ClientAnnouncement : ", string(clientAnnouncementJSON))
		}
	}()

	// Handle incoming ClientAnnouncement messages
	go func() {
		for {
			msg, err := subClientAnnouncement.Next(ctx)
			if err != nil {
				panic(err)
			}
			if msg.GetFrom().String() != host.ID().String() {
				log.Debugln("Received ClientAnnouncement message from ", msg.GetFrom().String())
			}
		}
	}()

	// Join the topic StorageNodeResponseStringFlag
	storageNodeResponseTopic, err := ps.Join(cfg.StorageNodeResponseStringFlag)
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
