// Package library tracks watch progress for continue watching.
package library

import (
	"cmp"
	"slices"
	"sync"
	"time"

	"github.com/plu-nz/app.lounge/internal/store"
)

// Entry is the latest progress on one title.
type Entry struct {
	ID           string  `json:"id"` // meta ID
	Type         string  `json:"type"`
	Name         string  `json:"name"`
	Poster       string  `json:"poster,omitempty"`
	Background   string  `json:"background,omitempty"`
	Logo         string  `json:"logo,omitempty"`
	VideoID      string  `json:"videoId"` // episode ID for series, meta ID for movies
	Season       int     `json:"season,omitempty"`
	Episode      int     `json:"episode,omitempty"`
	EpisodeTitle string  `json:"episodeTitle,omitempty"`
	Position     float64 `json:"position"`
	Duration     float64 `json:"duration"`
	UpdatedAt    int64   `json:"updatedAt"`
	// StreamKey identifies the last source so resume can prefer it.
	StreamKey string `json:"streamKey,omitempty"`
}

// Fraction returns how much of the video was watched.
func (e Entry) Fraction() float64 {
	if e.Duration <= 0 {
		return 0
	}
	return e.Position / e.Duration
}

// WatchedThreshold is the fraction at which a video counts as watched.
const WatchedThreshold = 0.92

type data struct {
	Entries map[string]Entry `json:"entries"`
	Watched map[string]int64 `json:"watched"` // video ID -> time
	// Dismissed holds the time each title left continue watching, so a
	// sync does not bring it back.
	Dismissed map[string]int64 `json:"dismissed,omitempty"`
}

// Library stores progress and watched videos.
type Library struct {
	store *store.Store
	mu    sync.Mutex
	d     data
	// OnChange, when set, runs after progress is saved locally.
	OnChange func()
}

const doc = "library"

// Load reads the library document.
func Load(st *store.Store) (*Library, error) {
	l := &Library{store: st, d: data{Entries: map[string]Entry{}, Watched: map[string]int64{}}}
	if _, err := st.Load(doc, &l.d); err != nil {
		return nil, err
	}
	if l.d.Entries == nil {
		l.d.Entries = map[string]Entry{}
	}
	if l.d.Watched == nil {
		l.d.Watched = map[string]int64{}
	}
	if l.d.Dismissed == nil {
		l.d.Dismissed = map[string]int64{}
	}
	return l, nil
}

// Save records progress. A video past the watched threshold is marked
// watched and leaves continue watching for movies.
func (l *Library) Save(e Entry) error {
	l.mu.Lock()
	e.UpdatedAt = time.Now().UnixMilli()
	if e.Fraction() >= WatchedThreshold {
		l.d.Watched[e.VideoID] = e.UpdatedAt
		e.Position = e.Duration
	}
	l.d.Entries[e.ID] = e
	delete(l.d.Dismissed, e.ID)
	err := l.store.Save(doc, l.d)
	l.mu.Unlock()
	if l.OnChange != nil {
		l.OnChange()
	}
	return err
}

// Get returns the entry for a meta ID.
func (l *Library) Get(id string) (Entry, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.d.Entries[id]
	return e, ok
}

// Remove drops a title from continue watching.
func (l *Library) Remove(id string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.d.Entries, id)
	l.d.Dismissed[id] = time.Now().UnixMilli()
	return l.store.Save(doc, l.d)
}

// SetWatched marks or unmarks a video as watched.
func (l *Library) SetWatched(videoID string, watched bool) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if watched {
		l.d.Watched[videoID] = time.Now().UnixMilli()
	} else {
		delete(l.d.Watched, videoID)
	}
	return l.store.Save(doc, l.d)
}

// Watched returns the watched video IDs among the given ones.
func (l *Library) Watched(videoIDs []string) []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []string
	for _, id := range videoIDs {
		if _, ok := l.d.Watched[id]; ok {
			out = append(out, id)
		}
	}
	return out
}

// ContinueWatching lists unfinished titles, most recent first. Finished
// series stay so the UI can offer the next episode.
func (l *Library) ContinueWatching() []Entry {
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []Entry
	for _, e := range l.d.Entries {
		finished := e.Fraction() >= WatchedThreshold
		if finished && e.Type != "series" {
			continue
		}
		if !finished && e.Position < 30 {
			continue
		}
		out = append(out, e)
	}
	slices.SortFunc(out, func(a, b Entry) int {
		if c := cmp.Compare(b.UpdatedAt, a.UpdatedAt); c != 0 {
			return c
		}
		return cmp.Compare(b.ID, a.ID) // stable order for saves in the same millisecond
	})
	return out
}
