package sources

import (
	"testing"

	"github.com/plu-nz/app.lounge/internal/settings"
)

func TestRulesFromSettings(t *testing.T) {
	if Auto(settings.Defaults()) {
		t.Fatal("defaults must ask")
	}
	s := settings.Settings{SourceMode: "auto", SourceMaxHeight: 1080, SourceMaxSizeGB: 10, SourcePrefer: "direct", SourceMinSeeders: 5}
	if !Auto(s) {
		t.Fatal("auto mode not detected")
	}
	if got, want := RulesFrom(s), (Rules{MaxHeight: 1080, MaxSizeGB: 10, Prefer: "direct", MinSeeders: 5}); got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
