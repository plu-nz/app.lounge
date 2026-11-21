package subtitle

import "testing"

func props(s Style) map[string]string {
	m := map[string]string{}
	for _, p := range Properties(s) {
		m[p[0]] = p[1]
	}
	return m
}

func check(t *testing.T, got, want map[string]string) {
	t.Helper()
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
}

func TestDefaults(t *testing.T) {
	check(t, props(Style{}), map[string]string{
		"sub-scale": "1", "sub-color": "#FFFFFF", "sub-pos": "100",
		"sub-border-style": "outline-and-shadow", "sub-border-size": "3",
	})
}

func TestBox(t *testing.T) {
	check(t, props(Style{Scale: 1.3, Color: "Yellow", Background: "box", Position: "high"}), map[string]string{
		"sub-scale": "1.3", "sub-color": "#FFE94A", "sub-pos": "86",
		"sub-border-style": "opaque-box", "sub-back-color": "#B3000000",
	})
}

func TestNoneRemovesOutline(t *testing.T) {
	check(t, props(Style{Background: "none"}), map[string]string{"sub-border-size": "0", "sub-shadow-offset": "0"})
}

func TestClampsAndFallsBack(t *testing.T) {
	check(t, props(Style{Scale: 99, Color: "magenta", Position: "x", Background: "?"}), map[string]string{
		"sub-scale": "3", "sub-color": "#FFFFFF", "sub-pos": "100", "sub-border-size": "3",
	})
}
