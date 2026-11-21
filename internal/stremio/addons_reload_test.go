package stremio

import (
	"context"
	"testing"

	"github.com/plu-nz/app.lounge/internal/store"
)

func TestReloadFollowsProfileSwitch(t *testing.T) {
	p, err := store.OpenProfilesDir(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	st := p.Store()
	if err := st.Save(addonsDoc, []Addon{{Manifest: Manifest{ID: "mine"}, Enabled: true}}); err != nil {
		t.Fatal(err)
	}
	other, _ := p.Create("Other")
	a := &Addons{store: st, list: []Addon{{Manifest: Manifest{ID: "mine"}, Enabled: true}}, managed: true}

	if err := p.Switch(other.ID); err != nil {
		t.Fatal(err)
	}
	// An empty profile would install defaults over the network; give it a list.
	if err := st.Save(addonsDoc, []Addon{{Manifest: Manifest{ID: "theirs"}}}); err != nil {
		t.Fatal(err)
	}
	if err := a.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := a.List(); len(got) != 1 || got[0].Manifest.ID != "theirs" || a.Managed() {
		t.Fatalf("after switch: %+v managed=%v", got, a.Managed())
	}
	if err := p.Switch(store.DefaultProfileID); err != nil {
		t.Fatal(err)
	}
	if err := a.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := a.List(); len(got) != 1 || got[0].Manifest.ID != "mine" {
		t.Fatalf("after switching back: %+v", got)
	}
}
