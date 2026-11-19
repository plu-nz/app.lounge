// Package settings stores user preferences.
package settings

import (
	"sync"

	"github.com/plu-nz/app.lounge/internal/store"
)

// Settings are the user's preferences.
type Settings struct {
	// Language is the UI language (en, pt, es); empty follows the system.
	Language string `json:"language"`
	// AudioLanguages and SubtitleLanguages are ISO 639-2 codes in priority order.
	AudioLanguages    []string `json:"audioLanguages"`
	SubtitleLanguages []string `json:"subtitleLanguages"`
	// SubtitleScale multiplies the subtitle size.
	SubtitleScale float64 `json:"subtitleScale"`
	// AutoplayNext starts the next episode when one ends.
	AutoplayNext bool `json:"autoplayNext"`
	// Accent selects an approved Darkppuccin accent.
	Accent string `json:"accent"`
}

// Defaults returns the first-run preferences.
func Defaults() Settings {
	return Settings{
		AudioLanguages:    []string{},
		SubtitleLanguages: []string{},
		SubtitleScale:     1,
		AutoplayNext:      true,
		Accent:            "blue",
	}
}

// Manager loads and saves settings.
type Manager struct {
	store *store.Store
	mu    sync.Mutex
	s     Settings
}

const doc = "settings"

// Load reads settings, falling back to defaults.
func Load(st *store.Store) (*Manager, error) {
	m := &Manager{store: st, s: Defaults()}
	if _, err := st.Load(doc, &m.s); err != nil {
		return nil, err
	}
	return m, nil
}

// Get returns the current settings.
func (m *Manager) Get() Settings {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.s
}

// Set replaces and saves the settings.
func (m *Manager) Set(s Settings) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.s = s
	return m.store.Save(doc, s)
}
