package blockregistry

type BlockRegistryManager interface {
	Save(registry BlockRegistry) error
	Load(registry BlockRegistry) (BlockRegistry, error)
}
