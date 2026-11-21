package store

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestMigrationMovesFlatDataIntoDefault(t *testing.T) {
	root := t.TempDir()
	docs := map[string]string{
		"settings.json": `{"language":"pt"}`,
		"library.json":  `{"entries":{}}`,
		"addons.json":   `[]`,
		"account.json":  `{"authKey":"k"}`,
	}
	for n, c := range docs {
		write(t, filepath.Join(root, n), c)
	}
	p, err := OpenProfilesDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := p.Active().ID; got != DefaultProfileID {
		t.Fatalf("active = %q", got)
	}
	for n, c := range docs {
		got, err := os.ReadFile(filepath.Join(root, "profiles", DefaultProfileID, n))
		if err != nil || string(got) != c {
			t.Fatalf("%s: %q, %v", n, got, err)
		}
		if _, err := os.Stat(filepath.Join(root, n)); !os.IsNotExist(err) {
			t.Fatalf("%s left behind", n)
		}
	}
	var s struct{ Language string }
	if ok, err := p.Store().Load("settings", &s); !ok || err != nil || s.Language != "pt" {
		t.Fatalf("store sees %v %v %+v", ok, err, s)
	}
	// A second launch does not migrate again or lose anything.
	p2, err := OpenProfilesDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(p2.List()) != 1 {
		t.Fatalf("profiles = %v", p2.List())
	}
	if ok, _ := p2.Store().Load("settings", &s); !ok {
		t.Fatal("settings lost on second launch")
	}
}

func TestMigrationResumesAfterInterruption(t *testing.T) {
	root := t.TempDir()
	// A first run moved one file but died before writing the index.
	write(t, filepath.Join(root, "profiles", DefaultProfileID, "library.json"), `{}`)
	write(t, filepath.Join(root, "settings.json"), `{}`)
	p, err := OpenProfilesDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"library", "settings"} {
		if ok, _ := p.Store().Load(n, &struct{}{}); !ok {
			t.Fatalf("%s missing", n)
		}
	}
}

func TestFreshInstallGetsDefaultProfile(t *testing.T) {
	p, err := OpenProfilesDir(filepath.Join(t.TempDir(), "lounge"))
	if err != nil {
		t.Fatal(err)
	}
	if l := p.List(); len(l) != 1 || l[0].ID != DefaultProfileID {
		t.Fatalf("profiles = %v", l)
	}
}

func TestSwitchIsolatesDataAndRemembersLast(t *testing.T) {
	root := t.TempDir()
	p, _ := OpenProfilesDir(root)
	if err := p.Store().Save("settings", map[string]string{"who": "default"}); err != nil {
		t.Fatal(err)
	}
	kid, err := p.Create("Kid")
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Switch(kid.ID); err != nil {
		t.Fatal(err)
	}
	var got map[string]string
	if ok, _ := p.Store().Load("settings", &got); ok {
		t.Fatalf("new profile sees %v", got)
	}
	if err := p.Store().Save("settings", map[string]string{"who": "kid"}); err != nil {
		t.Fatal(err)
	}
	if err := p.Switch(DefaultProfileID); err != nil {
		t.Fatal(err)
	}
	p.Store().Load("settings", &got)
	if got["who"] != "default" {
		t.Fatalf("default profile sees %v", got)
	}
	if err := p.Switch(kid.ID); err != nil {
		t.Fatal(err)
	}
	reopened, _ := OpenProfilesDir(root)
	if reopened.Active().ID != kid.ID {
		t.Fatalf("last = %q, want %q", reopened.Active().ID, kid.ID)
	}
	if err := p.Switch("nope"); !errors.Is(err, ErrUnknownProfile) {
		t.Fatalf("err = %v", err)
	}
}

func TestCreateRenameDelete(t *testing.T) {
	p, _ := OpenProfilesDir(t.TempDir())
	if _, err := p.Create("  "); !errors.Is(err, ErrProfileName) {
		t.Fatalf("err = %v", err)
	}
	a, _ := p.Create("Ana")
	b, _ := p.Create("Bia")
	if a.Hue == b.Hue || a.Hue == 0 {
		t.Fatalf("hues %d %d should differ from each other and the default", a.Hue, b.Hue)
	}
	if err := p.Rename(a.ID, " Ana Maria "); err != nil {
		t.Fatal(err)
	}
	if err := p.Delete(DefaultProfileID); !errors.Is(err, ErrActiveProfile) {
		t.Fatalf("active delete err = %v", err)
	}
	if err := p.Switch(a.ID); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(p.Root(), "profiles", b.ID, "x.json"), `{}`)
	if err := p.Delete(b.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(p.Root(), "profiles", b.ID)); !os.IsNotExist(err) {
		t.Fatal("profile data not removed")
	}
	if err := p.Delete(DefaultProfileID); err != nil {
		t.Fatal(err)
	}
	if err := p.Delete(a.ID); !errors.Is(err, ErrLastProfile) {
		t.Fatalf("last delete err = %v", err)
	}
	if l := p.List(); len(l) != 1 || l[0].Name != "Ana Maria" {
		t.Fatalf("profiles = %v", l)
	}
}
