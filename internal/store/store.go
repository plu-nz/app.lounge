// Package store keeps small JSON documents in the user's config directory.
package store

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// Store reads and writes named JSON documents in one directory.
type Store struct {
	dir string
	mu  sync.Mutex
}

// Open uses $XDG_CONFIG_HOME/<name>, creating it when needed.
func Open(name string) (*Store, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return OpenDir(filepath.Join(base, name))
}

// OpenDir uses an explicit directory, mainly for tests.
func OpenDir(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &Store{dir: dir}, nil
}

// Dir returns the directory the store writes to.
func (s *Store) Dir() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dir
}

// SetDir points the store at another directory, creating it when needed.
// Documents already loaded into memory are not reloaded; callers do that.
func (s *Store) SetDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	s.mu.Lock()
	s.dir = dir
	s.mu.Unlock()
	return nil
}

// Load decodes a document into v. A missing document leaves v unchanged and
// reports ok=false.
func (s *Store) Load(name string, v any) (ok bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.path(name))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, json.Unmarshal(data, v)
}

// Save writes a document atomically.
func (s *Store) Save(name string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tmp, err := os.CreateTemp(s.dir, name+".*.tmp")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), s.path(name))
}

func (s *Store) path(name string) string { return filepath.Join(s.dir, name+".json") }

// Delete removes a document; a missing document is not an error.
func (s *Store) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.Remove(s.path(name)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
