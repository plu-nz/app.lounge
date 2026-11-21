// Package segments finds the intro, recap, and credits of an episode, from
// the video's chapters or from a public segment database.
package segments

import "context"

// Kind is the type of a segment.
type Kind string

// Segment kinds.
const (
	Intro   Kind = "intro"
	Recap   Kind = "recap"
	Credits Kind = "credits"
)

// Segment is a span of playback, in seconds, that can be skipped.
type Segment struct {
	Kind  Kind    `json:"kind"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	// Source is "chapters" or the name of the provider.
	Source string `json:"source"`
}

// Provider looks up the segments of a series episode.
type Provider interface {
	Segments(ctx context.Context, imdbID string, season, episode int) ([]Segment, error)
}

// Merge combines segment lists in priority order: a kind found by an earlier
// list is not taken from a later one.
func Merge(lists ...[]Segment) []Segment {
	seen := map[Kind]bool{}
	var out []Segment
	for _, list := range lists {
		found := map[Kind]bool{}
		for _, s := range list {
			if seen[s.Kind] || s.End <= s.Start {
				continue
			}
			out = append(out, s)
			found[s.Kind] = true
		}
		for k := range found {
			seen[k] = true
		}
	}
	return out
}
