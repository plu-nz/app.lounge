package segments

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestMatch(t *testing.T) {
	cases := map[string]Kind{
		"Intro":              Intro,
		"Opening":            Intro,
		"OP":                 Intro,
		"Abertura":           Intro,
		"Introdução":         Intro,
		"Introducción":       Intro,
		"Previously on Lost": Recap,
		"Recap":              Recap,
		"Anteriormente":      Recap,
		"Resumen":            Recap,
		"Credits":            Credits,
		"End Credits":        Credits,
		"Ending":             Credits,
		"Créditos":           Credits,
		"Encerramento":       Credits,
		"Episode":            "",
		"Chapter 3":          "",
		"Opposite day":       "",
		"Pre-Credits Scene":  Credits,
		"Cold open":          "",
	}
	for title, want := range cases {
		if got := Match(title); got != want {
			t.Errorf("Match(%q) = %q, want %q", title, got, want)
		}
	}
}

func TestFromChapters(t *testing.T) {
	chapters := []Chapter{
		{"Previously on", 0},
		{"Opening", 60},
		{"Episode", 150},
		{"Credits", 2700},
	}
	got := FromChapters(chapters, 2800)
	want := []Segment{
		{Recap, 0, 60, "chapters"},
		{Intro, 60, 150, "chapters"},
		{Credits, 2700, 2800, "chapters"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("segment %d = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestMergePrefersEarlierLists(t *testing.T) {
	a := []Segment{{Intro, 1, 2, "chapters"}}
	b := []Segment{{Intro, 5, 9, "introdb"}, {Credits, 100, 120, "introdb"}, {Recap, 3, 3, "introdb"}}
	got := Merge(a, b)
	if len(got) != 2 || got[0].Source != "chapters" || got[1].Kind != Credits {
		t.Fatalf("got %v", got)
	}
}

const sample = `{"imdb_id":"tt1","media_type":"tv","is_movie":false,"season":1,"episode":2,
"intro":{"start_sec":2.5,"end_sec":58,"start_ms":2500,"end_ms":58000,"confidence":0.9,"submission_count":3},
"recap":null,"outro":{"start_sec":3631.5,"end_sec":3699.5},"post_credits":null,"no_recap":true}`

func TestIntroDBParsing(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		q := r.URL.Query()
		if r.URL.Path != "/segments" || q.Get("imdb_id") != "tt1" || q.Get("season") != "1" || q.Get("episode") != "2" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(sample))
	}))
	defer srv.Close()

	p := &Cached{Provider: &IntroDB{BaseURL: srv.URL, Client: srv.Client()}}
	for i := 0; i < 2; i++ {
		got, err := p.Segments(context.Background(), "tt1", 1, 2)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 || got[0] != (Segment{Intro, 2.5, 58, "introdb"}) || got[1] != (Segment{Credits, 3631.5, 3699.5, "introdb"}) {
			t.Fatalf("got %v", got)
		}
	}
	if hits.Load() != 1 {
		t.Errorf("provider asked %d times, want 1", hits.Load())
	}
	got, err := p.Segments(context.Background(), "tt9", 1, 1)
	if err != nil || len(got) != 0 {
		t.Errorf("unknown episode: %v, %v", got, err)
	}
}

func TestIntroDBFailures(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	p := &IntroDB{BaseURL: srv.URL, Client: srv.Client()}
	if _, err := p.Segments(context.Background(), "tt1", 1, 1); err == nil {
		t.Error("expected an error for a 500")
	}

	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
	}))
	defer slow.Close()
	p = &IntroDB{BaseURL: slow.URL, Client: &http.Client{Timeout: 50 * time.Millisecond}}
	if _, err := p.Segments(context.Background(), "tt1", 1, 1); err == nil {
		t.Error("expected a timeout")
	}
}
