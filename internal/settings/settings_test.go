package settings

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/plu-nz/app.lounge/internal/store"
)

func TestOldSettingsFollowTheGtkTheme(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"accent":"mauve","autoplayNext":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := store.OpenDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	m, err := Load(st)
	if err != nil {
		t.Fatal(err)
	}
	if got := m.Get(); got.Theme != "gtk" || got.Accent != "theme" {
		t.Fatalf("theme, accent = %q, %q; want gtk, theme", got.Theme, got.Accent)
	}
}

func TestCacheLimitDefaultsToTenAndKeepsUnlimited(t *testing.T) {
	dir := t.TempDir()
	st, _ := store.OpenDir(dir)
	m, err := Load(st)
	if err != nil {
		t.Fatal(err)
	}
	if got := m.Get().CacheLimitGB; got != 10 {
		t.Fatalf("default cache limit = %d, want 10", got)
	}
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"theme":"gtk","cacheLimitGB":0}`), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err = Load(st)
	if err != nil {
		t.Fatal(err)
	}
	if got := m.Get().CacheLimitGB; got != 0 {
		t.Fatalf("explicit unlimited = %d, want 0", got)
	}
}

func TestChosenThemeIsKept(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"theme":"lounge","accent":"mauve"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	st, _ := store.OpenDir(dir)
	m, err := Load(st)
	if err != nil {
		t.Fatal(err)
	}
	if got := m.Get(); got.Theme != "lounge" || got.Accent != "mauve" {
		t.Fatalf("theme, accent = %q, %q; want lounge, mauve", got.Theme, got.Accent)
	}
}
