package torrent

import (
	"io"
	"sync"
	"sync/atomic"
	"time"
)

// Phase says what the active torrent is doing before playback can start.
type Phase string

const (
	// PhaseConnecting means no peer is connected yet.
	PhaseConnecting Phase = "connecting"
	// PhaseMetadata means peers are connected but the torrent info is missing.
	PhaseMetadata Phase = "metadata"
	// PhaseDownloading means the file list is known and pieces are downloading.
	PhaseDownloading Phase = "downloading"
)

const (
	// bufferTarget is the amount of contiguous data ahead of the playback
	// position that counts as a full buffer in the UI.
	bufferTarget = 16 << 20
	// NoSeedersTimeout is how long a torrent may stay without peers or
	// metadata before the UI offers another source.
	NoSeedersTimeout = 30 * time.Second
	// maxAheadScan bounds how many pieces bufferedAhead looks at.
	maxAheadScan = 4096
)

// Stats is a live snapshot of the active torrent, sent to the UI.
type Stats struct {
	Active          bool    `json:"active"`
	InfoHash        string  `json:"infoHash"`
	Name            string  `json:"name"`
	Phase           Phase   `json:"phase"`
	Peers           int     `json:"peers"`      // connected peers
	TotalPeers      int     `json:"totalPeers"` // known peers
	Seeders         int     `json:"seeders"`
	DownloadRate    float64 `json:"downloadRate"` // bytes per second
	UploadRate      float64 `json:"uploadRate"`
	Downloaded      int64   `json:"downloaded"`
	Size            int64   `json:"size"`
	FileSize        int64   `json:"fileSize"`
	Progress        float64 `json:"progress"`        // 0..1 of the whole torrent
	BufferedBytes   int64   `json:"bufferedBytes"`   // contiguous bytes ahead of the read position
	BufferedPercent int     `json:"bufferedPercent"` // 0..100 of the buffer target
	NoSeeders       bool    `json:"noSeeders"`
}

// RateTracker turns a growing byte counter into bytes per second.
type RateTracker struct {
	bytes int64
	at    time.Time
	set   bool
}

// Sample records the counter at now and returns the rate since the last
// sample. The first sample returns 0.
func (r *RateTracker) Sample(bytes int64, now time.Time) float64 {
	if !r.set {
		r.bytes, r.at, r.set = bytes, now, true
		return 0
	}
	dt := now.Sub(r.at).Seconds()
	if dt <= 0 {
		return 0
	}
	rate := float64(bytes-r.bytes) / dt
	r.bytes, r.at = bytes, now
	if rate < 0 {
		return 0
	}
	return rate
}

// phaseOf names the loading stage from what is known about the swarm.
func phaseOf(hasInfo bool, peers int) Phase {
	switch {
	case hasInfo:
		return PhaseDownloading
	case peers > 0:
		return PhaseMetadata
	default:
		return PhaseConnecting
	}
}

// bufferedAhead returns how many contiguous bytes of a file are downloaded
// from pos (an offset inside the file). complete reports whether a piece
// index is fully downloaded. The file starts at fileOffset in the torrent.
func bufferedAhead(pieceLen, fileOffset, fileLen, pos int64, numPieces int, complete func(int) bool) int64 {
	if pieceLen <= 0 || fileLen <= 0 {
		return 0
	}
	pos = min(max(pos, 0), fileLen)
	abs := fileOffset + pos
	end := fileOffset + fileLen
	ahead := int64(0)
	for scanned := 0; abs < end && scanned < maxAheadScan; scanned++ {
		idx := int(abs / pieceLen)
		if idx >= numPieces || !complete(idx) {
			break
		}
		next := min(int64(idx+1)*pieceLen, end)
		ahead += next - abs
		abs = next
	}
	return ahead
}

// bufferPercent is the share of the buffer target that is filled.
func bufferPercent(buffered, fileLen int64) int {
	target := min(int64(bufferTarget), fileLen)
	if target <= 0 {
		return 0
	}
	return int(min(100, buffered*100/target))
}

// Watchdog detects a torrent that finds no peers or metadata for too long.
// It is a pure state machine: the caller passes the clock.
type Watchdog struct {
	Timeout time.Duration
	since   time.Time
	started bool
}

// Observe feeds the current swarm state and reports whether the torrent has
// been stuck (no connected peer, or no metadata) for at least Timeout.
func (w *Watchdog) Observe(now time.Time, hasInfo bool, peers int) bool {
	if hasInfo && peers > 0 {
		w.started = false
		return false
	}
	if !w.started {
		w.since, w.started = now, true
	}
	return now.Sub(w.since) >= w.Timeout
}

// Reset restarts the wait, for "Keep waiting".
func (w *Watchdog) Reset(now time.Time) {
	w.since, w.started = now, true
}

// session is the torrent whose stats are reported.
type session struct {
	mu       sync.Mutex
	hash     string
	download RateTracker
	upload   RateTracker
	dog      Watchdog
	fileIdx  int
	pos      atomic.Int64 // read offset in the streamed file
}

// posReader records where the HTTP server reads in the file, which is the
// playback position as far as the torrent can tell.
type posReader struct {
	io.ReadSeeker
	pos *atomic.Int64
}

func (r posReader) Read(p []byte) (int, error) {
	n, err := r.ReadSeeker.Read(p)
	r.pos.Add(int64(n))
	return n, err
}

func (r posReader) Seek(off int64, whence int) (int64, error) {
	n, err := r.ReadSeeker.Seek(off, whence)
	if err == nil {
		r.pos.Store(n)
	}
	return n, err
}
