// Package subtitle maps the subtitle appearance preferences onto mpv
// properties.
package subtitle

import (
	"strconv"
	"strings"
)

// Style is the subtitle appearance the user picked.
type Style struct {
	// Scale multiplies the subtitle size (mpv sub-scale).
	Scale float64 `json:"scale"`
	// Color is a preset name: white, yellow, green, cyan.
	Color string `json:"color"`
	// Background is "none", "outline" or "box".
	Background string `json:"background"`
	// Position is a preset name: bottom, raised or high.
	Position string `json:"position"`
}

// Colors maps each color preset to an mpv color.
var Colors = map[string]string{
	"white":  "#FFFFFF",
	"yellow": "#FFE94A",
	"green":  "#7CFF8A",
	"cyan":   "#6FE7FF",
}

// Positions maps each position preset to mpv's sub-pos, where 100 is the
// bottom edge of the screen.
var Positions = map[string]int{
	"bottom": 100,
	"raised": 94,
	"high":   86,
}

const (
	minScale = 0.3
	maxScale = 3
)

// Properties returns the mpv properties for a style, in the order to set
// them. Unknown or empty values fall back to the defaults (white text,
// outline, bottom, normal size), so settings saved before these options
// existed keep working.
func Properties(s Style) [][2]string {
	scale := s.Scale
	if scale <= 0 {
		scale = 1
	}
	scale = min(max(scale, minScale), maxScale)
	color, ok := Colors[strings.ToLower(s.Color)]
	if !ok {
		color = Colors["white"]
	}
	pos, ok := Positions[strings.ToLower(s.Position)]
	if !ok {
		pos = Positions["bottom"]
	}
	props := [][2]string{
		{"sub-scale", trimFloat(scale)},
		{"sub-color", color},
		{"sub-pos", strconv.Itoa(pos)},
	}
	switch strings.ToLower(s.Background) {
	case "none":
		props = append(props,
			[2]string{"sub-border-style", "outline-and-shadow"},
			[2]string{"sub-border-size", "0"},
			[2]string{"sub-shadow-offset", "0"},
		)
	case "box":
		props = append(props,
			[2]string{"sub-border-style", "opaque-box"},
			[2]string{"sub-border-size", "1.5"},
			[2]string{"sub-back-color", "#B3000000"},
		)
	default: // outline
		props = append(props,
			[2]string{"sub-border-style", "outline-and-shadow"},
			[2]string{"sub-border-size", "3"},
			[2]string{"sub-shadow-offset", "1"},
			[2]string{"sub-back-color", "#00000000"},
		)
	}
	return props
}

func trimFloat(f float64) string {
	s := strconv.FormatFloat(f, 'f', 3, 64)
	s = strings.TrimRight(s, "0")
	return strings.TrimSuffix(s, ".")
}
