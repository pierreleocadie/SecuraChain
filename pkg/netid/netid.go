package netid

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"net"
)

type NetID struct {
	NetID string
}

func (n NetID) getMACAddress() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, i := range interfaces {
		if !bytes.Equal(i.HardwareAddr, nil) {
			return i.HardwareAddr.String(), nil
		}
	}

	return "", fmt.Errorf("no MAC address found")
}

func (g NetID) GenerateNetID() (string, error) {
	mac, err := g.getMACAddress()
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256([]byte(mac))
	netID := fmt.Sprintf("%x", hash)[:8]
	return netID, nil
}
