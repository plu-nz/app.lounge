package torrent

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
)

func hash(c string) string { return strings.Repeat(c, 40) }

func TestPlanEvictionDropsOldestFirst(t *testing.T) {
	now := time.Now()
	entries := []CacheEntry{
		{Hash: hash("a"), Size: 40, LastUsed: now.Add(-1 * time.Hour)},
		{Hash: hash("b"), Size: 40, LastUsed: now.Add(-3 * time.Hour)},
		{Hash: hash("c"), Size: 40, LastUsed: now.Add(-2 * time.Hour)},
	}
	got := PlanEviction(entries, 90, nil)
	if want := []string{hash("b")}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	got = PlanEviction(entries, 50, nil)
	if want := []string{hash("b"), hash("c")}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestPlanEvictionSkipsProtectedAndUnlimited(t *testing.T) {
	now := time.Now()
	entries := []CacheEntry{
		{Hash: hash("a"), Size: 100, LastUsed: now.Add(-3 * time.Hour)},
		{Hash: hash("b"), Size: 100, LastUsed: now.Add(-1 * time.Hour)},
	}
	got := PlanEviction(entries, 10, map[string]bool{hash("a"): true})
	if want := []string{hash("b")}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if got := PlanEviction(entries, 0, nil); got != nil {
		t.Fatalf("unlimited planned %v", got)
	}
	if got := PlanEviction(entries, 500, nil); got != nil {
		t.Fatalf("under limit planned %v", got)
	}
}

func write(t *testing.T, path string, n int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, n), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestScanCacheSizesAndIgnoresStrangers(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, hash("a"), "sub", "f"), 30)
	write(t, filepath.Join(root, hash("a"), "g"), 12)
	write(t, filepath.Join(root, "notes"), 99)
	write(t, filepath.Join(root, "short", "f"), 99)
	entries, err := ScanCache(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Hash != hash("a") || entries[0].Size != 42 {
		t.Fatalf("entries = %+v", entries)
	}
	if got, err := ScanCache(filepath.Join(root, "missing")); err != nil || got != nil {
		t.Fatalf("missing root: %v, %v", got, err)
	}
}

func TestRemoveEntryStaysInsideRoot(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "torrents")
	outside := filepath.Join(base, "outside")
	write(t, filepath.Join(outside, "keep"), 5)
	write(t, filepath.Join(root, hash("a"), "f"), 5)
	// A symlink named like a hash that points outside the cache.
	if err := os.Symlink(outside, filepath.Join(root, hash("b"))); err != nil {
		t.Fatal(err)
	}
	// A symlink inside an entry must not be followed.
	if err := os.Symlink(outside, filepath.Join(root, hash("a"), "link")); err != nil {
		t.Fatal(err)
	}

	if err := RemoveEntry(root, hash("b")); err == nil {
		t.Fatal("removed a symlinked entry")
	}
	for _, bad := range []string{"..", "../outside", "", "a", strings.ToUpper(hash("a"))} {
		if err := RemoveEntry(root, bad); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
	if err := RemoveEntry(root, hash("a")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, hash("a"))); !os.IsNotExist(err) {
		t.Fatal("entry still there")
	}
	if _, err := os.Stat(filepath.Join(outside, "keep")); err != nil {
		t.Fatalf("outside file lost: %v", err)
	}
	if err := RemoveEntry(root, hash("c")); err != nil {
		t.Fatalf("missing entry: %v", err)
	}
}

func TestEngineEvictAndClearSkipActive(t *testing.T) {
	root := t.TempDir()
	var playing metainfo.Hash
	if err := playing.FromHexString(hash("a")); err != nil {
		t.Fatal(err)
	}
	e := &Engine{dataDir: root, active: map[metainfo.Hash]*torrent.Torrent{playing: nil}}
	old := time.Now().Add(-time.Hour)
	for _, c := range []string{"a", "b", "c"} {
		f := filepath.Join(root, hash(c), "f")
		write(t, f, 100)
		_ = os.Chtimes(f, old, old)
		_ = os.Chtimes(filepath.Dir(f), old, old)
	}
	// "a" is the oldest by name and is playing: it must survive both calls.
	e.policy = func() CachePolicy { return CachePolicy{LimitBytes: 250} }
	e.Evict()
	left, _ := ScanCache(root)
	if len(left) != 2 {
		t.Fatalf("left %+v, want the playing entry and one other", left)
	}
	freed, err := e.ClearCache()
	if err != nil || freed != 100 {
		t.Fatalf("freed %d, %v", freed, err)
	}
	if _, err := os.Stat(filepath.Join(root, hash("a"), "f")); err != nil {
		t.Fatalf("active data lost: %v", err)
	}
}
