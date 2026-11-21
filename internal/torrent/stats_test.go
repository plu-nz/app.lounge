package torrent

import (
	"testing"
	"time"
)

func TestRateTracker(t *testing.T) {
	var r RateTracker
	t0 := time.Unix(1000, 0)
	if got := r.Sample(0, t0); got != 0 {
		t.Fatalf("first sample = %v", got)
	}
	if got := r.Sample(2000, t0.Add(time.Second)); got != 2000 {
		t.Fatalf("rate = %v", got)
	}
	if got := r.Sample(2500, t0.Add(1500*time.Millisecond)); got != 1000 {
		t.Fatalf("rate = %v", got)
	}
	if got := r.Sample(100, t0.Add(3*time.Second)); got != 0 {
		t.Fatalf("counter reset should clamp to 0, got %v", got)
	}
	if got := r.Sample(500, t0.Add(3*time.Second)); got != 0 {
		t.Fatalf("zero interval = %v", got)
	}
}

func TestPhaseOf(t *testing.T) {
	cases := []struct {
		info  bool
		peers int
		want  Phase
	}{
		{false, 0, PhaseConnecting}, {false, 3, PhaseMetadata}, {true, 0, PhaseDownloading}, {true, 5, PhaseDownloading},
	}
	for _, c := range cases {
		if got := phaseOf(c.info, c.peers); got != c.want {
			t.Errorf("phaseOf(%v,%d) = %s, want %s", c.info, c.peers, got, c.want)
		}
	}
}

func TestBufferedAhead(t *testing.T) {
	// Pieces of 100 bytes; the file starts at 50 and is 400 bytes long, so it
	// spans pieces 0..4 (ends at 450).
	have := map[int]bool{0: true, 1: true, 2: true, 4: true}
	complete := func(i int) bool { return have[i] }
	cases := []struct {
		pos  int64
		want int64
	}{
		{0, 250},   // 50..300 complete, piece 3 missing
		{120, 130}, // 170..300
		{250, 0},   // piece 3 missing at 300
		{350, 50},  // 400..450 is piece 4, clipped at the file end
		{9999, 0},  // past the end
		{-5, 250},  // clamped
	}
	for _, c := range cases {
		if got := bufferedAhead(100, 50, 400, c.pos, 5, complete); got != c.want {
			t.Errorf("pos %d: got %d, want %d", c.pos, got, c.want)
		}
	}
	if got := bufferedAhead(0, 0, 10, 0, 1, complete); got != 0 {
		t.Errorf("zero piece length = %d", got)
	}
}

func TestBufferPercent(t *testing.T) {
	if got := bufferPercent(bufferTarget/2, 1<<30); got != 50 {
		t.Errorf("half = %d", got)
	}
	if got := bufferPercent(1<<30, 1<<30); got != 100 {
		t.Errorf("cap = %d", got)
	}
	if got := bufferPercent(500, 1000); got != 50 {
		t.Errorf("small file = %d", got)
	}
	if got := bufferPercent(5, 0); got != 0 {
		t.Errorf("empty = %d", got)
	}
}

func TestWatchdog(t *testing.T) {
	w := Watchdog{Timeout: 30 * time.Second}
	now := time.Unix(0, 0)
	if w.Observe(now, false, 0) {
		t.Fatal("fires immediately")
	}
	if w.Observe(now.Add(29*time.Second), false, 0) {
		t.Fatal("fires before the timeout")
	}
	if !w.Observe(now.Add(30*time.Second), false, 0) {
		t.Fatal("does not fire at the timeout")
	}
	// Peers without metadata still count as stuck.
	if !w.Observe(now.Add(40*time.Second), false, 4) {
		t.Fatal("peers without metadata should stay stuck")
	}
	// Keep waiting restarts the clock.
	w.Reset(now.Add(41 * time.Second))
	if w.Observe(now.Add(60*time.Second), false, 0) {
		t.Fatal("fires right after a reset")
	}
	if !w.Observe(now.Add(71*time.Second), false, 0) {
		t.Fatal("does not fire a timeout after the reset")
	}
	// Recovering clears the state.
	if w.Observe(now.Add(72*time.Second), true, 2) {
		t.Fatal("healthy swarm reported as stuck")
	}
	if w.Observe(now.Add(100*time.Second), true, 0) {
		t.Fatal("clock should restart after recovery")
	}
	if !w.Observe(now.Add(130*time.Second), true, 0) {
		t.Fatal("lost peers for a timeout should fire")
	}
}
