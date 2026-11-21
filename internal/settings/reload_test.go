package settings

import (
	"testing"

	"github.com/plu-nz/app.lounge/internal/store"
)

func TestReloadFollowsProfileSwitch(t *testing.T) {
	p, err := store.OpenProfilesDir(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m, err := Load(p.Store())
	if err != nil {
		t.Fatal(err)
	}
	s := m.Get()
	s.Language = "pt"
	if err := m.Set(s); err != nil {
		t.Fatal(err)
	}
	other, _ := p.Create("Other")
	if err := p.Switch(other.ID); err != nil {
		t.Fatal(err)
	}
	if err := m.Reload(); err != nil {
		t.Fatal(err)
	}
	if got := m.Get().Language; got != "" {
		t.Fatalf("language = %q in a fresh profile", got)
	}
	if err := p.Switch(store.DefaultProfileID); err != nil {
		t.Fatal(err)
	}
	if err := m.Reload(); err != nil {
		t.Fatal(err)
	}
	if got := m.Get().Language; got != "pt" {
		t.Fatalf("language = %q after switching back", got)
	}
}
