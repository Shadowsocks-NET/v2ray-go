package policy

// DefaultManager is the implementation of the Manager.
type DefaultManager struct{}

// Type implements common.HasType.
func (DefaultManager) Type() interface{} {
	return ManagerType()
}

// ForLevel implements Manager.
func (DefaultManager) ForLevel(level uint32) Session {
	return SessionDefault()
}

// ForSystem implements Manager.
func (DefaultManager) ForSystem() System {
	return System{}
}

// Start implements common.Runnable.
func (DefaultManager) Start() error {
	return nil
}

// Close implements common.Closable.
func (DefaultManager) Close() error {
	return nil
}
