package main

import (
	"github.com/plu-nz/app.lounge/internal/settings"
	"github.com/plu-nz/app.lounge/internal/tracks"
)

// TracksService picks audio and subtitle tracks and remembers manual choices.
type TracksService struct {
	settings  *settings.Manager
	overrides *tracks.Overrides
}

// Select returns the tracks to use for a title: its remembered choices first,
// then the global preferences.
func (s *TracksService) Select(titleID string, list []tracks.Track) tracks.Result {
	cfg := s.settings.Get()
	p := tracks.Prefs{SubMode: cfg.SubtitleMode}
	if len(cfg.AudioLanguages) > 0 {
		p.AudioLang = cfg.AudioLanguages[0]
	}
	if len(cfg.SubtitleLanguages) > 0 {
		p.SubLang = cfg.SubtitleLanguages[0]
	}
	return tracks.Select(list, p, s.overrides.Get(titleID))
}

// Remember stores a manual choice for a series, or a movie's title ID.
// kind is "audio" or "sub".
func (s *TracksService) Remember(titleID, kind string, choice tracks.Choice) error {
	return s.overrides.Remember(titleID, kind, choice)
}
