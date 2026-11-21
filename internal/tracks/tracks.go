// Package tracks picks audio and subtitle tracks from the user's preferences
// and remembers manual choices per title.
package tracks

import (
	"sort"
	"strings"
)

// Track is one entry of mpv's track list.
type Track struct {
	ID       int    `json:"id"`
	Type     string `json:"type"` // "audio" or "sub"; other types are ignored
	Lang     string `json:"lang"`
	Title    string `json:"title"`
	Default  bool   `json:"default"`
	Forced   bool   `json:"forced"`
	External bool   `json:"external"`
}

// Subtitle modes.
const (
	// ModeOn shows subtitles in the preferred language when one exists.
	ModeOn = "on"
	// ModeOff never shows subtitles on its own.
	ModeOff = "off"
	// ModeAuto shows subtitles only when the audio is not in the preferred
	// audio language.
	ModeAuto = "auto"
)

// Prefs are the global preferences.
type Prefs struct {
	AudioLang string `json:"audioLang"`
	SubLang   string `json:"subLang"`
	SubMode   string `json:"subMode"` // ModeOn, ModeOff, ModeAuto; empty is ModeOn
}

// Choice is a manual selection. For subtitles, Off means "no subtitles".
type Choice struct {
	Lang string `json:"lang,omitempty"`
	Off  bool   `json:"off,omitempty"`
}

// Override holds the manual choices made for one title.
type Override struct {
	Audio *Choice `json:"audio,omitempty"`
	Sub   *Choice `json:"sub,omitempty"`
}

// Result says what to select. A zero ID leaves the player's own choice.
type Result struct {
	AudioID int  `json:"audioId"`
	SubID   int  `json:"subId"`
	SubOff  bool `json:"subOff"`
}

// Select picks the audio and subtitle tracks. An override, when given, wins
// over the global preferences for the kind it covers; a choice with no
// matching track falls back to the preferences.
func Select(list []Track, p Prefs, o *Override) Result {
	var res Result
	var audio, subs []Track
	for _, tr := range list {
		switch tr.Type {
		case "audio":
			audio = append(audio, tr)
		case "sub":
			subs = append(subs, tr)
		}
	}

	var chosen *Track
	if o != nil && o.Audio != nil && o.Audio.Lang != "" {
		chosen = pickAudio(audio, o.Audio.Lang)
	}
	if chosen == nil {
		chosen = pickAudio(audio, p.AudioLang)
	}
	if chosen != nil {
		res.AudioID = chosen.ID
	}

	if o != nil && o.Sub != nil {
		if o.Sub.Off {
			res.SubOff = true
			return res
		}
		if tr := pickSub(subs, o.Sub.Lang); tr != nil {
			res.SubID = tr.ID
			return res
		}
	}

	switch p.SubMode {
	case ModeOff:
		res.SubOff = true
		return res
	case ModeAuto:
		if audioMatches(audio, chosen, p.AudioLang) {
			res.SubOff = true
			return res
		}
	}
	if tr := pickSub(subs, p.SubLang); tr != nil {
		res.SubID = tr.ID
	}
	return res
}

// audioMatches reports whether playback will be in the preferred language.
// Without a preference, or without a language tag on the playing track, it
// reports false so subtitles stay available.
func audioMatches(audio []Track, chosen *Track, preferred string) bool {
	if Normalize(preferred) == "" {
		return false
	}
	if chosen != nil {
		return Same(chosen.Lang, preferred)
	}
	// No track matched the preference; the player picks its default.
	for _, tr := range audio {
		if tr.Default {
			return Same(tr.Lang, preferred)
		}
	}
	return false
}

func isCommentary(tr Track) bool {
	t := strings.ToLower(tr.Title)
	for _, w := range []string{"commentary", "comentário", "comentario", "audio description", "descritivo"} {
		if strings.Contains(t, w) {
			return true
		}
	}
	return false
}

func pickAudio(audio []Track, lang string) *Track {
	if Normalize(lang) == "" {
		return nil
	}
	rank := func(tr Track) int {
		r := 0
		if isCommentary(tr) {
			r += 2
		}
		if !tr.Default {
			r++
		}
		return r
	}
	var best *Track
	for i := range audio {
		tr := audio[i]
		if !Same(tr.Lang, lang) {
			continue
		}
		if best == nil || rank(tr) < rank(*best) || (rank(tr) == rank(*best) && tr.ID < best.ID) {
			best = &audio[i]
		}
	}
	return best
}

func isSDH(tr Track) bool {
	t := strings.ToLower(strings.TrimSpace(tr.Title))
	if t == "cc" {
		return true
	}
	for _, w := range []string{"sdh", "hearing", "caption", "[cc]", "deaf", "surdos", "legenda oculta"} {
		if strings.Contains(t, w) {
			return true
		}
	}
	return false
}

func isForced(tr Track) bool {
	t := strings.ToLower(tr.Title)
	return tr.Forced || strings.Contains(t, "forced") || strings.Contains(t, "forçada")
}

// subRank orders subtitle candidates: regular before SDH before forced, and
// embedded before addon tracks within each kind.
func subRank(tr Track) int {
	r := 0
	switch {
	case isForced(tr):
		r = 4
	case isSDH(tr):
		r = 2
	}
	if tr.External {
		r++
	}
	return r
}

func pickSub(subs []Track, lang string) *Track {
	if Normalize(lang) == "" {
		return nil
	}
	var matches []Track
	for _, tr := range subs {
		if Same(tr.Lang, lang) {
			matches = append(matches, tr)
		}
	}
	if len(matches) == 0 {
		return nil
	}
	sort.SliceStable(matches, func(i, j int) bool {
		a, b := matches[i], matches[j]
		if ra, rb := subRank(a), subRank(b); ra != rb {
			return ra < rb
		}
		if a.Default != b.Default {
			return a.Default
		}
		return a.ID < b.ID
	})
	return &matches[0]
}
