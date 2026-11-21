package tracks

import (
	"sync"

	"github.com/plu-nz/app.lounge/internal/store"
)

// Overrides stores manual track choices keyed by the series ID, or the title
// ID for movies.
type Overrides struct {
	store *store.Store
	mu    sync.Mutex
	d     map[string]Override
}

const doc = "tracks"

// LoadOverrides reads the overrides document.
func LoadOverrides(st *store.Store) (*Overrides, error) {
	o := &Overrides{store: st, d: map[string]Override{}}
	if _, err := st.Load(doc, &o.d); err != nil {
		return nil, err
	}
	if o.d == nil {
		o.d = map[string]Override{}
	}
	return o, nil
}

// Get returns the override for a title, or nil.
func (o *Overrides) Get(key string) *Override {
	o.mu.Lock()
	defer o.mu.Unlock()
	v, ok := o.d[key]
	if !ok {
		return nil
	}
	return &v
}

// Remember stores a manual choice. kind is "audio" or "sub".
func (o *Overrides) Remember(key, kind string, c Choice) error {
	if key == "" {
		return nil
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	v := o.d[key]
	switch kind {
	case "audio":
		v.Audio = &c
	case "sub":
		v.Sub = &c
	default:
		return nil
	}
	o.d[key] = v
	return o.store.Save(doc, o.d)
}

// Reload replaces the in-memory overrides with the store's document, which may
// now point at another profile.
func (o *Overrides) Reload() error {
	d := map[string]Override{}
	if _, err := o.store.Load(doc, &d); err != nil {
		return err
	}
	if d == nil {
		d = map[string]Override{}
	}
	o.mu.Lock()
	o.d = d
	o.mu.Unlock()
	return nil
}
