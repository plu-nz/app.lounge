// Package torrent streams files from BitTorrent swarms over local HTTP so
// mpv can play them while they download.
package torrent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	alog "github.com/anacrolix/log"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/storage"
)

// defaultTrackers help find peers when an addon lists no sources.
var defaultTrackers = []string{
	"udp://tracker.opentrackr.org:1337/announce",
	"udp://open.demonii.com:1337/announce",
	"udp://open.stealth.si:80/announce",
	"udp://tracker.torrent.eu.org:451/announce",
	"udp://exodus.desync.com:6969/announce",
	"udp://tracker.openbittorrent.com:6969/announce",
}

var videoExt = []string{".mkv", ".mp4", ".avi", ".m4v", ".mov", ".webm", ".ts", ".wmv", ".flv", ".mpg", ".mpeg"}

// Engine owns the BitTorrent client and the local HTTP server.
type Engine struct {
	client  *torrent.Client
	dataDir string
	addr    string
	server  *http.Server

	mu     sync.Mutex
	active map[metainfo.Hash]*torrent.Torrent

	policy    func() CachePolicy // see cache.go
	evictStop chan struct{}
	sess      *session      // the torrent whose stats are reported
	abort     chan struct{} // closed by DropAll to cancel waiting Prepare calls
	quit      chan struct{} // closed by Close to stop the stats loop
}

// Status reports download progress for the UI.
type Status struct {
	InfoHash      string  `json:"infoHash"`
	Name          string  `json:"name"`
	Peers         int     `json:"peers"`
	Seeders       int     `json:"seeders"`
	DownloadSpeed float64 `json:"downloadSpeed"` // bytes per second
	Downloaded    int64   `json:"downloaded"`
	Size          int64   `json:"size"`
}

// Start creates the client with its data under cacheDir. Data from earlier
// runs stays as a cache that evict.go trims; Lounge streams and does not seed.
func Start(cacheDir string) (*Engine, error) {
	dataDir := filepath.Join(cacheDir, "torrents")
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, err
	}
	cfg := torrent.NewDefaultClientConfig()
	cfg.DataDir = dataDir
	cfg.DefaultStorage = storage.NewFileByInfoHash(dataDir)
	cfg.ListenPort = 0
	cfg.Seed = false
	cfg.Logger = cfg.Logger.WithFilterLevel(alog.Error)
	client, err := torrent.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("torrent client: %w", err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		client.Close()
		return nil, err
	}
	e := &Engine{client: client, dataDir: dataDir, addr: ln.Addr().String(), active: map[metainfo.Hash]*torrent.Torrent{}, abort: make(chan struct{}), quit: make(chan struct{})}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /stream/{hash}/{idx}", e.serveFile)
	mux.HandleFunc("HEAD /stream/{hash}/{idx}", e.serveFile)
	e.server = &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		if err := e.server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("torrent http: %v", err)
		}
	}()
	return e, nil
}

// Prepare adds a torrent, waits for its metadata, and returns a local URL
// for the chosen file. fileIdx < 0 picks the file matching filename, or
// the largest video.
func (e *Engine) Prepare(ctx context.Context, infoHash string, fileIdx int, filename string, sources []string) (string, error) {
	var h metainfo.Hash
	if err := h.FromHexString(strings.ToLower(infoHash)); err != nil {
		return "", fmt.Errorf("bad info hash %q: %w", infoHash, err)
	}
	t, _ := e.client.AddTorrentInfoHash(h)
	var trackers []string
	for _, s := range sources {
		if tr, ok := strings.CutPrefix(s, "tracker:"); ok {
			trackers = append(trackers, tr)
		}
	}
	trackers = append(trackers, defaultTrackers...)
	tiers := make([][]string, len(trackers))
	for i, tr := range trackers {
		tiers[i] = []string{tr}
	}
	t.AddTrackers(tiers)
	e.mu.Lock()
	e.active[h] = t
	abort := e.abort
	sess := &session{hash: h.HexString(), fileIdx: -1, dog: Watchdog{Timeout: NoSeedersTimeout}}
	sess.dog.Reset(time.Now())
	e.sess = sess
	e.mu.Unlock()
	touchEntry(e.dataDir, h.HexString())

	select {
	case <-t.GotInfo():
	case <-abort:
		return "", errors.New("cancelled")
	case <-ctx.Done():
		return "", fmt.Errorf("no peers sent the torrent metadata: %w", ctx.Err())
	}
	idx := pickFile(t.Files(), fileIdx, filename)
	if idx < 0 {
		return "", errors.New("the torrent has no video file")
	}
	sess.mu.Lock()
	sess.fileIdx = idx
	sess.mu.Unlock()
	return fmt.Sprintf("http://%s/stream/%s/%d", e.addr, h.HexString(), idx), nil
}

