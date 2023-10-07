package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"fmt"
	"os"
)

func main() {
	// Generate a new private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Sauvegarder la clé privée dans un fichier
	privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		fmt.Println(err)
		return
	}

	privateFile, err := os.Create("privateKey.pem")
	if err != nil {
		fmt.Println(err)
	}
	defer privateFile.Close()

	_, err = privateFile.Write(privateKeyBytes)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Sauvegarder la clé publique dans un fichier
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		fmt.Println(err)
		return
	}

	publicFile, err := os.Create("publicKey.pem")
	if err != nil {
		fmt.Println(err)
	}
	defer publicFile.Close()

	_, err = publicFile.Write(publicKeyBytes)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Afficher la clé privée et la clé publique
	fmt.Printf("Private Key: %x\n", privateKeyBytes)
	fmt.Printf("Public Key: %x\n", publicKeyBytes)
}
