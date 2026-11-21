package settings

// Reload replaces the in-memory settings with the store's document, which may
// now point at another profile.
func (m *Manager) Reload() error {
	fresh, err := Load(m.store)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.s = fresh.s
	m.mu.Unlock()
	return nil
}
