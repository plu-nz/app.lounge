// Package stremio implements a client for the Stremio addon protocol.
//
// Protocol reference: https://github.com/Stremio/stremio-addon-sdk/tree/master/docs
package stremio

import (
	"encoding/json"
	"slices"
	"strings"
)

// Manifest describes an addon and the resources it serves.
type Manifest struct {
	ID            string         `json:"id"`
	Version       string         `json:"version"`
	Name          string         `json:"name"`
	Description   string         `json:"description,omitempty"`
	Logo          string         `json:"logo,omitempty"`
	Background    string         `json:"background,omitempty"`
	Types         []string       `json:"types"`
	Catalogs      []CatalogDef   `json:"catalogs"`
	Resources     []Resource     `json:"resources"`
	IDPrefixes    []string       `json:"idPrefixes,omitempty"`
	BehaviorHints map[string]any `json:"behaviorHints,omitempty"`
}

// Resource is a resource name with optional type and ID prefix limits. The
// manifest may list it as a bare string or as an object.
type Resource struct {
	Name       string   `json:"name"`
	Types      []string `json:"types,omitempty"`
	IDPrefixes []string `json:"idPrefixes,omitempty"`
}

// UnmarshalJSON accepts both the string and the object form.
func (r *Resource) UnmarshalJSON(data []byte) error {
	var name string
	if err := json.Unmarshal(data, &name); err == nil {
		*r = Resource{Name: name}
		return nil
	}
	type plain Resource
	return json.Unmarshal(data, (*plain)(r))
}

// CatalogDef is one catalog an addon offers.
type CatalogDef struct {
	Type           string     `json:"type"`
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	Extra          []ExtraDef `json:"extra,omitempty"`
	ExtraSupported []string   `json:"extraSupported,omitempty"`
	ExtraRequired  []string   `json:"extraRequired,omitempty"`
}

// ExtraDef is a catalog filter such as search, genre, or skip.
type ExtraDef struct {
	Name         string   `json:"name"`
	IsRequired   bool     `json:"isRequired,omitempty"`
	Options      []string `json:"options,omitempty"`
	OptionsLimit int      `json:"optionsLimit,omitempty"`
}

// Supports reports whether the catalog accepts an extra property.
func (c CatalogDef) Supports(extra string) bool {
	for _, e := range c.Extra {
		if e.Name == extra {
			return true
		}
	}
	return slices.Contains(c.ExtraSupported, extra)
}

// Requires reports whether the catalog needs an extra property.
func (c CatalogDef) Requires(extra string) bool {
	for _, e := range c.Extra {
		if e.Name == extra && e.IsRequired {
			return true
		}
	}
	return slices.Contains(c.ExtraRequired, extra)
}

// RequiredExtras lists the extras that must be set to load the catalog.
func (c CatalogDef) RequiredExtras() []string {
	var out []string
	for _, e := range c.Extra {
		if e.IsRequired {
			out = append(out, e.Name)
		}
	}
	for _, e := range c.ExtraRequired {
		if !slices.Contains(out, e) {
			out = append(out, e)
		}
	}
	return out
}

// Genres returns the genre options of the catalog, if any.
func (c CatalogDef) Genres() []string {
	for _, e := range c.Extra {
		if e.Name == "genre" {
			return e.Options
		}
	}
	return nil
}

// Supports reports whether the manifest serves a resource for a type and ID.
func (m Manifest) Supports(resource, typ, id string) bool {
	for _, r := range m.Resources {
		if r.Name != resource {
			continue
		}
		types, prefixes := r.Types, r.IDPrefixes
		if len(types) == 0 {
			types = m.Types
		}
		if r.Types == nil && r.IDPrefixes == nil {
			prefixes = m.IDPrefixes
		}
		if typ != "" && len(types) > 0 && !slices.Contains(types, typ) {
			return false
		}
		if id != "" && len(prefixes) > 0 {
			return slices.ContainsFunc(prefixes, func(p string) bool { return strings.HasPrefix(id, p) })
		}
		return true
	}
	return false
}

// MetaPreview is a catalog item.
type MetaPreview struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"`
	Name        string   `json:"name"`
	Poster      string   `json:"poster,omitempty"`
	PosterShape string   `json:"posterShape,omitempty"`
	Background  string   `json:"background,omitempty"`
	Logo        string   `json:"logo,omitempty"`
	Description string   `json:"description,omitempty"`
	ReleaseInfo string   `json:"releaseInfo,omitempty"`
	ImdbRating  string   `json:"imdbRating,omitempty"`
	Genres      []string `json:"genres,omitempty"`
	Runtime     string   `json:"runtime,omitempty"`
}

// Meta is the full description of an item.
type Meta struct {
	MetaPreview
	Videos    []Video  `json:"videos,omitempty"`
	Cast      []string `json:"cast,omitempty"`
	Director  []string `json:"director,omitempty"`
	Writer    []string `json:"writer,omitempty"`
	Country   string   `json:"country,omitempty"`
	Released  string   `json:"released,omitempty"`
	Language  string   `json:"language,omitempty"`
	Trailers  []any    `json:"trailers,omitempty"`
	Links     []Link   `json:"links,omitempty"`
	AwardInfo string   `json:"awards,omitempty"`
}

// Link is a meta link such as a genre or cast member.
type Link struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	URL      string `json:"url"`
}

// Video is an episode or part of a meta item.
type Video struct {
	ID        string `json:"id"`
	Title     string `json:"title,omitempty"`
	Name      string `json:"name,omitempty"`
	Season    int    `json:"season,omitempty"`
	Episode   int    `json:"episode,omitempty"`
	Number    int    `json:"number,omitempty"`
	Released  string `json:"released,omitempty"`
	Thumbnail string `json:"thumbnail,omitempty"`
	Overview  string `json:"overview,omitempty"`
}

// Stream is one playable source returned by an addon.
type Stream struct {
	URL           string      `json:"url,omitempty"`
	YtID          string      `json:"ytId,omitempty"`
	InfoHash      string      `json:"infoHash,omitempty"`
	FileIdx       *int        `json:"fileIdx,omitempty"`
	ExternalURL   string      `json:"externalUrl,omitempty"`
	Name          string      `json:"name,omitempty"`
	Title         string      `json:"title,omitempty"`
	Description   string      `json:"description,omitempty"`
	Sources       []string    `json:"sources,omitempty"`
	Subtitles     []Subtitle  `json:"subtitles,omitempty"`
	BehaviorHints StreamHints `json:"behaviorHints,omitempty"`
}

// StreamHints carries playback hints for a stream.
type StreamHints struct {
	NotWebReady  bool              `json:"notWebReady,omitempty"`
	BingeGroup   string            `json:"bingeGroup,omitempty"`
	Filename     string            `json:"filename,omitempty"`
	VideoSize    int64             `json:"videoSize,omitempty"`
	ProxyHeaders *ProxyHeaders     `json:"proxyHeaders,omitempty"`
	CountryWL    []string          `json:"countryWhitelist,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
}

// ProxyHeaders are HTTP headers a stream needs.
type ProxyHeaders struct {
	Request  map[string]string `json:"request,omitempty"`
	Response map[string]string `json:"response,omitempty"`
}

// Subtitle is an external subtitle track.
type Subtitle struct {
	ID   string `json:"id"`
	URL  string `json:"url"`
	Lang string `json:"lang"`
}
