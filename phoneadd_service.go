package main

import (
	"context"
	"encoding/base64"
	"sync"

	qrcode "github.com/skip2/go-qrcode"

	"github.com/plu-nz/app.lounge/internal/phoneadd"
	"github.com/plu-nz/app.lounge/internal/stremio"
)

// PhoneSession is what the UI shows while waiting for the phone.
type PhoneSession struct {
	URL    string `json:"url"`
	QRCode string `json:"qrcode"` // PNG data URL
}

// PhoneAddService lets a phone on the local network install addons.
type PhoneAddService struct {
	addons *stremio.Addons
	client *stremio.Client
	emit   func(name string, data ...any) // set after the app exists

	mu  sync.Mutex
	cur *phoneadd.Server
}

// Start opens a short-lived LAN server and returns its link and QR code.
// Results arrive as "phoneadd" events: {ok, name, error}; a final
// {stopped: true} event follows when the session ends.
func (s *PhoneAddService) Start() (PhoneSession, error) {
	if s.addons.Managed() {
		return PhoneSession{}, stremio.ErrManaged
	}
	s.Stop()
	ip, err := phoneadd.LANAddr()
	if err != nil {
		return PhoneSession{}, err
	}
	var srv *phoneadd.Server
	srv, link, err := phoneadd.Start(phoneadd.Options{
		IP:        ip,
		Normalize: stremio.NormalizeURL,
		Install: func(ctx context.Context, u string) (string, error) {
			// The manifest fetch inside Install validates the link.
			ad, err := s.addons.Install(ctx, u)
			if err != nil {
				return "", err
			}
			return ad.Manifest.Name, nil
		},
		OnResult: func(r phoneadd.Result) {
			s.send(map[string]any{"ok": r.OK, "name": r.Name, "error": r.Error})
		},
		OnStop: func() {
			s.mu.Lock()
			if s.cur == srv {
				s.cur = nil
			}
			s.mu.Unlock()
			s.send(map[string]any{"stopped": true})
		},
	})
	if err != nil {
		return PhoneSession{}, err
	}
	png, err := qrcode.Encode(link, qrcode.Medium, 512)
	if err != nil {
		srv.Stop()
		return PhoneSession{}, err
	}
	s.mu.Lock()
	s.cur = srv
	s.mu.Unlock()
	return PhoneSession{URL: link, QRCode: "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)}, nil
}

// Stop ends the session, if any.
func (s *PhoneAddService) Stop() {
	s.mu.Lock()
	cur := s.cur
	s.mu.Unlock()
	if cur != nil {
		cur.Stop()
	}
}

func (s *PhoneAddService) send(data map[string]any) {
	if s.emit != nil {
		s.emit("phoneadd", data)
	}
}

