package main

import "github.com/plu-nz/app.lounge/internal/torrent"

// CacheService exposes the torrent cache to the Settings screen. Wails runs
// each call off the UI thread, so a slow disk scan never blocks the window.
type CacheService struct {
	torrents *torrent.Engine
}

// Size returns the bytes of torrent data on disk.
func (s *CacheService) Size() int64 {
	if s.torrents == nil {
		return 0
	}
	return s.torrents.CacheSize()
}

// Clear deletes cached torrent data except the torrent being played and
// returns the bytes freed.
func (s *CacheService) Clear() (int64, error) {
	if s.torrents == nil {
		return 0, nil
	}
	return s.torrents.ClearCache()
}
