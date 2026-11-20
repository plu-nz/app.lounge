package main

import (
	"context"
	"log"
	"time"

	"github.com/plu-nz/app.lounge/internal/library"
	"github.com/plu-nz/app.lounge/internal/settings"
	"github.com/plu-nz/app.lounge/internal/stremio"
)

func timeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}

// ContentService serves catalogs, details, streams, and subtitles.
type ContentService struct {
	content *stremio.Content
}

// Rows lists catalog rows; typ is "", "movie", or "series".
func (s *ContentService) Rows(typ string) []stremio.Row { return s.content.Rows(typ) }

// Catalog loads one catalog page.
func (s *ContentService) Catalog(addonID, typ, catalogID string, extra map[string]string) ([]stremio.MetaPreview, error) {
	ctx, cancel := timeout(25 * time.Second)
	defer cancel()
	return s.content.Catalog(ctx, addonID, typ, catalogID, extra)
}

// Search queries every searchable catalog.
func (s *ContentService) Search(query string) []stremio.SearchRow {
	ctx, cancel := timeout(25 * time.Second)
	defer cancel()
	return s.content.Search(ctx, query)
}

// Meta returns the full description of an item.
func (s *ContentService) Meta(typ, id string) (*stremio.Meta, error) {
	ctx, cancel := timeout(25 * time.Second)
	defer cancel()
	return s.content.Meta(ctx, typ, id)
}

// Streams lists sources for a video, grouped by addon.
func (s *ContentService) Streams(typ, id string) []stremio.StreamGroup {
	ctx, cancel := timeout(40 * time.Second)
	defer cancel()
	return s.content.Streams(ctx, typ, id)
}

// Subtitles lists external subtitles for a video.
func (s *ContentService) Subtitles(typ, id string) []stremio.Subtitle {
	ctx, cancel := timeout(20 * time.Second)
	defer cancel()
	return s.content.Subtitles(ctx, typ, id, nil)
}

// AddonService manages installed addons.
type AddonService struct {
	addons *stremio.Addons
}

// List returns installed addons in order.
func (s *AddonService) List() []stremio.Addon { return s.addons.List() }

// Managed reports whether a Stremio account owns the addon list.
func (s *AddonService) Managed() bool { return s.addons.Managed() }

// Install adds an addon from its manifest URL.
func (s *AddonService) Install(url string) (stremio.Addon, error) {
	ctx, cancel := timeout(20 * time.Second)
	defer cancel()
	return s.addons.Install(ctx, url)
}

// Remove uninstalls an addon.
func (s *AddonService) Remove(id string) error { return s.addons.Remove(id) }

// SetEnabled turns an addon on or off.
func (s *AddonService) SetEnabled(id string, enabled bool) error {
	return s.addons.SetEnabled(id, enabled)
}

// Move shifts an addon in the order.
func (s *AddonService) Move(id string, delta int) error { return s.addons.Move(id, delta) }

// Refresh reloads every manifest.
func (s *AddonService) Refresh() error {
	ctx, cancel := timeout(60 * time.Second)
	defer cancel()
	return s.addons.Refresh(ctx)
}

// LibraryService stores watch progress.
type LibraryService struct {
	library *library.Library
}

// ContinueWatching lists unfinished titles.
func (s *LibraryService) ContinueWatching() []library.Entry { return s.library.ContinueWatching() }

// Get returns progress for a meta ID, or nil.
func (s *LibraryService) Get(id string) *library.Entry {
	if e, ok := s.library.Get(id); ok {
		return &e
	}
	return nil
}

// Save records progress.
func (s *LibraryService) Save(e library.Entry) error { return s.library.Save(e) }

// Remove drops a title from continue watching.
func (s *LibraryService) Remove(id string) error { return s.library.Remove(id) }

// SetWatched marks a video as watched or not.
func (s *LibraryService) SetWatched(videoID string, watched bool) error {
	return s.library.SetWatched(videoID, watched)
}

// Watched returns which of the given videos were watched.
func (s *LibraryService) Watched(videoIDs []string) []string { return s.library.Watched(videoIDs) }

// SettingsService stores preferences.
type SettingsService struct {
	settings *settings.Manager
}

// Get returns the preferences.
func (s *SettingsService) Get() settings.Settings { return s.settings.Get() }

// Set saves the preferences.
func (s *SettingsService) Set(v settings.Settings) error { return s.settings.Set(v) }

// AppService exposes app-level helpers to the UI.
type AppService struct {
	debug bool
}

// Debug reports whether LOUNGE_DEBUG is set.
func (s *AppService) Debug() bool { return s.debug }

// Log writes a UI diagnostic line to the app log when debugging.
func (s *AppService) Log(message string) {
	if s.debug {
		log.Printf("ui: %s", message)
	}
}
