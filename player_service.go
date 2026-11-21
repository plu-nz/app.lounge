package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/plu-nz/app.lounge/internal/player"
	"github.com/plu-nz/app.lounge/internal/stremio"
	"github.com/plu-nz/app.lounge/internal/subtitle"
	"github.com/plu-nz/app.lounge/internal/torrent"
)

// PlayerService exposes playback controls to the UI.
type PlayerService struct {
	player   *player.Player
	torrents *torrent.Engine
}

// PlayRequest describes what to play.
type PlayRequest struct {
	URL       string             `json:"url"`
	InfoHash  string             `json:"infoHash"`
	FileIdx   *int               `json:"fileIdx"`
	Filename  string             `json:"filename"`
	Sources   []string           `json:"sources"`
	Headers   map[string]string  `json:"headers"`
	Start     float64            `json:"start"`
	Title     string             `json:"title"`
	Subtitles []stremio.Subtitle `json:"subtitles"`
}

// Load resolves a stream and starts it.
func (s *PlayerService) Load(req PlayRequest) error {
	url := req.URL
	if url == "" && req.InfoHash != "" {
		if s.torrents == nil {
			return fmt.Errorf("torrent streaming is unavailable")
		}
		idx := -1
		if req.FileIdx != nil {
			idx = *req.FileIdx
		}
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		u, err := s.torrents.Prepare(ctx, req.InfoHash, idx, req.Filename, req.Sources)
		if err != nil {
			return err
		}
		url = u
	}
	if url == "" {
		return fmt.Errorf("the stream has no playable URL")
	}
	var headers []string
	for k, v := range req.Headers {
		headers = append(headers, k+": "+strings.ReplaceAll(v, ",", `\,`))
	}
	props := [][2]string{
		{"http-header-fields", strings.Join(headers, ",")},
		{"start", "none"},
		{"force-media-title", req.Title},
		{"pause", "no"},
		// The subtitle delay belongs to one video, so a new one starts at 0.
		{"sub-delay", "0"},
	}
	if req.Start > 0 {
		props[1][1] = fmt.Sprintf("%.3f", req.Start)
	}
	for _, p := range props {
		if err := s.player.SetProperty(p[0], p[1]); err != nil {
			return err
		}
	}
	if err := s.player.Command("loadfile", url, "replace"); err != nil {
		return err
	}
	for _, sub := range req.Subtitles {
		// "auto" keeps the selected track; the title shows the language.
		_ = s.player.Command("sub-add", sub.URL, "auto", sub.Lang, sub.Lang)
	}
	return nil
}

// TorrentStatus reports download progress for a torrent stream.
func (s *PlayerService) TorrentStatus(infoHash string) (*torrent.Status, error) {
	if s.torrents == nil {
		return nil, nil
	}
	st, ok := s.torrents.Status(infoHash)
	if !ok {
		return nil, nil
	}
	return &st, nil
}

// Command runs a raw mpv command, for example ["seek", "10"].
func (s *PlayerService) Command(args []string) error {
	return s.player.Command(args...)
}

// SetProperty sets an mpv property from its string form.
func (s *PlayerService) SetProperty(name, value string) error {
	return s.player.SetProperty(name, value)
}

// Property returns an mpv property as a JSON string.
func (s *PlayerService) Property(name string) (string, error) {
	v, err := s.player.Property(name)
	return string(v), err
}

// Stop ends playback and frees torrent data.
func (s *PlayerService) Stop() error {
	err := s.player.Command("stop")
	if s.torrents != nil {
		go s.torrents.DropAll()
	}
	return err
}

// SetSubtitleStyle applies the subtitle appearance to mpv.
func (s *PlayerService) SetSubtitleStyle(style subtitle.Style) error {
	for _, p := range subtitle.Properties(style) {
		if err := s.player.SetProperty(p[0], p[1]); err != nil {
			return err
		}
	}
	return nil
}
