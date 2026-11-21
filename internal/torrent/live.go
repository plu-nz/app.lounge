package torrent

import (
	"time"

	"github.com/anacrolix/torrent/metainfo"
)

// Stats returns a live snapshot of the active torrent. Unlike Status it also
// works before the metadata arrives. Active is false when nothing is loading.
func (e *Engine) Stats() Stats {
	return e.statsAt(time.Now())
}

func (e *Engine) statsAt(now time.Time) Stats {
	e.mu.Lock()
	sess := e.sess
	var h metainfo.Hash
	if sess != nil {
		_ = h.FromHexString(sess.hash)
	}
	t := e.active[h]
	e.mu.Unlock()
	if sess == nil || t == nil {
		return Stats{}
	}
	sess.mu.Lock()
	defer sess.mu.Unlock()

	ts := t.Stats()
	st := Stats{
		Active: true, InfoHash: sess.hash, Name: t.Name(),
		Peers: ts.ActivePeers, TotalPeers: ts.TotalPeers, Seeders: ts.ConnectedSeeders,
		DownloadRate: sess.download.Sample(ts.BytesReadUsefulData.Int64(), now),
		UploadRate:   sess.upload.Sample(ts.BytesWrittenData.Int64(), now),
	}
	info := t.Info()
	st.Phase = phaseOf(info != nil, st.Peers)
	st.NoSeeders = sess.dog.Observe(now, info != nil, st.Peers)
	if info == nil {
		return st
	}
	st.Size = t.Length()
	st.Downloaded = t.BytesCompleted()
	if st.Size > 0 {
		st.Progress = float64(st.Downloaded) / float64(st.Size)
	}
	files := t.Files()
	if sess.fileIdx >= 0 && sess.fileIdx < len(files) {
		f := files[sess.fileIdx]
		st.FileSize = f.Length()
		st.BufferedBytes = bufferedAhead(info.PieceLength, f.Offset(), f.Length(), sess.pos.Load(), t.NumPieces(),
			func(i int) bool { return t.Piece(i).State().Complete })
		st.BufferedPercent = bufferPercent(st.BufferedBytes, st.FileSize)
	}
	return st
}

// KeepWaiting restarts the no-seeders timer after the user chose to wait.
func (e *Engine) KeepWaiting() {
	e.mu.Lock()
	sess := e.sess
	e.mu.Unlock()
	if sess == nil {
		return
	}
	sess.mu.Lock()
	sess.dog.Reset(time.Now())
	sess.mu.Unlock()
}

// StartStats emits the active torrent's stats every second until Close.
// Nothing is emitted while no torrent is loading.
func (e *Engine) StartStats(emit func(Stats)) {
	go func() {
		tick := time.NewTicker(time.Second)
		defer tick.Stop()
		for {
			select {
			case <-e.quit:
				return
			case <-tick.C:
				if st := e.Stats(); st.Active {
					emit(st)
				}
			}
		}
	}()
}
