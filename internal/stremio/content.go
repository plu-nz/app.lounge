package stremio

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
)

// Row identifies one catalog shown as a row.
type Row struct {
	AddonID   string   `json:"addonId"`
	AddonName string   `json:"addonName"`
	Type      string   `json:"type"`
	CatalogID string   `json:"catalogId"`
	Name      string   `json:"name"`
	Genres    []string `json:"genres,omitempty"`
}

// SearchRow is a row of search results.
type SearchRow struct {
	Row
	Items []MetaPreview `json:"items"`
	Error string        `json:"error,omitempty"`
}

// StreamGroup is the streams one addon returned.
type StreamGroup struct {
	AddonID   string   `json:"addonId"`
	AddonName string   `json:"addonName"`
	Streams   []Stream `json:"streams"`
	Error     string   `json:"error,omitempty"`
}

// Content answers catalog, meta, stream, and subtitle queries across the
// enabled addons.
type Content struct {
	Client *Client
	Addons *Addons
}

// Rows lists the catalogs that load without required filters, in addon
// order. typ limits the list to one type when it is not empty.
func (c *Content) Rows(typ string) []Row {
	var rows []Row
	for _, ad := range c.Addons.Enabled() {
		for _, cat := range ad.Manifest.Catalogs {
			if typ != "" && cat.Type != typ {
				continue
			}
			if len(cat.RequiredExtras()) > 0 {
				continue
			}
			rows = append(rows, Row{
				AddonID: ad.Manifest.ID, AddonName: ad.Manifest.Name,
				Type: cat.Type, CatalogID: cat.ID, Name: cat.Name, Genres: cat.Genres(),
			})
		}
	}
	return rows
}

// Catalog loads one page of a catalog. extra holds filters such as genre,
// search, or skip.
func (c *Content) Catalog(ctx context.Context, addonID, typ, catalogID string, extra map[string]string) ([]MetaPreview, error) {
	ad, ok := c.Addons.Get(addonID)
	if !ok {
		return nil, fmt.Errorf("addon %q is not installed", addonID)
	}
	items, err := c.Client.Catalog(ctx, ad.TransportURL, typ, catalogID, extra)
	for i := range items {
		if items[i].Type == "" {
			items[i].Type = typ
		}
	}
	return items, err
}

// Search queries every catalog that supports search, in parallel.
func (c *Content) Search(ctx context.Context, query string) []SearchRow {
	var rows []SearchRow
	for _, ad := range c.Addons.Enabled() {
		for _, cat := range ad.Manifest.Catalogs {
			if !cat.Supports("search") {
				continue
			}
			// Catalogs that need another filter besides search cannot run.
			if slices.ContainsFunc(cat.RequiredExtras(), func(e string) bool { return e != "search" }) {
				continue
			}
			rows = append(rows, SearchRow{Row: Row{
				AddonID: ad.Manifest.ID, AddonName: ad.Manifest.Name,
				Type: cat.Type, CatalogID: cat.ID, Name: cat.Name,
			}})
		}
	}
	var wg sync.WaitGroup
	for i := range rows {
		wg.Go(func() {
			r := &rows[i]
			items, err := c.Catalog(ctx, r.AddonID, r.Type, r.CatalogID, map[string]string{"search": query})
			r.Items = items
			if err != nil {
				r.Error = err.Error()
			}
		})
	}
	wg.Wait()
	return slices.DeleteFunc(rows, func(r SearchRow) bool { return len(r.Items) == 0 })
}

// Meta returns the first full description an addon offers for an item.
func (c *Content) Meta(ctx context.Context, typ, id string) (*Meta, error) {
	var errs []error
	for _, ad := range c.Addons.Enabled() {
		if !ad.Manifest.Supports("meta", typ, id) {
			continue
		}
		m, err := c.Client.Meta(ctx, ad.TransportURL, typ, id)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", ad.Manifest.Name, err))
			continue
		}
		if m != nil {
			if m.Type == "" {
				m.Type = typ
			}
			return m, nil
		}
	}
	if len(errs) == 0 {
		return nil, fmt.Errorf("no addon describes %s %s", typ, id)
	}
	return nil, errors.Join(errs...)
}

// Streams asks every stream addon for a video ID, in parallel.
func (c *Content) Streams(ctx context.Context, typ, id string) []StreamGroup {
	var groups []StreamGroup
	var urls []string
	for _, ad := range c.Addons.Enabled() {
		if ad.Manifest.Supports("stream", typ, id) {
			groups = append(groups, StreamGroup{AddonID: ad.Manifest.ID, AddonName: ad.Manifest.Name})
			urls = append(urls, ad.TransportURL)
		}
	}
	var wg sync.WaitGroup
	for i := range groups {
		wg.Go(func() {
			streams, err := c.Client.Streams(ctx, urls[i], typ, id)
			groups[i].Streams = streams
			if err != nil {
				groups[i].Error = err.Error()
			}
		})
	}
	wg.Wait()
	return groups
}

// Subtitles collects external subtitles from every subtitle addon.
func (c *Content) Subtitles(ctx context.Context, typ, id string, extra map[string]string) []Subtitle {
	var urls []string
	for _, ad := range c.Addons.Enabled() {
		if ad.Manifest.Supports("subtitles", typ, id) {
			urls = append(urls, ad.TransportURL)
		}
	}
	results := make([][]Subtitle, len(urls))
	var wg sync.WaitGroup
	for i, u := range urls {
		wg.Go(func() {
			results[i], _ = c.Client.Subtitles(ctx, u, typ, id, extra)
		})
	}
	wg.Wait()
	var out []Subtitle
	for _, r := range results {
		out = append(out, r...)
	}
	return out
}
