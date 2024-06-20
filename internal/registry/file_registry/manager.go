package fileregistry

type FileRegistryManager interface {
	Save(registry FileRegistry) error
	Load(registry FileRegistry) (FileRegistry, error)
}
