package segments

import (
	"regexp"
	"strings"
)

// Chapter is an entry of mpv's chapter-list.
type Chapter struct {
	Title string  `json:"title"`
	Time  float64 `json:"time"`
}

// Titles are matched after lowercasing and removing accents. Recap comes
// first so "previously on" is never read as an intro.
var chapterPatterns = []struct {
	kind Kind
	re   *regexp.Regexp
}{
	{Recap, regexp.MustCompile(`\b(recap|previously|previamente|anteriormente|resumo|resumen)\b`)},
	{Credits, regexp.MustCompile(`\b(credits?|creditos|ending|encerramento|outro|cierre|epilogo)\b|^ed$`)},
	{Intro, regexp.MustCompile(`\b(intro|introduction|introducao|introduccion|opening|abertura|apertura|title sequence|theme song)\b|^op$`)},
}

var accentless = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i",
	"ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u",
	"ç", "c", "ñ", "n",
)

// Match returns the segment kind a chapter title names, or "" for none.
func Match(title string) Kind {
	t := accentless.Replace(strings.ToLower(strings.TrimSpace(title)))
	for _, p := range chapterPatterns {
		if p.re.MatchString(t) {
			return p.kind
		}
	}
	return ""
}

// FromChapters turns matching chapters into segments. A segment runs from its
// chapter to the next one, or to the end of the file for the last chapter.
func FromChapters(chapters []Chapter, duration float64) []Segment {
	var out []Segment
	for i, c := range chapters {
		kind := Match(c.Title)
		if kind == "" {
			continue
		}
		end := duration
		if i+1 < len(chapters) {
			end = chapters[i+1].Time
		}
		if end <= c.Time {
			continue
		}
		out = append(out, Segment{Kind: kind, Start: c.Time, End: end, Source: "chapters"})
	}
	return out
}
