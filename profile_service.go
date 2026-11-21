package main

import (
	"log"
	"sync"
	"time"

	"github.com/plu-nz/app.lounge/internal/library"
	"github.com/plu-nz/app.lounge/internal/settings"
	"github.com/plu-nz/app.lounge/internal/store"
	"github.com/plu-nz/app.lounge/internal/stremio"
)

// ProfileService lists, edits, and switches profiles. Switching points the
// shared store at the profile's directory and reloads every service that
// caches documents, so the app keeps running; the UI then reloads itself.
type ProfileService struct {
	profiles *store.Profiles
	library  *library.Library
	settings *settings.Manager
	addons   *stremio.Addons
	account  *AccountService
	// changed tells the UI that the active profile's data was replaced.
	changed func()

	mu     sync.Mutex
	chosen bool
}

// List returns every profile.
func (s *ProfileService) List() []store.Profile { return s.profiles.List() }

// Active returns the profile in use.
func (s *ProfileService) Active() store.Profile { return s.profiles.Active() }

// NeedsPicker reports whether the UI should ask who is watching: more than
// one profile exists and none was chosen since the app started.
func (s *ProfileService) NeedsPicker() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.chosen && len(s.profiles.List()) > 1
}

// Switch makes a profile the active one and reloads the services.
func (s *ProfileService) Switch(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.chosen = true
	if id == s.profiles.Active().ID {
		return nil
	}
	if err := s.profiles.Switch(id); err != nil {
		return err
	}
	if err := s.reload(); err != nil {
		return err
	}
	s.changed()
	return nil
}

// Create adds a profile without switching to it.
func (s *ProfileService) Create(name string) (store.Profile, error) {
	return s.profiles.Create(name)
}

// Rename changes a profile's name.
func (s *ProfileService) Rename(id, name string) error { return s.profiles.Rename(id, name) }

// Delete removes a profile and its data. It refuses the last profile and the
// one in use.
func (s *ProfileService) Delete(id string) error { return s.profiles.Delete(id) }

// reload re-reads every cached document from the newly active profile.
func (s *ProfileService) reload() error {
	if err := s.library.Reload(); err != nil {
		return err
	}
	if err := s.settings.Reload(); err != nil {
		return err
	}
	ctx, cancel := timeout(15 * time.Second)
	defer cancel()
	if err := s.addons.Reload(ctx); err != nil {
		return err
	}
	if err := s.account.reload(); err != nil {
		return err
	}
	if s.account.Status().SignedIn {
		go func() {
			if _, err := s.account.Sync(); err != nil {
				log.Printf("stremio sync: %v", err)
			}
		}()
	}
	return nil
}

// reload re-reads the signed-in account from the store, which may now point
// at another profile.
func (s *AccountService) reload() error {
	var acc stremio.Account
	ok, err := s.store.Load(accountDoc, &acc)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.account, s.syncedAt, s.syncErr = nil, 0, ""
	if ok && acc.AuthKey != "" {
		s.account = &acc
	}
	managed := s.account != nil
	s.mu.Unlock()
	s.addons.SetManaged(managed)
	return nil
}
