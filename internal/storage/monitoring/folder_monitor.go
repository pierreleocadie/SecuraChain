package monitoring

// Définition de l'espace utilisé par l'utilisateur
// Vérifier périodiquement
// Vérifier mais à quel fréquence
// Générer des alertes ou des journaux sur l'utilisateion de l'espadce de stockage

// //-----------------
// package main

// import (
// 	"fmt"
// 	"io/ioutil"
// 	"path/filepath"
// )

// // StorageMonitor surveille l'espace utilisé dans le dossier de stockage IPFS.
// func StorageMonitor(storagePath string, maxStorageSize int64) {
// 	for {
// 		usedSize, err := calculateFolderSize(storagePath)
// 		if err != nil {
// 			fmt.Println("Erreur lors du calcul de la taille du dossier:", err)
// 			continue
// 		}

// 		if usedSize > maxStorageSize {
// 			fmt.Println("Limite d'espace de stockage dépassée.")
// 			// Ajouter ici la logique pour gérer le dépassement de capacité
// 		}

// 		// Attente avant la prochaine vérification, par exemple 10 minutes
// 		// time.Sleep(10 * time.Minute)
// 	}
// }

// // calculateFolderSize calcule la taille totale des fichiers dans un dossier.
// func calculateFolderSize(path string) (int64, error) {
// 	var size int64
// 	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
// 		if err != nil {
// 			return err
// 		}
// 		size += info.Size()
// 		return nil
// 	})
// 	return size, err
// }

// func main() {
// 	// Exemple d'utilisation de StorageMonitor
// 	storagePath := "./IPFS_Storage" // Chemin vers le dossier de stockage IPFS
// 	maxStorageSize := 10 * 1024 * 1024 // 10 MB par exemple
// 	StorageMonitor(storagePath, maxStorageSize)
// }
// //-------------------------
