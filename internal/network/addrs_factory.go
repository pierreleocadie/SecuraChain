package network

import (
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
)

// FilterOutPrivateAddrs filters out private IP addresses from a list of multiaddresses.
func FilterOutPrivateAddrs(addrs []ma.Multiaddr) []ma.Multiaddr {
	var publicAddrs []ma.Multiaddr

	for _, addr := range addrs {
		if !isPrivateAddr(addr) {
			publicAddrs = append(publicAddrs, addr)
		}
	}

	return publicAddrs
}

// isPrivateAddr checks if a multiaddress corresponds to a private IP address.
func isPrivateAddr(addr ma.Multiaddr) bool {
	return !manet.IsPublicAddr(addr)
}