// Status returns progress for an active torrent.
func (e *Engine) Status(infoHash string) (Status, bool) {
	var h metainfo.Hash
	if err := h.FromHexString(strings.ToLower(infoHash)); err != nil {
		return Status{}, false
	}
	e.mu.Lock()
	t, ok := e.active[h]
	e.mu.Unlock()
	if !ok || t.Info() == nil {
		return Status{}, false
	}
	st := t.Stats()
	return Status{
		InfoHash: infoHash, Name: t.Name(),
		Peers: st.ActivePeers, Seeders: st.ConnectedSeeders,
		DownloadSpeed: downloadRate(t),
		Downloaded:    t.BytesCompleted(), Size: t.Length(),
	}, true
}

// DropAll stops every torrent. Its data stays in the cache for the next
// play and is trimmed by Evict.
func (e *Engine) DropAll() {
	e.mu.Lock()
	for h, t := range e.active {
		t.Drop()
		touchEntry(e.dataDir, h.HexString())
		delete(e.active, h)
	}
	e.sess = nil
	close(e.abort)
	e.abort = make(chan struct{})
	e.mu.Unlock()
	e.Evict()
}

// Close stops the server and the client. Cached data is removed only when
// the cache policy says to clear on exit.
func (e *Engine) Close() {
	if e.evictStop != nil {
		close(e.evictStop)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	close(e.quit)
	_ = e.server.Shutdown(ctx)
	e.DropAll()
	e.client.Close()
	if e.cachePolicy().ClearOnExit {
		_, _ = e.ClearCache()
	}
}

func (e *Engine) serveFile(w http.ResponseWriter, r *http.Request) {
	var h metainfo.Hash
	if err := h.FromHexString(r.PathValue("hash")); err != nil {
		http.Error(w, "bad hash", http.StatusBadRequest)
		return
	}
	idx, err := strconv.Atoi(r.PathValue("idx"))
	e.mu.Lock()
	t, ok := e.active[h]
	e.mu.Unlock()
	if !ok || err != nil || t.Info() == nil || idx < 0 || idx >= len(t.Files()) {
		http.NotFound(w, r)
		return
	}
	f := t.Files()[idx]
	reader := f.NewReader()
	defer reader.Close()
	reader.SetContext(r.Context())
	reader.SetResponsive()
	reader.SetReadahead(f.Length() / 100)
	w.Header().Set("Content-Type", contentType(f.DisplayPath()))
	var content io.ReadSeeker = reader
	e.mu.Lock()
	if e.sess != nil && e.sess.hash == h.HexString() {
		content = posReader{ReadSeeker: reader, pos: &e.sess.pos}
	}
	e.mu.Unlock()
	http.ServeContent(w, r, path.Base(f.DisplayPath()), time.Time{}, content)
}

func pickFile(files []*torrent.File, fileIdx int, filename string) int {
	if fileIdx >= 0 && fileIdx < len(files) {
		return fileIdx
	}
	if filename != "" {
		if i := slices.IndexFunc(files, func(f *torrent.File) bool {
			return path.Base(f.DisplayPath()) == filename
		}); i >= 0 {
			return i
		}
	}
	best, size := -1, int64(-1)
	for i, f := range files {
		if isVideo(f.DisplayPath()) && f.Length() > size {
			best, size = i, f.Length()
		}
	}
	return best
}

func isVideo(name string) bool {
	return slices.Contains(videoExt, strings.ToLower(filepath.Ext(name)))
}

func contentType(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".mp4", ".m4v":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".mkv":
		return "video/x-matroska"
	default:
		return "application/octet-stream"
	}
}

// rates remembers the last byte count per torrent to compute speed.
var rates sync.Map // *torrent.Torrent -> rateSample

type rateSample struct {
	bytes int64
	at    time.Time
}

func downloadRate(t *torrent.Torrent) float64 {
	now := time.Now()
	read := t.Stats().BytesReadUsefulData
	cur := rateSample{bytes: read.Int64(), at: now}
	prev, ok := rates.Swap(t, cur)
	if !ok {
		return 0
	}
	p := prev.(rateSample)
	dt := now.Sub(p.at).Seconds()
	if dt <= 0 {
		return 0
	}
	return float64(cur.bytes-p.bytes) / dt
}
