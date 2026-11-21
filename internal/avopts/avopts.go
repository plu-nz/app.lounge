// Package avopts maps the audio and HDR preferences onto mpv properties.
//
// HDR: only tone-mapping to SDR is offered. True HDR passthrough needs mpv to
// own the swapchain (vo=gpu-next with the Vulkan or Wayland color-management
// context) so that target-colorspace-hint can tell the compositor the
// content's colorspace. Lounge renders through libmpv's OpenGL render API into
// a GtkGLArea under WebKitGTK: GTK owns the surface, exposes no HDR/colorspace
// control for it, and the web UI is composited over the video as SDR, so
// target-colorspace-hint has nothing to act on. HDR video is therefore always
// mapped into the SDR range of the display.
package avopts

import "strings"

// Audio output modes.
const (
	AudioStereo      = "stereo"
	AudioAuto        = "auto"
	AudioPassthrough = "passthrough"
)

// HDR modes.
const (
	HDRTonemap = "tonemap"
	HDROff     = "off"
)

// Prefs are the preferences that affect audio output and HDR handling.
type Prefs struct {
	AudioOutput string // stereo (default), auto, passthrough
	NightMode   bool
	AudioDevice string // mpv device name; empty or "auto" for the default
	HDR         string // tonemap (default), off
}

// Property is one mpv property assignment.
type Property struct{ Name, Value string }

// passthroughCodecs are the codecs sent undecoded over HDMI/S/PDIF.
const passthroughCodecs = "ac3,eac3,dts,dts-hd,truehd"

// nightFilter compresses dynamic range: quiet dialogue is lifted and loud
// effects are held back. The filter is skipped for passthrough, which cannot
// be processed.
const nightFilter = "lavfi=[acompressor=threshold=0.089:ratio=6:attack=20:release=300:makeup=4]"

// Properties returns the mpv properties for the preferences, in apply order.
// Unknown values fall back to the defaults.
func Properties(p Prefs) []Property {
	var props []Property
	switch p.AudioOutput {
	case AudioPassthrough:
		props = append(props, Property{"audio-spdif", passthroughCodecs}, Property{"audio-channels", "auto"})
	case AudioAuto:
		props = append(props, Property{"audio-spdif", ""}, Property{"audio-channels", "auto-safe"})
	default:
		props = append(props, Property{"audio-spdif", ""}, Property{"audio-channels", "stereo"})
	}
	filter := ""
	if p.NightMode && p.AudioOutput != AudioPassthrough {
		filter = nightFilter
	}
	props = append(props, Property{"af", filter})
	device := strings.TrimSpace(p.AudioDevice)
	if device == "" {
		device = "auto"
	}
	props = append(props, Property{"audio-device", device})

	// target-colorspace-hint stays off: see the package comment.
	props = append(props, Property{"target-colorspace-hint", "no"})
	if p.HDR == HDROff {
		props = append(props, Property{"hdr-compute-peak", "no"}, Property{"tone-mapping", "clip"})
	} else {
		props = append(props, Property{"hdr-compute-peak", "yes"}, Property{"tone-mapping", "auto"})
	}
	return props
}
