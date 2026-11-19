package stremio

import (
	"encoding/json"
	"testing"
)

func TestNormalizeURL(t *testing.T) {
	cases := map[string]string{
		"stremio://example.com/abc/manifest.json": "https://example.com/abc/manifest.json",
		"https://example.com/abc":                 "https://example.com/abc/manifest.json",
		"https://example.com/abc/":                "https://example.com/abc/manifest.json",
		" https://example.com/manifest.json ":     "https://example.com/manifest.json",
	}
	for in, want := range cases {
		got, err := NormalizeURL(in)
		if err != nil || got != want {
			t.Errorf("NormalizeURL(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := NormalizeURL("ftp://example.com"); err == nil {
		t.Error("expected an error for ftp")
	}
}

func TestResourceURL(t *testing.T) {
	got := ResourceURL("https://a.io/x/manifest.json", "catalog", "movie", "top",
		map[string]string{"search": "the office", "skip": "100"})
	want := "https://a.io/x/catalog/movie/top/search=the%20office&skip=100.json"
	if got != want {
		t.Errorf("got %s want %s", got, want)
	}
	got = ResourceURL("https://a.io/manifest.json", "stream", "series", "tt1:1:2", nil)
	if want := "https://a.io/stream/series/tt1:1:2.json"; got != want {
		t.Errorf("got %s want %s", got, want)
	}
}

func TestManifestSupports(t *testing.T) {
	var m Manifest
	data := `{"id":"x","name":"X","types":["movie","series"],"idPrefixes":["tt"],
	  "resources":["catalog",{"name":"stream","types":["movie"],"idPrefixes":["tt","kitsu"]}],"catalogs":[]}`
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		t.Fatal(err)
	}
	checks := []struct {
		res, typ, id string
		want         bool
	}{
		{"stream", "movie", "tt123", true},
		{"stream", "series", "tt123", false},
		{"stream", "movie", "kitsu:1", true},
		{"stream", "movie", "yt:1", false},
		{"catalog", "series", "", true},
		{"meta", "movie", "tt1", false},
	}
	for _, c := range checks {
		if got := m.Supports(c.res, c.typ, c.id); got != c.want {
			t.Errorf("Supports(%s,%s,%s) = %v", c.res, c.typ, c.id, got)
		}
	}
}
