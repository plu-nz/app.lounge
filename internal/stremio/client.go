package stremio

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

// Client fetches addon resources with a small in-memory cache.
type Client struct {
	HTTP *http.Client

	mu    sync.Mutex
	cache map[string]cached
}

type cached struct {
	body    []byte
	expires time.Time
}

// NewClient returns a client with sensible timeouts.
func NewClient() *Client {
	return &Client{
		HTTP:  &http.Client{Timeout: 20 * time.Second},
		cache: map[string]cached{},
	}
}

// NormalizeURL turns an install link into a manifest URL.
func NormalizeURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if rest, ok := strings.CutPrefix(raw, "stremio://"); ok {
		raw = "https://" + rest
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", fmt.Errorf("not an addon URL: %q", raw)
	}
	if !strings.HasSuffix(u.Path, "/manifest.json") {
		u.Path = strings.TrimSuffix(u.Path, "/") + "/manifest.json"
	}
	return u.String(), nil
}

// BaseURL strips manifest.json from a manifest URL.
func BaseURL(manifestURL string) string {
	base, _, _ := strings.Cut(manifestURL, "?")
	return strings.TrimSuffix(base, "/manifest.json")
}

// Manifest fetches and decodes a manifest.
func (c *Client) Manifest(ctx context.Context, manifestURL string) (Manifest, error) {
	var m Manifest
	if err := c.getJSON(ctx, manifestURL, 0, &m); err != nil {
		return m, err
	}
	if m.ID == "" || m.Name == "" {
		return m, fmt.Errorf("invalid manifest at %s", manifestURL)
	}
	return m, nil
}

// ResourceURL builds the URL of a resource request.
func ResourceURL(manifestURL, resource, typ, id string, extra map[string]string) string {
	var b strings.Builder
	b.WriteString(BaseURL(manifestURL))
	for _, part := range []string{resource, typ, id} {
		b.WriteByte('/')
		b.WriteString(url.PathEscape(part))
	}
	if len(extra) > 0 {
		keys := make([]string, 0, len(extra))
		for k := range extra {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		pairs := make([]string, 0, len(keys))
		for _, k := range keys {
			pairs = append(pairs, url.PathEscape(k)+"="+url.PathEscape(extra[k]))
		}
		b.WriteByte('/')
		b.WriteString(strings.Join(pairs, "&"))
	}
	b.WriteString(".json")
	return b.String()
}

// Catalog fetches one catalog page.
func (c *Client) Catalog(ctx context.Context, manifestURL, typ, id string, extra map[string]string) ([]MetaPreview, error) {
	var res struct {
		Metas []MetaPreview `json:"metas"`
	}
	err := c.getJSON(ctx, ResourceURL(manifestURL, "catalog", typ, id, extra), 30*time.Minute, &res)
	return res.Metas, err
}

// Meta fetches the full description of an item.
func (c *Client) Meta(ctx context.Context, manifestURL, typ, id string) (*Meta, error) {
	var res struct {
		Meta *Meta `json:"meta"`
	}
	if err := c.getJSON(ctx, ResourceURL(manifestURL, "meta", typ, id, nil), 6*time.Hour, &res); err != nil {
		return nil, err
	}
	return res.Meta, nil
}

// Streams fetches the streams an addon offers for a video ID.
func (c *Client) Streams(ctx context.Context, manifestURL, typ, id string) ([]Stream, error) {
	var res struct {
		Streams []Stream `json:"streams"`
	}
	err := c.getJSON(ctx, ResourceURL(manifestURL, "stream", typ, id, nil), 5*time.Minute, &res)
	return res.Streams, err
}

// Subtitles fetches external subtitles for a video ID.
func (c *Client) Subtitles(ctx context.Context, manifestURL, typ, id string, extra map[string]string) ([]Subtitle, error) {
	var res struct {
		Subtitles []Subtitle `json:"subtitles"`
	}
	err := c.getJSON(ctx, ResourceURL(manifestURL, "subtitles", typ, id, extra), 30*time.Minute, &res)
	return res.Subtitles, err
}

func (c *Client) getJSON(ctx context.Context, u string, ttl time.Duration, v any) error {
	if ttl > 0 {
		c.mu.Lock()
		hit, ok := c.cache[u]
		c.mu.Unlock()
		if ok && time.Now().Before(hit.expires) {
			return json.Unmarshal(hit.body, v)
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", u, resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("decode %s: %w", u, err)
	}
	if ttl > 0 {
		c.mu.Lock()
		c.cache[u] = cached{body: body, expires: time.Now().Add(ttl)}
		c.mu.Unlock()
	}
	return nil
}
