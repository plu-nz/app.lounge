package segments

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// IntroDBURL is the public IntroDB API; reads need no key.
const IntroDBURL = "https://api.introdb.app"

// IntroDB reads segments from introdb.app: GET /segments?imdb_id&season&episode.
type IntroDB struct {
	BaseURL string
	Client  *http.Client
}

// NewIntroDB returns a client with a short timeout.
func NewIntroDB() *IntroDB {
	return &IntroDB{BaseURL: IntroDBURL, Client: &http.Client{Timeout: 4 * time.Second}}
}

type introDBSpan struct {
	StartSec float64 `json:"start_sec"`
	EndSec   float64 `json:"end_sec"`
}

type introDBResponse struct {
	Intro       *introDBSpan `json:"intro"`
	Recap       *introDBSpan `json:"recap"`
	Outro       *introDBSpan `json:"outro"`
	PostCredits *introDBSpan `json:"post_credits"`
}

// Segments implements Provider. A null span means nobody timed it, so it is
// left out; an unknown episode yields no segments.
func (p *IntroDB) Segments(ctx context.Context, imdbID string, season, episode int) ([]Segment, error) {
	q := url.Values{"imdb_id": {imdbID}, "season": {fmt.Sprint(season)}, "episode": {fmt.Sprint(episode)}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.BaseURL+"/segments?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	res, err := p.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("introdb: %s", res.Status)
	}
	var body introDBResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return nil, err
	}
	var out []Segment
	add := func(kind Kind, s *introDBSpan) {
		if s != nil && s.EndSec > s.StartSec && s.StartSec >= 0 {
			out = append(out, Segment{Kind: kind, Start: s.StartSec, End: s.EndSec, Source: "introdb"})
		}
	}
	add(Recap, body.Recap)
	add(Intro, body.Intro)
	add(Credits, body.Outro)
	// Post-credits only counts when the credits themselves are not timed.
	if body.Outro == nil {
		add(Credits, body.PostCredits)
	}
	return out, nil
}

// Cached remembers successful lookups, including empty ones, so an episode is
// asked about once per run.
type Cached struct {
	Provider Provider
	mu       sync.Mutex
	cache    map[string][]Segment
}

// Segments implements Provider.
func (c *Cached) Segments(ctx context.Context, imdbID string, season, episode int) ([]Segment, error) {
	key := fmt.Sprintf("%s:%d:%d", imdbID, season, episode)
	c.mu.Lock()
	if v, ok := c.cache[key]; ok {
		c.mu.Unlock()
		return v, nil
	}
	c.mu.Unlock()
	v, err := c.Provider.Segments(ctx, imdbID, season, episode)
	if err != nil {
		// A failure may be temporary, so it is not remembered.
		return nil, err
	}
	c.mu.Lock()
	if c.cache == nil {
		c.cache = map[string][]Segment{}
	}
	c.cache[key] = v
	c.mu.Unlock()
	return v, err
}
