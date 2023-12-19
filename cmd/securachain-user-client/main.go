package main

import (
	"log"
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/pierreleocadie/SecuraChain/pkg/aes"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

func main() {
	a := app.New()
	w := a.NewWindow("SecuraChain User Client")
	w.Resize(fyne.NewSize(800, 600))

	// Create an input to set file path with a browse button to save the ECDSA key pair
	ecdsaInput := widget.NewLabel("")
	ecdsaBrowseButton := widget.NewButton("Browse", func() {
		dialog := dialog.NewFolderOpen(func(dir fyne.ListableURI, err error) {
			if err == nil && (dir != nil) {
				u, er := url.Parse(dir.String())
				if er == nil {
					ecdsaInput.SetText(u.Path)
				}
			}
		}, w)
		dialog.Show()
	})

	// Create an input to set file path with a browse button to save the AES key
	aesInput := widget.NewLabel("")
	aesBrowseButton := widget.NewButton("Browse", func() {
		dialog := dialog.NewFolderOpen(func(dir fyne.ListableURI, err error) {
			if err == nil && (dir != nil) {
				u, er := url.Parse(dir.String())
				if er == nil {
					aesInput.SetText(u.Path)
				}
			}
		}, w)
		dialog.Show()
	})

	// Create a button to load an ECDSA key pair
	ecdsaButtonLoad := widget.NewButton("Load ECDSA Key Pair", func() {
		ecdsaKeyPair, err := ecdsa.LoadKeys("ecdsaPrivateKey", "ecdsaPublicKey", ecdsaInput.Text)
		if err != nil {
			log.Println(err)
			return
		}
		log.Println(ecdsaKeyPair)
	})

	// Create a button to generate a new ECDSA key pair
	ecdsaButton := widget.NewButton("Generate ECDSA Key Pair", func() {
		ecdsaKeyPair, err := ecdsa.NewECDSAKeyPair()
		if err != nil {
			return
		}
		err = ecdsaKeyPair.SaveKeys("ecdsaPrivateKey", "ecdsaPublicKey", ecdsaInput.Text)
		if err != nil {
			log.Println(err)
			return
		}
	})

	// Create a button to load an AES key
	var aesKey aes.Key
	aesButtonLoad := widget.NewButton("Load AES Key", func() {
		aesKeyL, err := aes.LoadKey("aesKey", aesInput.Text)
		if err != nil {
			log.Println(err)
			return
		}
		aesKey = aesKeyL
		log.Println(aesKey)
	})

	// Create a button to generate a new AES key
	aesButton := widget.NewButton("Generate AES Key", func() {
		aesKey, err := aes.NewAESKey()
		if err != nil {
			return
		}
		err = aesKey.SaveKey("aesKey", aesInput.Text)
		if err != nil {
			log.Println(err)
			return
		}
	})

	// Create a new button to select a file
	selectedFileLabel := widget.NewLabel("")
	selectFileButton := widget.NewButton("Select File", func() {
		dialog := dialog.NewFileOpen(func(file fyne.URIReadCloser, err error) {
			if err == nil && (file != nil) {
				selectedFileLabel.SetText(file.URI().Path())
			}
		}, w)
		dialog.Show()
	})

	// create a new button to send a file over the network
	/* sendFileButton := widget.NewButton("Send File", func() {
		// TODO
		// 1. Encrypt the file with AES
		if aesKey == nil {
			log.Println("AES key not loaded")
			return
		}

		encryptedFilePath := selectedFileLabel.Text + ".enc"
		err := aesKey.EncryptFile(selectedFileLabel.Text, encryptedFilePath)
		if err != nil {
			log.Println(err)
			return
		}

		// 2. Compute the checksum of the encrypted file
		// 3. Get the size of the encrypted file
		// 4. Get the name of the encrypted file
		// 5. Get the extension of the encrypted file
		// 6. Create a new ClientAnnouncement
	}) */

	hBoxECDSA := container.New(
		layout.NewHBoxLayout(),
		ecdsaInput,
		layout.NewSpacer(),
		ecdsaBrowseButton,
	)
	hBoxAES := container.New(
		layout.NewHBoxLayout(),
		aesInput,
		layout.NewSpacer(),
		aesBrowseButton,
	)
	hBoxSelectFile := container.New(
		layout.NewHBoxLayout(),
		selectFileButton,
		selectedFileLabel,
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
	)

	w.SetContent(vBox)
	w.ShowAndRun()
}
