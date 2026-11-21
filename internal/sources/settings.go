package sources

import "github.com/plu-nz/app.lounge/internal/settings"

// Auto reports whether the user chose automatic source selection.
func Auto(s settings.Settings) bool { return s.SourceMode == "auto" }

// RulesFrom reads the source selection rules from the preferences.
func RulesFrom(s settings.Settings) Rules {
	return Rules{
		MaxHeight:  s.SourceMaxHeight,
		MaxSizeGB:  float64(s.SourceMaxSizeGB),
		Prefer:     s.SourcePrefer,
		MinSeeders: s.SourceMinSeeders,
	}
}
