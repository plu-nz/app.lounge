package library

import (
	"testing"

	"github.com/plu-nz/app.lounge/internal/store"
)

func TestReloadFollowsProfileSwitch(t *testing.T) {
	p, err := store.OpenProfilesDir(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	l, err := Load(p.Store())
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Save(Entry{ID: "tt1", VideoID: "tt1", Position: 10, Duration: 100}); err != nil {
		t.Fatal(err)
	}
	other, _ := p.Create("Other")
	if err := p.Switch(other.ID); err != nil {
		t.Fatal(err)
	}
	if err := l.Reload(); err != nil {
		t.Fatal(err)
	}
	if _, ok := l.Get("tt1"); ok {
		t.Fatal("progress leaked into the other profile")
	}
	if err := l.Save(Entry{ID: "tt2", VideoID: "tt2", Position: 5, Duration: 100}); err != nil {
		t.Fatal(err)
	}
	if err := p.Switch(store.DefaultProfileID); err != nil {
		t.Fatal(err)
	}
	if err := l.Reload(); err != nil {
		t.Fatal(err)
	}
	if _, ok := l.Get("tt1"); !ok {
		t.Fatal("original progress lost")
	}
	if _, ok := l.Get("tt2"); ok {
		t.Fatal("other profile's progress leaked back")
	}
}
