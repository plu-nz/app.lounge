package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

// DefaultProfileID names the profile that adopts data from before profiles.
const DefaultProfileID = "default"

const (
	profilesDoc  = "profiles.json"
	profilesDir  = "profiles"
	avatarHues   = 8
	maxNameRunes = 24
)

var (
	// ErrLastProfile is returned when deleting the only profile.
	ErrLastProfile = errors.New("the last profile cannot be deleted")
	// ErrActiveProfile is returned when deleting the profile in use.
	ErrActiveProfile = errors.New("switch to another profile before deleting this one")
	// ErrUnknownProfile is returned for an ID that does not exist.
	ErrUnknownProfile = errors.New("unknown profile")
	// ErrProfileName is returned for an empty name.
	ErrProfileName = errors.New("a profile needs a name")
)

// Profile is one person's slice of the app: its own documents live in
// <root>/profiles/<ID>/.
type Profile struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Hue is the avatar color as a CSS hue angle.
	Hue int `json:"hue"`
}

type profilesIndex struct {
	Last     string    `json:"last"`
	Profiles []Profile `json:"profiles"`
}

// Profiles manages the profile list and points a Store at the active one.
type Profiles struct {
	root  string
	store *Store

	mu  sync.Mutex
	idx profilesIndex
}

// OpenProfiles uses $XDG_CONFIG_HOME/<name> as the root.
func OpenProfiles(name string) (*Profiles, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return OpenProfilesDir(filepath.Join(base, name))
}

// OpenProfilesDir resolves the profile list under root, migrating documents
// kept directly in root (the layout before profiles) into the default
// profile, then opens a Store on the last used profile.
func OpenProfilesDir(root string) (*Profiles, error) {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	p := &Profiles{root: root}
	ok, err := p.readIndex()
	if err != nil {
		return nil, err
	}
	if !ok {
		if err := p.migrate(); err != nil {
			return nil, err
		}
	}
	if !p.has(p.idx.Last) {
		p.idx.Last = p.idx.Profiles[0].ID
	}
	st, err := OpenDir(p.dirOf(p.idx.Last))
	if err != nil {
		return nil, err
	}
	p.store = st
	return p, nil
}

// Store is the shared store; it follows the active profile.
func (p *Profiles) Store() *Store { return p.store }

// Root returns the config directory holding the profile index.
func (p *Profiles) Root() string { return p.root }

// List returns the profiles in creation order.
func (p *Profiles) List() []Profile {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.idx.Profiles)
}

// Active returns the profile the store points at.
func (p *Profiles) Active() Profile {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.idx.Profiles[p.find(p.idx.Last)]
}

// Switch makes a profile the active one and remembers it for the next launch.
func (p *Profiles) Switch(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.has(id) {
		return ErrUnknownProfile
	}
	if err := p.store.SetDir(p.dirOf(id)); err != nil {
		return err
	}
	prev := p.idx.Last
	p.idx.Last = id
	if err := p.writeIndex(); err != nil {
		p.idx.Last = prev
		_ = p.store.SetDir(p.dirOf(prev))
		return err
	}
	return nil
}

// Create adds a profile with an avatar color that is not yet in use when
// possible. It does not switch to it.
func (p *Profiles) Create(name string) (Profile, error) {
	name, err := cleanName(name)
	if err != nil {
		return Profile{}, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	pr := Profile{ID: newID(), Name: name, Hue: p.nextHue()}
	if err := os.MkdirAll(p.dirOf(pr.ID), 0o700); err != nil {
		return Profile{}, err
	}
	p.idx.Profiles = append(p.idx.Profiles, pr)
	if err := p.writeIndex(); err != nil {
		p.idx.Profiles = p.idx.Profiles[:len(p.idx.Profiles)-1]
		return Profile{}, err
	}
	return pr, nil
}

// Rename changes a profile's display name.
func (p *Profiles) Rename(id, name string) error {
	name, err := cleanName(name)
	if err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	i := p.find(id)
	if i < 0 {
		return ErrUnknownProfile
	}
	old := p.idx.Profiles[i].Name
	p.idx.Profiles[i].Name = name
	if err := p.writeIndex(); err != nil {
		p.idx.Profiles[i].Name = old
		return err
	}
	return nil
}

// Delete removes a profile and all its data. The last profile and the active
// profile cannot be deleted.
func (p *Profiles) Delete(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	i := p.find(id)
	if i < 0 {
		return ErrUnknownProfile
	}
	if len(p.idx.Profiles) == 1 {
		return ErrLastProfile
	}
	if id == p.idx.Last {
		return ErrActiveProfile
	}
	removed := p.idx.Profiles[i]
	p.idx.Profiles = slices.Delete(slices.Clone(p.idx.Profiles), i, i+1)
	if err := p.writeIndex(); err != nil {
		p.idx.Profiles = slices.Insert(p.idx.Profiles, i, removed)
		return err
	}
	return os.RemoveAll(p.dirOf(id))
}

// migrate creates the default profile and moves the flat JSON documents of
// the old layout into it. Moving is a rename, so nothing is copied or lost,
// and a run interrupted halfway finishes on the next launch.
func (p *Profiles) migrate() error {
	dir := p.dirOf(DefaultProfileID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	entries, err := os.ReadDir(p.root)
	if err != nil {
		return err
	}
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() || n == profilesDoc || !strings.HasSuffix(n, ".json") {
			continue
		}
		if err := os.Rename(filepath.Join(p.root, n), filepath.Join(dir, n)); err != nil {
			return fmt.Errorf("migrate %s: %w", n, err)
		}
	}
	p.idx = profilesIndex{
		Last:     DefaultProfileID,
		Profiles: []Profile{{ID: DefaultProfileID, Name: "Default", Hue: 0}},
	}
	return p.writeIndex()
}

func (p *Profiles) readIndex() (bool, error) {
	data, err := os.ReadFile(filepath.Join(p.root, profilesDoc))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(data, &p.idx); err != nil {
		return false, err
	}
	// An index without profiles is unusable; rebuild it from the old layout.
	return len(p.idx.Profiles) > 0, nil
}

func (p *Profiles) writeIndex() error {
	data, err := json.MarshalIndent(p.idx, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(p.root, profilesDoc+".tmp")
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(p.root, profilesDoc))
}

func (p *Profiles) dirOf(id string) string { return filepath.Join(p.root, profilesDir, id) }

func (p *Profiles) find(id string) int {
	return slices.IndexFunc(p.idx.Profiles, func(pr Profile) bool { return pr.ID == id })
}

func (p *Profiles) has(id string) bool { return p.find(id) >= 0 }

// nextHue picks the first avatar hue no profile uses, cycling when all are.
func (p *Profiles) nextHue() int {
	step := 360 / avatarHues
	for i := 0; i < avatarHues; i++ {
		h := i * step
		if !slices.ContainsFunc(p.idx.Profiles, func(pr Profile) bool { return pr.Hue == h }) {
			return h
		}
	}
	return len(p.idx.Profiles) % avatarHues * step
}

func cleanName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrProfileName
	}
	if r := []rune(name); len(r) > maxNameRunes {
		name = string(r[:maxNameRunes])
	}
	return name, nil
}

func newID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
