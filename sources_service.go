package main

import (
	"github.com/plu-nz/app.lounge/internal/settings"
	"github.com/plu-nz/app.lounge/internal/sources"
	"github.com/plu-nz/app.lounge/internal/stremio"
)

// SourcesService picks a source for automatic source selection.
type SourcesService struct {
	settings *settings.Manager
}

// Pick returns the best source among the groups under the user's rules, or
// nil when automatic selection is off or no source passes them.
func (s *SourcesService) Pick(groups []stremio.StreamGroup) *sources.Ref {
	prefs := s.settings.Get()
	if !sources.Auto(prefs) {
		return nil
	}
	ref, ok := sources.Best(groups, sources.RulesFrom(prefs))
	if !ok {
		return nil
	}
	return &ref
}
