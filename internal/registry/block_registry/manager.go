package blockregistry

type BlockRegistryManager interface {
	Save(registry BlockRegistry) error
	Load() (BlockRegistry, error)
}
