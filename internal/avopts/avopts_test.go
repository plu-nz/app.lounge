package avopts

import "testing"

func get(props []Property, name string) string {
	for _, p := range props {
		if p.Name == name {
			return p.Value
		}
	}
	return "<missing>"
}

func TestDefaultsAreStereoAndTonemap(t *testing.T) {
	p := Properties(Prefs{})
	for name, want := range map[string]string{
		"audio-spdif": "", "audio-channels": "stereo", "af": "", "audio-device": "auto",
		"hdr-compute-peak": "yes", "tone-mapping": "auto", "target-colorspace-hint": "no",
	} {
		if got := get(p, name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
}

func TestAutoKeepsMultichannel(t *testing.T) {
	p := Properties(Prefs{AudioOutput: AudioAuto})
	if get(p, "audio-channels") != "auto-safe" || get(p, "audio-spdif") != "" {
		t.Errorf("unexpected %v", p)
	}
}

func TestPassthroughAndNightMode(t *testing.T) {
	p := Properties(Prefs{AudioOutput: AudioPassthrough, NightMode: true})
	if get(p, "audio-spdif") != "ac3,eac3,dts,dts-hd,truehd" {
		t.Errorf("spdif = %q", get(p, "audio-spdif"))
	}
	if get(p, "af") != "" {
		t.Error("night filter must not apply to passthrough")
	}
	if get(Properties(Prefs{NightMode: true}), "af") == "" {
		t.Error("night mode should set a filter")
	}
}

func TestDeviceAndHDROff(t *testing.T) {
	p := Properties(Prefs{AudioDevice: " alsa/hdmi:CARD=Generic,DEV=0 ", HDR: HDROff})
	if get(p, "audio-device") != "alsa/hdmi:CARD=Generic,DEV=0" {
		t.Errorf("device = %q", get(p, "audio-device"))
	}
	if get(p, "hdr-compute-peak") != "no" || get(p, "tone-mapping") != "clip" {
		t.Errorf("hdr off: %v", p)
	}
}
