package node

import (
	"context"
	"time"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/network"
)

func PubsubKeepRelayConnectionAlive(ctx context.Context,
	psh *PubSubHub, host host.Host, cfg *config.Config, log *ipfsLog.ZapEventLogger) {
	// Run KeepRelayConnectionAlive only if its behind a NAT
	if !network.NATDiscovery(log) {
		return
	}

	// Handle incoming KeepRelayConnectionAlive messages
	go func() {
		for {
			msg, err := psh.KeepRelayConnectionAliveSub.Next(ctx)
			if err != nil {
				log.Errorf("Failed to get next message from KeepRelayConnectionAlive topic: %s", err)
				continue
			}
			if msg.GetFrom() == host.ID() {
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
			err := psh.KeepRelayConnectionAliveTopic.Publish(ctx, network.GeneratePacket(host.ID()))
			if err != nil {
				log.Errorf("Failed to publish KeepRelayConnectionAlive message: %s", err)
				continue
			}
			log.Debugf("KeepRelayConnectionAlive message sent successfully")
		}
	}()
}
