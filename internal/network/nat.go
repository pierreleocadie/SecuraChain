package network

import (
	"context"
	"io"
	"net"
	"net/http"

	ipfsLog "github.com/ipfs/go-log/v2"
)

func getLocalIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String(), nil
}

func getPublicIP() (string, error) {
	ctx := context.Background()

	// Création d'une requête HTTP avec contexte
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.ipify.org", nil)
	if err != nil {
		return "", err
	}

	// Execution de la requête HTTP
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Lecture de la réponse
	ip, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(ip), nil
}

func NATDiscovery(log *ipfsLog.ZapEventLogger) bool {
	localIP, err := getLocalIP()
	if err != nil {
		log.Errorf("Error getting local IP: %s", err)
		return false
	}

	publicIP, err := getPublicIP()
	if err != nil {
		log.Errorf("Error getting public IP: %s", err)
		return false
	}

	log.Debugf("Local IP: %s", localIP)
	log.Debugf("Public IP: %s", publicIP)

	if localIP != publicIP {
		log.Infof("This machine is behind NAT.")
		return true
	}

	log.Infof("This machine is not behind NAT.")
	return false
}
