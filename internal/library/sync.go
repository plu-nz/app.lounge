package library

import (
	"context"
	"strconv"
	"strings"

	"github.com/plu-nz/app.lounge/internal/stremio"
)

// Remote is the account's library store.
type Remote interface {
	LibraryItems(ctx context.Context) ([]stremio.LibraryItem, error)
	PutLibraryItems(ctx context.Context, items []stremio.LibraryItem) error
}

// SyncResult counts what a sync changed.
type SyncResult struct {
	Pulled int `json:"pulled"`
	Pushed int `json:"pushed"`
}

const pushBatch = 100

// Sync merges the account's library into the local one and sends local
// progress the account does not have. The rule is last modified wins per
// title: newer remote progress is never overwritten, remote items are
// never deleted, and a local title the account has seen more recently is
// replaced. A failed push leaves the pulled data in place.
func (l *Library) Sync(ctx context.Context, r Remote) (SyncResult, error) {
	remote, err := r.LibraryItems(ctx)
	if err != nil {
		return SyncResult{}, err
	}
	var res SyncResult
	res.Pulled, err = l.merge(remote)
	if err != nil {
		return res, err
	}
	toPush := l.pending(remote)
	for len(toPush) > 0 {
		n := min(pushBatch, len(toPush))
		if err := r.PutLibraryItems(ctx, toPush[:n]); err != nil {
			return res, err
		}
		res.Pushed += n
		toPush = toPush[n:]
	}
	return res, nil
}

// merge applies remote items to the local library and returns how many
// titles it created or replaced.
func (l *Library) merge(remote []stremio.LibraryItem) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	changed := 0
	dirty := false
	for _, r := range remote {
		if r.ID == "" {
			continue
		}
		videoID := r.State.VideoID
		if videoID == "" {
			videoID = r.ID
		}
		mt := r.MTimeMillis()
		// Finished videos count as watched. Watched is only ever added to
		// here, so a video watched locally stays watched.
		finishedMovie := r.Type == "movie" && (r.State.TimesWatch > 0 || r.State.Flagged > 0)
		finishedEpisode := r.State.Duration > 0 && float64(r.State.TimeOffset)/float64(r.State.Duration) >= WatchedThreshold
		if _, ok := l.d.Watched[videoID]; !ok && (finishedMovie || finishedEpisode) {
			l.d.Watched[videoID] = mt
			dirty = true
		}
		// Same rule as stremio-core's continue watching.
		if (r.Removed && !r.Temp) || r.State.TimeOffset <= 0 {
			continue
		}
		if t, ok := l.d.Dismissed[r.ID]; ok && t >= mt {
			continue
		}
		local, ok := l.d.Entries[r.ID]
		if ok && local.UpdatedAt >= mt {
			continue
		}
		e := Entry{
			ID: r.ID, Type: r.Type, Name: r.Name, Poster: r.Poster,
			VideoID:   videoID,
			Position:  float64(r.State.TimeOffset) / 1000,
			Duration:  float64(r.State.Duration) / 1000,
			UpdatedAt: mt,
		}
		if ok && local.VideoID == videoID {
			// Keep what only Lounge knows about the same video.
			e.Background, e.Logo, e.StreamKey = local.Background, local.Logo, local.StreamKey
			e.Season, e.Episode, e.EpisodeTitle = local.Season, local.Episode, local.EpisodeTitle
			if e.Poster == "" {
				e.Poster = local.Poster
			}
		} else if r.Type == "series" {
			e.Season, e.Episode = parseEpisode(videoID)
		}
		l.d.Entries[r.ID] = e
		delete(l.d.Dismissed, r.ID)
		changed++
		dirty = true
	}
	if !dirty {
		return changed, nil
	}
	return changed, l.store.Save(doc, l.d)
}

// pending builds the items to send: local titles the account lacks or
// holds with older progress.
func (l *Library) pending(remote []stremio.LibraryItem) []stremio.LibraryItem {
	l.mu.Lock()
	defer l.mu.Unlock()
	byID := make(map[string]stremio.LibraryItem, len(remote))
	for _, r := range remote {
		byID[r.ID] = r
	}
	var out []stremio.LibraryItem
	for _, e := range l.d.Entries {
		r, ok := byID[e.ID]
		if ok && r.MTimeMillis() >= e.UpdatedAt {
			continue
		}
		out = append(out, toItem(e, r, ok))
	}
	return out
}

// toItem puts an entry into an account item, keeping the server's other
// fields. A title the account does not know becomes a temporary item, the
// same as when Stremio plays something that is not in the library.
func toItem(e Entry, base stremio.LibraryItem, exists bool) stremio.LibraryItem {
	it := base
	if !exists {
		it = stremio.LibraryItem{ID: e.ID, Removed: true, Temp: true, CTime: stremio.FormatTime(e.UpdatedAt)}
	}
	it.Name, it.Type = e.Name, e.Type
	if e.Poster != "" {
		it.Poster = e.Poster
	}
	if it.Removed {
		it.Temp = true // shows in continue watching without joining the library
	}
	it.MTime = stremio.FormatTime(e.UpdatedAt)
	it.State.LastWatched = it.MTime
	it.State.VideoID = e.VideoID
	it.State.Duration = int64(e.Duration * 1000)
	it.State.TimeOffset = int64(e.Position * 1000)
	if e.Fraction() >= WatchedThreshold && e.Type != "series" {
		// A finished movie is flagged watched with no resume point.
		it.State.TimeOffset = 0
		it.State.TimesWatch = max(it.State.TimesWatch, 1)
		it.State.Flagged = 1
	}
	return it
}

// parseEpisode reads season and episode from an ID like "tt0944947:1:2".
func parseEpisode(videoID string) (season, episode int) {
	parts := strings.Split(videoID, ":")
	if len(parts) < 3 {
		return 0, 0
	}
	s, err1 := strconv.Atoi(parts[len(parts)-2])
	e, err2 := strconv.Atoi(parts[len(parts)-1])
	if err1 != nil || err2 != nil {
		return 0, 0
	}
	return s, e
}
