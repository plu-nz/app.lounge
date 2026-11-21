// Package sources ranks the streams addons offer for a video, so Lounge can
// pick the best one without asking.
package sources

import (
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/plu-nz/app.lounge/internal/stremio"
)

// Transport is how a stream is delivered.
type Transport int

const (
	Direct Transport = iota
	Torrent
)

// Rules are the user's source selection preferences. The zero value accepts
// every playable source and ranks by quality alone.
type Rules struct {
	// MaxHeight caps the resolution (2160, 1080, 720); 0 means any.
	// Sources of a known higher resolution are rejected.
	MaxHeight int
	// MaxSizeGB rejects sources of a known larger size; 0 means any.
	MaxSizeGB float64
	// Prefer ranks one transport above the other regardless of quality:
	// "direct", "torrent", or empty for no preference.
	Prefer string
	// MinSeeders rejects torrents that report fewer seeders. Torrents that
	// report none are kept, since not every addon says.
	MinSeeders int
}

// Info is what could be read from a stream's text and hints. Zero means the
// field is unknown.
type Info struct {
	Height    int
	SizeBytes int64
	Seeders   int
	HasSeeds  bool
	HDR       bool
	Transport Transport
}

var (
	resRe     = regexp.MustCompile(`(?i)\b(2160p|1440p|1080p|720p|576p|480p)\b`)
	dimRe     = regexp.MustCompile(`(?i)\b\d{3,4}\s?x\s?(2160|1440|1080|720|480)\b`)
	fourKRe   = regexp.MustCompile(`(?i)\b(4k|uhd)\b`)
	sdRe      = regexp.MustCompile(`(?i)\bsd\b`)
	sizeRe    = regexp.MustCompile(`(?i)(\d+(?:[.,]\d+)?)\s*([KMGT])i?B\b`)
	seedersRe = regexp.MustCompile(`(?i)(?:👤|\bseeders?\b|\bseeds?\b)\s*[:=]?\s*(\d+)`)
	hdrRe     = regexp.MustCompile(`(?i)\bHDR(10\+?)?\b|\bDV\b|dolby.?vision`)
)

func text(s stremio.Stream) string {
	return s.Name + "\n" + s.Title + "\n" + s.Description + "\n" + s.BehaviorHints.Filename
}

// Parse reads quality, size, seeders, and transport from a stream.
func Parse(s stremio.Stream) Info {
	t := text(s)
	in := Info{HDR: hdrRe.MatchString(t)}
	if s.InfoHash != "" || strings.HasPrefix(s.URL, "magnet:") {
		in.Transport = Torrent
	}
	switch {
	case resRe.MatchString(t):
		in.Height, _ = strconv.Atoi(strings.TrimSuffix(strings.ToLower(resRe.FindString(t)), "p"))
	case dimRe.MatchString(t):
		in.Height, _ = strconv.Atoi(dimRe.FindStringSubmatch(t)[1])
	case fourKRe.MatchString(t):
		in.Height = 2160
	case sdRe.MatchString(t):
		in.Height = 480
	}
	if in.Height == 576 {
		in.Height = 480
	}
	if n := s.BehaviorHints.VideoSize; n > 0 {
		in.SizeBytes = n
	} else if m := sizeRe.FindStringSubmatch(t); m != nil {
		v, _ := strconv.ParseFloat(strings.Replace(m[1], ",", ".", 1), 64)
		mult := map[string]float64{"K": 1 << 10, "M": 1 << 20, "G": 1 << 30, "T": 1 << 40}[strings.ToUpper(m[2])]
		in.SizeBytes = int64(v * mult)
	}
	if in.Transport == Torrent {
		if m := seedersRe.FindStringSubmatch(t); m != nil {
			in.Seeders, _ = strconv.Atoi(m[1])
			in.HasSeeds = true
		}
	}
	return in
}

// Playable reports whether Lounge can play a stream.
func Playable(s stremio.Stream) bool { return s.URL != "" || s.InfoHash != "" }

// Accept reports whether a stream passes the rules.
func (r Rules) Accept(s stremio.Stream) bool {
	if !Playable(s) {
		return false
	}
	in := Parse(s)
	if r.MaxHeight > 0 && in.Height > r.MaxHeight {
		return false
	}
	if r.MaxSizeGB > 0 && in.SizeBytes > int64(r.MaxSizeGB*(1<<30)) {
		return false
	}
	if r.MinSeeders > 0 && in.Transport == Torrent && in.HasSeeds && in.Seeders < r.MinSeeders {
		return false
	}
	return true
}

// Score ranks a stream; higher is better. Order of importance: the preferred
// transport, then resolution, then HDR, then seeders, then size (a bigger
// file at the same resolution has the higher bitrate).
func (r Rules) Score(s stremio.Stream) float64 {
	in := Parse(s)
	var score float64
	if (r.Prefer == "direct" && in.Transport == Direct) || (r.Prefer == "torrent" && in.Transport == Torrent) {
		score += 1e6
	}
	// Unknown resolution ranks between 480p and 720p.
	h := in.Height
	if h == 0 {
		h = 576
	}
	score += float64(h) * 100
	if in.HDR {
		score += 50
	}
	if in.HasSeeds {
		score += math.Min(math.Log10(float64(in.Seeders)+1)*10, 40)
	}
	score += math.Min(float64(in.SizeBytes)/float64(1<<30), 40) / 2
	return score
}

// Ref locates a stream inside a list of addon groups.
type Ref struct {
	Group  int `json:"group"`
	Stream int `json:"stream"`
}

// Rank returns every accepted stream, best first. Equal scores keep the
// order the addons gave, which follows the user's addon order.
func Rank(groups []stremio.StreamGroup, r Rules) []Ref {
	type scored struct {
		ref   Ref
		score float64
	}
	var all []scored
	for gi, g := range groups {
		for si, s := range g.Streams {
			if r.Accept(s) {
				all = append(all, scored{Ref{gi, si}, r.Score(s)})
			}
		}
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].score > all[j].score })
	out := make([]Ref, len(all))
	for i, s := range all {
		out[i] = s.ref
	}
	return out
}

// Best returns the top-ranked stream, if any passes the rules.
func Best(groups []stremio.StreamGroup, r Rules) (Ref, bool) {
	ranked := Rank(groups, r)
	if len(ranked) == 0 {
		return Ref{}, false
	}
	return ranked[0], true
}
