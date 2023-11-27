package internal

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	icore "github.com/ipfs/boxo/coreiface"
	"github.com/ipfs/kubo/core"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/multiformats/go-multiaddr"
)

var rendezvousStringFlag = fmt.Sprintln("SecuraChainNetwork")

func setupDHTDiscovery(ctx context.Context, host host.Host) {
	/*
	 * NETWORK PEER DISCOVERY WITH DHT
	 */
	// Initialize DHT in client mode
	dhtDiscovery := discovery.NewDHTDiscovery(
		false, // false pour le mode client
		rendezvousStringFlag,
		[]multiaddr.Multiaddr{}, // Liste des adresses des pairs de démarrage, si nécessaire
		0,                       // Intervalle de rafraîchissement de la découverte, ajustez selon vos besoins
	)

	// Run DHT
	if err := dhtDiscovery.Run(ctx, host); err != nil {
		log.Println("Failed to run DHT: ", err)
		return
	}
}

func waitForTermSignal() {
	// wait for a SIGINT or SIGTERM signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("Received signal, shutting down...")
}

// To connect to boostrap nodes

func ConnectToBoostrapNodes(ctx context.Context, node *core.IpfsNode, ipfsApi icore.CoreAPI, bootsrapPeers []string) error {

	setupDHTDiscovery(ctx, node.PeerHost)
	return nil
}
