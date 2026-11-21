package main

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/plu-nz/app.lounge/internal/library"
	"github.com/plu-nz/app.lounge/internal/store"
	"github.com/plu-nz/app.lounge/internal/stremio"
)

const accountDoc = "account"

// AccountStatus is what the UI shows about the Stremio account.
type AccountStatus struct {
	SignedIn bool   `json:"signedIn"`
	Email    string `json:"email"`
	SyncedAt int64  `json:"syncedAt"`
	Error    string `json:"error,omitempty"`
	// Library sync: when it last succeeded, what it moved, and why it failed.
	LibrarySyncedAt int64  `json:"librarySyncedAt"`
	LibraryPulled   int    `json:"libraryPulled"`
	LibraryPushed   int    `json:"libraryPushed"`
	LibraryError    string `json:"libraryError,omitempty"`
}

// AccountService signs in to Stremio and mirrors the account's addons.
type AccountService struct {
	client *stremio.Client
	addons *stremio.Addons
	store  *store.Store
	// changed tells the UI that the addon list was replaced.
	changed func()
	// libraryChanged tells the UI that a sync updated the library.
	libraryChanged func()
	lib            *library.Library
	librarySync    libSync
	pushTimer      *time.Timer

	mu       sync.Mutex
	account  *stremio.Account
	syncedAt int64
	syncErr  string
}

func newAccountService(client *stremio.Client, addons *stremio.Addons, lib *library.Library, st *store.Store) (*AccountService, error) {
	s := &AccountService{client: client, addons: addons, store: st, lib: lib, changed: func() {}, libraryChanged: func() {}}
	lib.OnChange = s.pushSoon
	var acc stremio.Account
	ok, err := st.Load(accountDoc, &acc)
	if err != nil {
		return nil, err
	}
	if ok && acc.AuthKey != "" {
		s.account = &acc
		addons.SetManaged(true)
	}
	return s, nil
}

// Status reports the signed-in account, if any.
func (s *AccountService) Status() AccountStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.account == nil {
		return AccountStatus{}
	}
	return AccountStatus{
		SignedIn: true, Email: s.account.Email, SyncedAt: s.syncedAt, Error: s.syncErr,
		LibrarySyncedAt: s.librarySync.at, LibraryPulled: s.librarySync.pulled,
		LibraryPushed: s.librarySync.pushed, LibraryError: s.librarySync.err,
	}
}

// Login signs in with email and password, then syncs addons.
func (s *AccountService) Login(email, password string) (AccountStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	acc, err := s.client.Login(ctx, email, password)
	if err != nil {
		return AccountStatus{}, err
	}
	return s.signIn(ctx, acc)
}

// CreateLink starts the sign-in that the user confirms on another device.
func (s *AccountService) CreateLink() (stremio.LinkCode, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return s.client.CreateLink(ctx)
}

// CheckLink completes the code sign-in once the user confirms it. It
// returns signedIn=false while the code is still waiting.
func (s *AccountService) CheckLink(code string) (AccountStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	key, err := s.client.ReadLink(ctx, code)
	if errors.Is(err, stremio.ErrLinkPending) {
		return AccountStatus{}, nil
	}
	if err != nil {
		return AccountStatus{}, err
	}
	acc, err := s.client.User(ctx, key)
	if err != nil {
		// The email is only for display; the key is what matters.
		log.Printf("stremio user: %v", err)
		acc = stremio.Account{AuthKey: key}
	}
	return s.signIn(ctx, acc)
}

// Logout forgets the account. Installed addons stay as they are and
// become editable again.
func (s *AccountService) Logout() error {
	s.mu.Lock()
	acc := s.account
	s.account = nil
	s.syncedAt, s.syncErr = 0, ""
	s.librarySync = libSync{}
	if s.pushTimer != nil {
		s.pushTimer.Stop()
		s.pushTimer = nil
	}
	s.mu.Unlock()
	s.addons.SetManaged(false)
	if acc != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.client.Logout(ctx, acc.AuthKey); err != nil {
			log.Printf("stremio logout: %v", err)
		}
	}
	return s.store.Delete(accountDoc)
}

// Sync replaces the addon list with the account's collection.
func (s *AccountService) Sync() (AccountStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := s.sync(ctx)
	return s.Status(), err
}

func (s *AccountService) signIn(ctx context.Context, acc stremio.Account) (AccountStatus, error) {
	if err := s.store.Save(accountDoc, acc); err != nil {
		return AccountStatus{}, err
	}
	s.mu.Lock()
	s.account = &acc
	s.mu.Unlock()
	s.addons.SetManaged(true)
	err := s.sync(ctx)
	return s.Status(), err
}

func (s *AccountService) sync(ctx context.Context) error {
	s.mu.Lock()
	acc := s.account
	s.mu.Unlock()
	if acc == nil {
		return errors.New("not signed in to Stremio")
	}
	descs, err := s.client.AddonCollection(ctx, acc.AuthKey)
	if err == nil {
		err = s.addons.ReplaceFromAccount(descs)
	}
	s.mu.Lock()
	if err != nil {
		s.syncErr = err.Error()
	} else {
		s.syncErr = ""
		s.syncedAt = time.Now().UnixMilli()
	}
	s.mu.Unlock()
	if err == nil {
		s.changed()
	}
	// The library syncs on its own; its failure is reported separately.
	if lerr := s.syncLibrary(ctx); lerr != nil && err == nil {
		log.Printf("stremio library sync: %v", lerr)
	}
	return err
}
