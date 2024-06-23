package fileregistry

type Message struct {
	OwnerPublicKey string
	Registry       []FileData
}
