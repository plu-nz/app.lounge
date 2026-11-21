package library

// Reload replaces the in-memory library with the store's document, which may
// now point at another profile.
func (l *Library) Reload() error {
	fresh, err := Load(l.store)
	if err != nil {
		return err
	}
	l.mu.Lock()
	l.d = fresh.d
	l.mu.Unlock()
	return nil
}
