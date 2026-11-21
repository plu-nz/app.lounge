// Package phoneadd runs a short-lived HTTP server on the local network so a
// phone can send an addon manifest URL to Lounge.
package phoneadd

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

//go:embed page.html
var page []byte

// Defaults for a session.
const (
	DefaultTTL   = 10 * time.Minute
	maxBodyBytes = 4 << 10
	maxAttempts  = 20
)

// Result reports one attempt to add an addon.
type Result struct {
	OK    bool   `json:"ok"`
	Name  string `json:"name,omitempty"`
	Error string `json:"error,omitempty"`
}

// Options configures a Server.
type Options struct {
	// IP is the only address the server listens on.
	IP net.IP
	// TTL is how long the server lives; zero means DefaultTTL.
	TTL time.Duration
	// Normalize validates and normalizes the submitted link.
	Normalize func(raw string) (string, error)
	// Install checks the manifest and installs the addon, returning its name.
	Install func(ctx context.Context, manifestURL string) (string, error)
	// OnResult is called after every attempt (optional).
	OnResult func(Result)
	// OnStop is called once when the server stops (optional).
	OnStop func()
}

// Server is one phone-add session.
type Server struct {
	opts  Options
	token string
	ln    net.Listener
	srv   *http.Server
	host  string // Host header the phone must use; empty skips the check

	mu       sync.Mutex
	busy     bool
	attempts int
	done     bool
	stopped  chan struct{}
	timer    *time.Timer
}

// New builds a server without listening; use Handler in tests or Start.
func New(opts Options) (*Server, error) {
	if opts.TTL <= 0 {
		opts.TTL = DefaultTTL
	}
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	s := &Server{opts: opts, token: base64.RawURLEncoding.EncodeToString(b), stopped: make(chan struct{})}
	s.srv = &http.Server{
		Handler:           s.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      45 * time.Second,
		MaxHeaderBytes:    8 << 10,
	}
	return s, nil
}

// Start listens on opts.IP with a random port and returns the phone URL.
func Start(opts Options) (*Server, string, error) {
	if opts.IP == nil {
		return nil, "", errors.New("no listen address")
	}
	s, err := New(opts)
	if err != nil {
		return nil, "", err
	}
	ln, err := net.Listen("tcp4", net.JoinHostPort(opts.IP.String(), "0"))
	if err != nil {
		return nil, "", err
	}
	s.ln = ln
	s.host = ln.Addr().String()
	go s.srv.Serve(ln)
	s.timer = time.AfterFunc(s.opts.TTL, s.Stop)
	return s, s.URL(), nil
}

// URL is the address the phone opens.
func (s *Server) URL() string {
	return "http://" + s.ln.Addr().String() + "/" + s.token + "/"
}

// Token returns the secret path segment.
func (s *Server) Token() string { return s.token }

// Done is closed once the server has stopped.
func (s *Server) Done() <-chan struct{} { return s.stopped }

// Stop shuts the server down; it is safe to call more than once.
func (s *Server) Stop() {
	s.mu.Lock()
	if s.done {
		s.mu.Unlock()
		return
	}
	s.done = true
	if s.timer != nil {
		s.timer.Stop()
	}
	s.mu.Unlock()
	if s.ln != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		s.srv.Shutdown(ctx)
		cancel()
	}
	close(s.stopped)
	if s.opts.OnStop != nil {
		s.opts.OnStop()
	}
}

func (s *Server) finished() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.done
}

// Handler serves the phone page and its API.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	prefix := "/" + s.token + "/"
	mux.HandleFunc("GET "+prefix, s.handlePage)
	mux.HandleFunc("POST "+prefix+"add", s.handleAdd)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// Anything but the exact token path, or a wrong Host (DNS rebinding),
		// looks like a missing page.
		if s.finished() || !strings.HasPrefix(r.URL.Path, prefix) || (s.host != "" && r.Host != s.host) {
			http.NotFound(w, r)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func (s *Server) handlePage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/"+s.token+"/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; connect-src 'self'; base-uri 'none'; form-action 'none'")
	w.Write(page)
}

func reply(w http.ResponseWriter, code int, res Result) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(res)
}

func (s *Server) handleAdd(w http.ResponseWriter, r *http.Request) {
	// Browsers send Origin on cross-site POSTs; only the page itself may post.
	if o := r.Header.Get("Origin"); o != "" {
		if u, err := url.Parse(o); err != nil || u.Host != r.Host {
			reply(w, http.StatusForbidden, Result{Error: "forbidden"})
			return
		}
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		reply(w, http.StatusUnsupportedMediaType, Result{Error: "expected JSON"})
		return
	}
	var body struct {
		URL string `json:"url"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		code := http.StatusBadRequest
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			code = http.StatusRequestEntityTooLarge
		}
		reply(w, code, Result{Error: "invalid request"})
		return
	}

	s.mu.Lock()
	switch {
	case s.done:
		s.mu.Unlock()
		reply(w, http.StatusGone, Result{Error: "session ended"})
		return
	case s.busy:
		s.mu.Unlock()
		reply(w, http.StatusTooManyRequests, Result{Error: "busy"})
		return
	case s.attempts >= maxAttempts:
		s.mu.Unlock()
		reply(w, http.StatusTooManyRequests, Result{Error: "too many attempts"})
		return
	}
	s.busy = true
	s.attempts++
	s.mu.Unlock()

	res := s.add(r.Context(), body.URL)

	s.mu.Lock()
	s.busy = false
	s.mu.Unlock()
	if s.opts.OnResult != nil {
		s.opts.OnResult(res)
	}
	if res.OK {
		reply(w, http.StatusOK, res)
		// One-time use: the session ends after the first success.
		go s.Stop()
		return
	}
	reply(w, http.StatusUnprocessableEntity, res)
}

func (s *Server) add(ctx context.Context, raw string) Result {
	norm, err := s.opts.Normalize(raw)
	if err != nil {
		return Result{Error: err.Error()}
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	name, err := s.opts.Install(ctx, norm)
	if err != nil {
		return Result{Error: err.Error()}
	}
	return Result{OK: true, Name: name}
}
