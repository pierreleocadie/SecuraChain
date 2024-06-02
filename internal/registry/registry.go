package registry

type Registeries interface {
	IndexingRegistry | FileRegistry | BlockRegistry | BlockData | RegistryMessage
}
