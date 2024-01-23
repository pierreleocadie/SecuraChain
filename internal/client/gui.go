package client

import (
	"context"
	"net/url"

	ipfsLog "github.com/ipfs/go-log/v2"
	iface "github.com/ipfs/kubo/core/coreiface"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/pkg/aes"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func CreateECDSAWidgets(w fyne.Window, ecdsaInput *widget.Label, log *ipfsLog.ZapEventLogger, ecdsaKeyPair *ecdsa.KeyPair) (fyne.CanvasObject, *widget.Button, *widget.Button) { //nolint: lll
	ecdsaBrowseButton := BrowseButton(w, ecdsaInput, log)
	ecdsaButton := GenerateECDSAKeyPairButton(ecdsaInput, log)
	ecdsaButtonLoad := LoadECDSAButton(ecdsaKeyPair, ecdsaInput, log)

	hBoxECDSA := container.New(
		layout.NewHBoxLayout(),
		ecdsaInput,
		layout.NewSpacer(),
		ecdsaBrowseButton,
	)

	return hBoxECDSA, ecdsaButtonLoad, ecdsaButton
}

func CreateAESWidgets(w fyne.Window, aesInput *widget.Label, log *ipfsLog.ZapEventLogger, aesKey *aes.Key) (fyne.CanvasObject, *widget.Button, *widget.Button) { //nolint: lll
	aesBrowseButton := BrowseButton(w, aesInput, log)
	aesButton := GenerateAESKeyButton(aesInput, log)
	aesButtonLoad := LoadAESButton(aesKey, aesInput, log)

	hBoxAES := container.New(
		layout.NewHBoxLayout(),
		aesInput,
		layout.NewSpacer(),
		aesBrowseButton,
	)

	return hBoxAES, aesButtonLoad, aesButton
}

func CreateFileSelectionWidgets(w fyne.Window, selectedFileLabel *widget.Label, log *ipfsLog.ZapEventLogger) (fyne.CanvasObject, *widget.Button) {
	selectFileButton := SelectFileButton(w, selectedFileLabel, log)

	hBoxSelectFile := container.New(
		layout.NewHBoxLayout(),
		selectFileButton,
		selectedFileLabel,
	)

	return hBoxSelectFile, selectFileButton
}

func BrowseButton(w fyne.Window, inputLabel *widget.Label, log *ipfsLog.ZapEventLogger) *widget.Button {
	return widget.NewButton("Browse", func() {
		dialog := dialog.NewFolderOpen(func(dir fyne.ListableURI, err error) {
			if err == nil && (dir != nil) {
				u, er := url.Parse(dir.String())
				if er == nil {
					inputLabel.SetText(u.Path)
					log.Debugf("Key will be saved in : %v", u.Path)
				}
			}
		}, w)
		dialog.Show()
	})
}

func LoadECDSAButton(ecdsaKeyPair *ecdsa.KeyPair, ecdsaInputLabel *widget.Label, log *ipfsLog.ZapEventLogger) *widget.Button {
	return widget.NewButton("Load ECDSA Key Pair", func() {
		if ecdsaInputLabel.Text == "" {
			log.Debug("Please select a directory to save the ECDSA key pair")
			return
		}

		ecdsaKeyPairL, err := ecdsa.LoadKeys("ecdsaPrivateKey", "ecdsaPublicKey", ecdsaInputLabel.Text)
		if err != nil {
			log.Errorln("Error loading ECDSA key pair : ", err)
			return
		}
		*ecdsaKeyPair = ecdsaKeyPairL
		log.Debug("ECDSA key pair loaded")
	})
}

func LoadAESButton(aesKey *aes.Key, aesInputLabel *widget.Label, log *ipfsLog.ZapEventLogger) *widget.Button {
	return widget.NewButton("Load AES Key", func() {
		if aesInputLabel.Text == "" {
			log.Debug("Please select a directory to save the AES key")
			return
		}

		aesKeyL, err := aes.LoadKey("aesKey", aesInputLabel.Text)
		if err != nil {
			log.Errorln("Error loading AES key : ", err)
			return
		}
		*aesKey = aesKeyL
		log.Debug("AES key loaded")
	})
}

func GenerateECDSAKeyPairButton(ecdsaInputLabel *widget.Label, log *ipfsLog.ZapEventLogger) *widget.Button {
	return widget.NewButton("Generate ECDSA Key Pair", func() {
		if ecdsaInputLabel.Text == "" {
			log.Debug("Please select a directory to save the ECDSA key pair")
			return
		}

		ecdsaKeyPair, err := ecdsa.NewECDSAKeyPair()
		if err != nil {
			return
		}

		err = ecdsaKeyPair.SaveKeys("ecdsaPrivateKey", "ecdsaPublicKey", ecdsaInputLabel.Text)
		if err != nil {
			log.Errorln("Error saving ECDSA key pair : ", err)
			return
		}
	})
}

func GenerateAESKeyButton(aesInputLabel *widget.Label, log *ipfsLog.ZapEventLogger) *widget.Button {
	return widget.NewButton("Generate AES Key", func() {
		if aesInputLabel.Text == "" {
			log.Debug("Please select a directory to save the AES key")
			return
		}

		aesKey, err := aes.NewAESKey()
		if err != nil {
			log.Errorln("Error generating AES key : ", err)
			return
		}

		err = aesKey.SaveKey("aesKey", aesInputLabel.Text)
		if err != nil {
			log.Errorln("Error saving AES key : ", err)
			return
		}
	})
}

func SelectFileButton(w fyne.Window, selectedFileInputLabel *widget.Label, log *ipfsLog.ZapEventLogger) *widget.Button {
	return widget.NewButton("Select File", func() {
		dialog := dialog.NewFileOpen(func(file fyne.URIReadCloser, err error) {
			if err == nil && (file != nil) {
				selectedFileInputLabel.SetText(file.URI().Path())
				log.Debugf("File selected : %v", selectedFileInputLabel.Text)
			}
		}, w)
		dialog.Show()
	})
}

func SendFileButton(ctx context.Context, selectedFile *widget.Label,
	ecdsaKeyPair *ecdsa.KeyPair, aesKey *aes.Key, ipfsAPI iface.CoreAPI,
	clientAnnouncementChan chan *transaction.ClientAnnouncement,
	log *ipfsLog.ZapEventLogger) *widget.Button {

	return widget.NewButton("Send File", func() {
		if *ecdsaKeyPair == nil {
			log.Debug("Please generate or load an ECDSA key pair")
			return
		}

		if *aesKey == nil {
			log.Debug("Please generate or load an AES key")
			return
		}

		err := SendFile(ctx, selectedFile.Text, ecdsaKeyPair, aesKey, ipfsAPI, clientAnnouncementChan, log)
		if err != nil {
			log.Errorln("Error sending file : ", err)
			return
		}
	})
}
