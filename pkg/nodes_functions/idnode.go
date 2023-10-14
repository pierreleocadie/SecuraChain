package main

import (
	"crypto/sha512"
	"fmt"
	"net"
)

func main() {
	macAdress := getMACAdress()
	if macAdress != nil {
		fmt.Printf("Id node : %x", hachMacAdress(macAdress))
	} else {
		fmt.Println("Adresse MAC non trouvée.")
	}

}

// give the macAdress
func getMACAdress() net.HardwareAddr {
	interfaces, err := net.Interfaces() // function to obtain the list of network interfaces
	if err != nil {
		fmt.Println("Erreur lors de la récupération des interfaces réseau:", err)
		return nil
	}

	for _, iface := range interfaces {
		if iface.Name == "en0" {
			macAdress := iface.HardwareAddr
			return macAdress
		}

	}
	return nil
}

// give the hash
func hachMacAdress(macAdress net.HardwareAddr) string {
	// Hash MAC address to SHA-512
	hash := sha512.New()
	hash.Write(macAdress) // Hash mac address
	macHash := hash.Sum(nil)

	macHashString := fmt.Sprintf("%s", macHash)[:8]

	return macHashString
}
