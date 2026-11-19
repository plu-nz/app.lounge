package stremio

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/plu-nz/app.lounge/internal/store"
)

// DefaultAddons are installed on first launch.
var DefaultAddons = []string{
	"https://v3-cinemeta.strem.io/manifest.json",
	"https://opensubtitles-v3.strem.io/manifest.json",
}

// Addon is an installed addon.
type Addon struct {
	TransportURL string   `json:"transportUrl"`
	Manifest     Manifest `json:"manifest"`
	Enabled      bool     `json:"enabled"`
}

// Addons manages the installed addon collection.
type Addons struct {
	client *Client
	store  *store.Store

	mu   sync.RWMutex
	list []Addon
}

const addonsDoc = "addons"

// LoadAddons reads the collection, installing the defaults on first run.
func LoadAddons(ctx context.Context, client *Client, st *store.Store) (*Addons, error) {
	a := &Addons{client: client, store: st}
	ok, err := st.Load(addonsDoc, &a.list)
	if err != nil {
		return nil, err
	}
	if !ok {
		for _, u := range DefaultAddons {
			if _, err := a.Install(ctx, u); err != nil {
				// Offline first runs still start; defaults can be added later.
				continue
			}
		}
	}
	return a, nil
}

// List returns a copy of the collection in order.
func (a *Addons) List() []Addon {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return slices.Clone(a.list)
}

// Enabled returns the enabled addons in order.
func (a *Addons) Enabled() []Addon {
	a.mu.RLock()
	defer a.mu.RUnlock()
	var out []Addon
	for _, ad := range a.list {
		if ad.Enabled {
			out = append(out, ad)
		}
	}
	return out
}

// Get returns an installed addon by manifest ID.
func (a *Addons) Get(id string) (Addon, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	i := a.index(id)
	if i < 0 {
		return Addon{}, false
	}
	return a.list[i], true
}

// Install fetches a manifest and adds or updates the addon.
func (a *Addons) Install(ctx context.Context, rawURL string) (Addon, error) {
	u, err := NormalizeURL(rawURL)
	if err != nil {
		return Addon{}, err
	}
	m, err := a.client.Manifest(ctx, u)
	if err != nil {
		return Addon{}, err
	}
	ad := Addon{TransportURL: u, Manifest: m, Enabled: true}
	a.mu.Lock()
	if i := a.index(m.ID); i >= 0 {
		ad.Enabled = a.list[i].Enabled
		a.list[i] = ad
	} else {
		a.list = append(a.list, ad)
	}
	err = a.saveLocked()
	a.mu.Unlock()
	return ad, err
}

// Remove uninstalls an addon.
func (a *Addons) Remove(id string) error {
	return a.update(id, func(i int) { a.list = slices.Delete(a.list, i, i+1) })
}

// SetEnabled turns an addon on or off without removing it.
func (a *Addons) SetEnabled(id string, enabled bool) error {
	return a.update(id, func(i int) { a.list[i].Enabled = enabled })
}

// Move shifts an addon up (negative delta) or down in the order.
func (a *Addons) Move(id string, delta int) error {
	return a.update(id, func(i int) {
		j := min(max(i+delta, 0), len(a.list)-1)
		ad := a.list[i]
		a.list = slices.Delete(a.list, i, i+1)
		a.list = slices.Insert(a.list, j, ad)
	})
}

// Refresh reloads every manifest, keeping the order and enabled state.
func (a *Addons) Refresh(ctx context.Context) error {
	var errs []error
	for _, ad := range a.List() {
		if _, err := a.Install(ctx, ad.TransportURL); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", ad.Manifest.Name, err))
		}
	}
	return errors.Join(errs...)
}

func (a *Addons) update(id string, fn func(i int)) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	i := a.index(id)
	if i < 0 {
		return fmt.Errorf("addon %q is not installed", id)
	}
	fn(i)
	return a.saveLocked()
}

func (a *Addons) index(id string) int {
	return slices.IndexFunc(a.list, func(ad Addon) bool { return ad.Manifest.ID == id })
}

func (a *Addons) saveLocked() error { return a.store.Save(addonsDoc, a.list) }
