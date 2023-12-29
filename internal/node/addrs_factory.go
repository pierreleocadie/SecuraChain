package node

import (
	"net"

	ma "github.com/multiformats/go-multiaddr"
)

// isPrivateAddr checks if a given multiaddress corresponds to a private network
func isPrivateAddr(maddr ma.Multiaddr) bool {
	// Extract the IP address from the multiaddress
	ip, err := maddr.ValueForProtocol(ma.P_IP4)
	if err != nil {
		// Handle the error or ignore the address
		return false
	}

	// Parse the IP address
	netIP := net.ParseIP(ip)
	if netIP == nil {
		return false
	}

	// Check if the IP is in a private range
	return netIP.IsPrivate()
}

// filterOutPrivateAddrs filters out private network addresses
func filterOutPrivateAddrs(addrs []ma.Multiaddr) []ma.Multiaddr {
	var publicAddrs []ma.Multiaddr
	for _, addr := range addrs {
		if !isPrivateAddr(addr) {
			publicAddrs = append(publicAddrs, addr)
		}
	}
	return publicAddrs
}
