package library

import (
	"context"
	"testing"

	"github.com/plu-nz/app.lounge/internal/store"
	"github.com/plu-nz/app.lounge/internal/stremio"
)

type fakeRemote struct {
	items []stremio.LibraryItem
	put   []stremio.LibraryItem
}

func (f *fakeRemote) LibraryItems(context.Context) ([]stremio.LibraryItem, error) {
	return f.items, nil
}

func (f *fakeRemote) PutLibraryItems(_ context.Context, it []stremio.LibraryItem) error {
	f.put = append(f.put, it...)
	return nil
}

func item(id, typ, video string, offsetMs, durMs, mtime int64) stremio.LibraryItem {
	return stremio.LibraryItem{
		ID: id, Name: id, Type: typ, MTime: stremio.FormatTime(mtime),
		State: stremio.LibraryItemState{VideoID: video, TimeOffset: offsetMs, Duration: durMs},
	}
}

func newLib(t *testing.T) *Library {
	t.Helper()
	st, err := store.OpenDir(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	l, err := Load(st)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func TestSyncMerge(t *testing.T) {
	l := newLib(t)
	l.d.Entries["new-local"] = Entry{ID: "new-local", Type: "movie", Name: "NL", VideoID: "new-local", Position: 100, Duration: 5000, UpdatedAt: 5000}
	l.d.Entries["local-newer"] = Entry{ID: "local-newer", Type: "movie", VideoID: "local-newer", Position: 900, Duration: 5000, UpdatedAt: 9000}
	l.d.Entries["remote-newer"] = Entry{ID: "remote-newer", Type: "movie", VideoID: "remote-newer", Position: 10, Duration: 5000, UpdatedAt: 1000, StreamKey: "sk"}
	l.d.Entries["same"] = Entry{ID: "same", Type: "movie", VideoID: "same", Position: 50, Duration: 5000, UpdatedAt: 3000}
	l.d.Entries["gone"] = Entry{ID: "gone", Type: "movie", VideoID: "gone", Position: 50, Duration: 5000, UpdatedAt: 1}
	_ = l.Remove("gone")
	r := &fakeRemote{items: []stremio.LibraryItem{
		item("local-newer", "movie", "local-newer", 100000, 5000000, 2000),
		item("remote-newer", "movie", "remote-newer", 700000, 5000000, 8000),
		item("same", "movie", "same", 50000, 5000000, 3000),
		item("only-remote", "series", "tt9:2:3", 120000, 2400000, 4000),
		item("gone", "movie", "gone", 70000, 5000000, 0),
		{ID: "watched-movie", Type: "movie", MTime: stremio.FormatTime(100), State: stremio.LibraryItemState{VideoID: "watched-movie", TimesWatch: 1}},
		{ID: "not-in-lib", Type: "movie", Removed: true, MTime: stremio.FormatTime(100), State: stremio.LibraryItemState{TimeOffset: 99000}},
	}}
	res, err := l.Sync(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	if res.Pulled != 2 || res.Pushed != 2 {
		t.Fatalf("result = %+v", res)
	}
	if e := l.d.Entries["local-newer"]; e.Position != 900 {
		t.Fatalf("newer local progress was overwritten: %+v", e)
	}
	if e := l.d.Entries["remote-newer"]; e.Position != 700 || e.StreamKey != "sk" || e.UpdatedAt != 8000 {
		t.Fatalf("remote progress not merged: %+v", e)
	}
	if e := l.d.Entries["only-remote"]; e.Season != 2 || e.Episode != 3 || e.Position != 120 {
		t.Fatalf("remote-only entry = %+v", e)
	}
	if _, ok := l.d.Entries["gone"]; ok {
		t.Fatal("dismissed title came back")
	}
	if _, ok := l.d.Entries["not-in-lib"]; ok {
		t.Fatal("removed item became continue watching")
	}
	if len(l.Watched([]string{"watched-movie"})) != 1 {
		t.Fatal("finished remote movie not marked watched")
	}
	ids := map[string]stremio.LibraryItem{}
	for _, it := range r.put {
		ids[it.ID] = it
	}
	if _, ok := ids["new-local"]; !ok || !ids["new-local"].Temp || !ids["new-local"].Removed {
		t.Fatalf("new title not pushed as temp item: %+v", ids)
	}
	if it, ok := ids["local-newer"]; !ok || it.State.TimeOffset != 900000 || it.MTime != stremio.FormatTime(9000) {
		t.Fatalf("newer local not pushed: %+v", it)
	}
	if len(ids) != 2 {
		t.Fatalf("pushed %v", ids)
	}

	// A second sync against the updated remote pushes nothing.
	r.items = append(r.items, r.put...) // later duplicates stand in for server upserts
	r.items = dedupe(r.items)
	r.put = nil
	res, err = l.Sync(context.Background(), r)
	if err != nil || res.Pushed != 0 {
		t.Fatalf("second sync = %+v, %v", res, err)
	}
}

func TestFinishedMoviePush(t *testing.T) {
	it := toItem(Entry{ID: "m", Type: "movie", VideoID: "m", Position: 5000, Duration: 5000, UpdatedAt: 1}, stremio.LibraryItem{}, false)
	if it.State.TimeOffset != 0 || it.State.TimesWatch != 1 || it.State.Flagged != 1 {
		t.Fatalf("state = %+v", it.State)
	}
}

// dedupe keeps the last item per ID, like the server's upsert.
func dedupe(in []stremio.LibraryItem) []stremio.LibraryItem {
	last := map[string]stremio.LibraryItem{}
	for _, it := range in {
		last[it.ID] = it
	}
	var out []stremio.LibraryItem
	for _, it := range last {
		out = append(out, it)
	}
	return out
}
