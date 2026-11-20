package stremio

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/plu-nz/app.lounge/internal/store"
)

func TestAccountAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if r.Method == http.MethodPost {
			data, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(data, &body)
		}
		switch r.URL.Path {
		case "/api/login":
			if body["type"] != "Login" || body["password"] != "pw" {
				w.Write([]byte(`{"error":{"code":2,"message":"Wrong password"}}`))
				return
			}
			w.Write([]byte(`{"result":{"authKey":"k1","user":{"email":"a@b.c"}}}`))
		case "/api/addonCollectionGet":
			if body["authKey"] != "k1" {
				w.Write([]byte(`{"error":{"code":1,"message":"Session does not exist"}}`))
				return
			}
			w.Write([]byte(`{"result":{"addons":[
			  {"transportUrl":"https://a.io/manifest.json","manifest":{"id":"a","name":"A","version":"1","types":["movie"],"resources":["stream"],"catalogs":[]},"flags":{"official":true}},
			  {"transportUrl":"http://127.0.0.1:11470/local-addon/manifest.json","manifest":{"id":"org.stremio.local","name":"Local","version":"1","types":[],"resources":[],"catalogs":[]}},
			  {"transportUrl":"https://b.io/manifest.json","manifest":{"id":"b","name":"B","version":"1","types":["series"],"resources":["catalog"],"catalogs":[]}}
			],"lastModified":"2026-11-19T00:00:00Z"}}`))
		case "/link/read":
			if r.URL.Query().Get("code") == "OK" {
				w.Write([]byte(`{"result":{"authKey":"k2"}}`))
				return
			}
			w.Write([]byte(`{"error":{"code":101,"message":"Invalid or expired token"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	apiURL, linkURL = srv.URL+"/api/", srv.URL+"/link/"

	c := NewClient()
	ctx := context.Background()
	if _, err := c.Login(ctx, "a@b.c", "bad"); err == nil || err.Error() != "Wrong password" {
		t.Fatalf("bad password: %v", err)
	}
	acc, err := c.Login(ctx, "a@b.c", "pw")
	if err != nil || acc.AuthKey != "k1" || acc.Email != "a@b.c" {
		t.Fatalf("login = %+v, %v", acc, err)
	}
	if _, err := c.ReadLink(ctx, "WAIT"); !errors.Is(err, ErrLinkPending) {
		t.Fatalf("pending read: %v", err)
	}
	if key, err := c.ReadLink(ctx, "OK"); err != nil || key != "k2" {
		t.Fatalf("confirmed read = %q, %v", key, err)
	}

	descs, err := c.AddonCollection(ctx, "k1")
	if err != nil || len(descs) != 3 {
		t.Fatalf("collection = %d, %v", len(descs), err)
	}
	st, _ := store.OpenDir(t.TempDir())
	a := &Addons{client: c, store: st, list: []Addon{{Manifest: Manifest{ID: "b"}, Enabled: false}}}
	if err := a.ReplaceFromAccount(descs); err != nil {
		t.Fatal(err)
	}
	got := a.List()
	if len(got) != 2 || got[0].Manifest.ID != "a" || !got[0].Enabled || got[1].Manifest.ID != "b" || got[1].Enabled {
		t.Fatalf("mirrored list = %+v", got)
	}
	a.SetManaged(true)
	if _, err := a.Install(ctx, "https://c.io/manifest.json"); !errors.Is(err, ErrManaged) {
		t.Fatalf("install while managed: %v", err)
	}
}
