package stremio

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLibraryDatastore(t *testing.T) {
	var put map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(data, &body)
		if body["authKey"] != "k1" || body["collection"] != "libraryItem" {
			w.Write([]byte(`{"error":{"code":1,"message":"Session does not exist"}}`))
			return
		}
		switch r.URL.Path {
		case "/api/datastoreGet":
			if body["all"] != true {
				t.Errorf("get body = %v", body)
			}
			w.Write([]byte(`{"result":[{"_id":"tt1","name":"One","type":"movie","poster":"","removed":false,"temp":false,
			 "_ctime":"2026-01-01T00:00:00.000Z","_mtime":"2026-02-01T10:00:00.500Z","posterShape":"poster","extra":{"a":1},
			 "state":{"lastWatched":null,"timeWatched":5,"timeOffset":60000,"overallTimeWatched":9,"timesWatched":0,"flaggedWatched":0,"duration":6000000,"video_id":"tt1","watched":"x","noNotif":false}}]}`))
		case "/api/datastorePut":
			put = body
			w.Write([]byte(`{"result":{"success":true}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	apiURL = srv.URL + "/api/"

	c := NewClient()
	ctx := context.Background()
	if _, err := c.LibraryItems(ctx, "bad"); err == nil {
		t.Fatal("bad key should fail")
	}
	items, err := c.LibraryItems(ctx, "k1")
	if err != nil || len(items) != 1 {
		t.Fatalf("items = %v, %v", items, err)
	}
	it := items[0]
	if it.State.TimeOffset != 60000 || it.MTimeMillis() != 1769940000500 {
		t.Fatalf("item = %+v mtime %d", it, it.MTimeMillis())
	}
	it.State.TimeOffset = 90000
	if err := c.PutLibraryItems(ctx, "k1", []LibraryItem{it}); err != nil {
		t.Fatal(err)
	}
	changes := put["changes"].([]any)
	sent := changes[0].(map[string]any)
	state := sent["state"].(map[string]any)
	if sent["extra"] == nil || state["watched"] != "x" || state["overallTimeWatched"] != float64(9) || state["timeOffset"] != float64(90000) {
		t.Fatalf("server fields not preserved: %v", sent)
	}
	fresh, _ := json.Marshal(LibraryItem{ID: "n"})
	if !strings.Contains(string(fresh), `"posterShape"`) || !strings.Contains(string(fresh), `"noNotif"`) {
		t.Fatalf("new item lacks required fields: %s", fresh)
	}
}
