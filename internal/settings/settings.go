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
	// SubtitleMode is "on" (use the preferred language), "off", or "auto"
	// (only when the audio is not in the preferred language). Empty means "on".
	SubtitleMode string `json:"subtitleMode"`
	// SubtitleScale multiplies the subtitle size.
	SubtitleScale float64 `json:"subtitleScale"`
	// SubtitleColor, SubtitleBackground and SubtitlePosition are presets
	// understood by internal/subtitle; empty means the default.
	SubtitleColor      string `json:"subtitleColor"`
	SubtitleBackground string `json:"subtitleBackground"`
	SubtitlePosition   string `json:"subtitlePosition"`
	// AutoplayNext starts the next episode when one ends.
	AutoplayNext bool `json:"autoplayNext"`
	// SkipSegments is how intros, recaps, and credits are skipped: "off",
	// "ask" (a button), or "auto".
	SkipSegments string `json:"skipSegments"`
	// Theme is "gtk" to follow the desktop's GTK theme or "lounge" for the
	// built-in Tinta colors. Empty means "gtk".
	Theme string `json:"theme"`
	// Accent is "theme" for the GTK theme's accent, or one of the
	// Tinta accents.
	Accent string `json:"accent"`
	// CacheLimitGB caps the torrent cache in gigabytes; 0 means unlimited.
	CacheLimitGB int `json:"cacheLimitGB"`
	// ClearCacheOnExit deletes the torrent cache when the app closes.
	ClearCacheOnExit bool `json:"clearCacheOnExit"`
	// AudioOutput is "stereo" (downmix, default), "auto" (multichannel) or
	// "passthrough" (bitstream over HDMI). Empty means "stereo".
	AudioOutput string `json:"audioOutput"`
	// NightMode compresses the audio's dynamic range.
	NightMode bool `json:"nightMode"`
	// AudioDevice is an mpv audio device name; empty is the system default.
	AudioDevice string `json:"audioDevice"`
	// HDR is "tonemap" (HDR to SDR, default) or "off". Empty means "tonemap".
	HDR string `json:"hdr"`
}

// Defaults returns the first-run preferences.
func Defaults() Settings {
	return Settings{
		AudioLanguages:     []string{},
		SubtitleLanguages:  []string{},
		SubtitleScale:      1,
		SubtitleColor:      "white",
		SubtitleBackground: "outline",
		SubtitlePosition:   "bottom",
		AutoplayNext:       true,
		SkipSegments:       "ask",
		Theme:              "gtk",
		Accent:             "theme",
		CacheLimitGB:       10,
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
	// A file without a theme keeps it empty, which marks older settings.
	m.s.Theme = ""
	if _, err := st.Load(doc, &m.s); err != nil {
		return nil, err
	}
	// Settings saved before the theme option existed picked an accent among
	// Lounge's own colors only; they now follow the GTK theme and its accent.
	if m.s.Theme == "" {
		m.s.Theme, m.s.Accent = "gtk", "theme"
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
