package stremio

import "context"

// Reload replaces the collection with the one in the store, which may now
// point at another profile. A store without addons gets the defaults, like a
// first run. The account mirror flag is cleared; the caller sets it again
// when the profile has a signed-in account.
func (a *Addons) Reload(ctx context.Context) error {
	var list []Addon
	ok, err := a.store.Load(addonsDoc, &list)
	if err != nil {
		return err
	}
	a.mu.Lock()
	a.list, a.managed = list, false
	a.mu.Unlock()
	if !ok {
		for _, u := range DefaultAddons {
			// Offline first runs still work; defaults can be added later.
			_, _ = a.install(ctx, u)
		}
	}
	return nil
}
