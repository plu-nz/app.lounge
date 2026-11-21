package main

import (
	"encoding/json"
	"log"

	"github.com/plu-nz/app.lounge/internal/avopts"
	"github.com/plu-nz/app.lounge/internal/settings"
)

// AudioDevice is an output mpv can play to.
type AudioDevice struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ApplyAV pushes the audio output and HDR preferences to mpv. Properties mpv
// refuses (for example a device that was unplugged) are logged and skipped so
// the others still apply; the rest take effect live or on the next file.
func (s *PlayerService) ApplyAV(prefs settings.Settings) {
	for _, p := range avopts.Properties(avopts.Prefs{
		AudioOutput: prefs.AudioOutput,
		NightMode:   prefs.NightMode,
		AudioDevice: prefs.AudioDevice,
		HDR:         prefs.HDR,
	}) {
		if err := s.player.SetProperty(p.Name, p.Value); err != nil {
			log.Printf("audio/hdr: %v", err)
		}
	}
}

// AudioDevices lists mpv's audio outputs.
func (s *PlayerService) AudioDevices() ([]AudioDevice, error) {
	raw, err := s.player.Property("audio-device-list")
	if err != nil {
		return nil, err
	}
	var list []AudioDevice
	return list, json.Unmarshal(raw, &list)
}
