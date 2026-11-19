package library

import (
	"testing"

	"github.com/plu-nz/app.lounge/internal/store"
)

func TestContinueWatching(t *testing.T) {
	st, err := store.OpenDir(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	l, err := Load(st)
	if err != nil {
		t.Fatal(err)
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(l.Save(Entry{ID: "tt1", Type: "movie", VideoID: "tt1", Position: 600, Duration: 6000}))
	must(l.Save(Entry{ID: "tt2", Type: "movie", VideoID: "tt2", Position: 5900, Duration: 6000}))
	must(l.Save(Entry{ID: "tt3", Type: "series", VideoID: "tt3:1:1", Position: 2000, Duration: 2100}))
	must(l.Save(Entry{ID: "tt4", Type: "movie", VideoID: "tt4", Position: 10, Duration: 6000}))

	got := l.ContinueWatching()
	if len(got) != 2 || got[0].ID != "tt3" || got[1].ID != "tt1" {
		t.Fatalf("unexpected list: %+v", got)
	}
	if w := l.Watched([]string{"tt2", "tt3:1:1", "tt1"}); len(w) != 2 {
		t.Fatalf("watched = %v", w)
	}

	again, err := Load(st)
	must(err)
	if _, ok := again.Get("tt1"); !ok {
		t.Fatal("progress did not persist")
	}
}
