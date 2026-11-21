package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/plu-nz/app.lounge/internal/stremio"
)

// pushDelay waits for playback to settle before progress goes to the
// account, so one viewing is a single sync, not one per progress save.
const pushDelay = 20 * time.Second

// libSync is the outcome of the last library sync.
type libSync struct {
	at     int64
	pulled int
	pushed int
	err    string
}

// remoteLibrary adapts the Stremio client to library.Remote for one
// signed-in account.
type remoteLibrary struct {
	client  *stremio.Client
	authKey string
}

func (r remoteLibrary) LibraryItems(ctx context.Context) ([]stremio.LibraryItem, error) {
	return r.client.LibraryItems(ctx, r.authKey)
}

func (r remoteLibrary) PutLibraryItems(ctx context.Context, items []stremio.LibraryItem) error {
	return r.client.PutLibraryItems(ctx, r.authKey, items)
}

var libraryMu sync.Mutex // one library sync at a time

// syncLibrary merges the account's library with the local one and sends
// local progress back. Nothing happens while signed out.
func (s *AccountService) syncLibrary(ctx context.Context) error {
	s.mu.Lock()
	acc := s.account
	s.mu.Unlock()
	if acc == nil {
		return nil
	}
	libraryMu.Lock()
	res, err := s.lib.Sync(ctx, remoteLibrary{s.client, acc.AuthKey})
	libraryMu.Unlock()
	s.mu.Lock()
	if s.account == nil || s.account.AuthKey != acc.AuthKey {
		// Signed out while syncing: report nothing.
		s.mu.Unlock()
		return nil
	}
	if err != nil {
		s.librarySync.err = err.Error()
	} else {
		s.librarySync = libSync{at: time.Now().UnixMilli(), pulled: res.Pulled, pushed: res.Pushed}
	}
	s.mu.Unlock()
	if err == nil && res.Pulled > 0 {
		s.libraryChanged()
	}
	return err
}

// pushSoon schedules a library sync after the last local change. It is a
// no-op while signed out.
func (s *AccountService) pushSoon() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.account == nil {
		return
	}
	if s.pushTimer != nil {
		s.pushTimer.Stop()
	}
	s.pushTimer = time.AfterFunc(pushDelay, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := s.syncLibrary(ctx); err != nil {
			log.Printf("stremio library sync: %v", err)
		}
	})
}
