package torrent

import (
	"log"
	"time"
)

// CachePolicy is the user's cache preference, read each time it applies.
type CachePolicy struct {
	// LimitBytes caps the cache; 0 or less means unlimited.
	LimitBytes int64
	// ClearOnExit deletes all cached data when the app closes.
	ClearOnExit bool
}

// SetCachePolicy installs the source of the cache policy and starts the
// background eviction check. Call it once, right after Start.
func (e *Engine) SetCachePolicy(policy func() CachePolicy) {
	e.policy = policy
	e.evictStop = make(chan struct{})
	go func() {
		e.Evict()
		tick := time.NewTicker(time.Minute)
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				// A playing torrent keeps growing, so check periodically.
				e.Evict()
			case <-e.evictStop:
				return
			}
		}
	}()
}

func (e *Engine) cachePolicy() CachePolicy {
	if e.policy == nil {
		return CachePolicy{}
	}
	return e.policy()
}

// protectedHashes are the torrents in use, which are never evicted.
func (e *Engine) protectedHashes() map[string]bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	p := make(map[string]bool, len(e.active))
	for h := range e.active {
		p[h.HexString()] = true
	}
	return p
}

// Evict deletes least recently used torrent data until the cache fits its
// limit. Torrents in use are skipped.
func (e *Engine) Evict() {
	limit := e.cachePolicy().LimitBytes
	if limit <= 0 {
		return
	}
	entries, err := ScanCache(e.dataDir)
	if err != nil {
		log.Printf("torrent cache scan: %v", err)
		return
	}
	for _, h := range PlanEviction(entries, limit, e.protectedHashes()) {
		if err := RemoveEntry(e.dataDir, h); err != nil {
			log.Printf("torrent cache evict: %v", err)
		}
	}
}

// CacheSize returns the bytes of torrent data on disk.
func (e *Engine) CacheSize() int64 {
	entries, _ := ScanCache(e.dataDir)
	var n int64
	for _, en := range entries {
		n += en.Size
	}
	return n
}

// ClearCache deletes all torrent data except torrents in use and returns the
// bytes freed.
func (e *Engine) ClearCache() (int64, error) {
	entries, err := ScanCache(e.dataDir)
	if err != nil {
		return 0, err
	}
	protected := e.protectedHashes()
	var freed int64
	var firstErr error
	for _, en := range entries {
		if protected[en.Hash] {
			continue
		}
		if err := RemoveEntry(e.dataDir, en.Hash); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		freed += en.Size
	}
	return freed, firstErr
}
