package torrent

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// CacheEntry is one torrent's data directory in the cache.
type CacheEntry struct {
	Hash     string // directory name, a lowercase hex info hash
	Size     int64
	LastUsed time.Time
}

// hashDir matches the directories the engine creates: one per info hash.
var hashDir = regexp.MustCompile(`^[0-9a-f]{40}$`)

// PlanEviction picks the entries to delete, least recently used first, until
// the remaining total fits under limit. A limit <= 0 means unlimited. Entries
// whose hash is protected are never picked, even when that leaves the cache
// over the limit.
func PlanEviction(entries []CacheEntry, limit int64, protected map[string]bool) []string {
	if limit <= 0 {
		return nil
	}
	var total int64
	for _, e := range entries {
		total += e.Size
	}
	order := make([]CacheEntry, len(entries))
	copy(order, entries)
	sort.SliceStable(order, func(i, j int) bool {
		if !order[i].LastUsed.Equal(order[j].LastUsed) {
			return order[i].LastUsed.Before(order[j].LastUsed)
		}
		return order[i].Hash < order[j].Hash
	})
	var victims []string
	for _, e := range order {
		if total <= limit {
			break
		}
		if protected[e.Hash] {
			continue
		}
		victims = append(victims, e.Hash)
		total -= e.Size
	}
	return victims
}

// ScanCache lists the torrent directories directly under root with their
// sizes and last use. Symlinks are never followed or counted, and anything
// that is not a hash-named directory is ignored.
func ScanCache(root string) ([]CacheEntry, error) {
	items, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var entries []CacheEntry
	for _, it := range items {
		if !it.IsDir() || !hashDir.MatchString(it.Name()) {
			continue
		}
		dir := filepath.Join(root, it.Name())
		info, err := it.Info()
		if err != nil {
			continue
		}
		e := CacheEntry{Hash: it.Name(), LastUsed: info.ModTime()}
		_ = filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.Type().IsRegular() {
				if fi, err := d.Info(); err == nil {
					e.Size += fi.Size()
					if fi.ModTime().After(e.LastUsed) {
						e.LastUsed = fi.ModTime()
					}
				}
			}
			return nil
		})
		entries = append(entries, e)
	}
	return entries, nil
}

// RemoveEntry deletes the data of one torrent under root. It refuses names
// that are not info hashes and directories that are symlinks or resolve
// outside root, so nothing beyond Lounge's own cache can be deleted.
func RemoveEntry(root, hash string) error {
	if !hashDir.MatchString(hash) {
		return fmt.Errorf("refusing to remove %q: not an info hash", hash)
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	dir := filepath.Join(realRoot, hash)
	info, err := os.Lstat(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("refusing to remove %q: not a plain directory", dir)
	}
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return err
	}
	if filepath.Dir(resolved) != realRoot || !strings.HasPrefix(resolved, realRoot+string(filepath.Separator)) {
		return fmt.Errorf("refusing to remove %q: outside the cache", resolved)
	}
	return os.RemoveAll(resolved)
}

// touchEntry marks a torrent's data as used now.
func touchEntry(root, hash string) {
	if !hashDir.MatchString(hash) {
		return
	}
	now := time.Now()
	_ = os.Chtimes(filepath.Join(root, hash), now, now)
}
