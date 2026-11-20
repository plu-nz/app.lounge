package player

import (
	"sync"
	"time"
)

// throttle limits how often fast-changing properties reach the UI. mpv
// reports time-pos on every video frame; redrawing the controls that often
// costs frames at 4K and makes the overlay flicker. The latest value is
// always delivered, at most one interval late.
type throttle struct {
	every time.Duration
	emit  func(Event)
	names map[string]bool

	mu      sync.Mutex
	last    map[string]time.Time
	pending map[string]Event
	timer   *time.Timer
}

func newThrottle(every time.Duration, emit func(Event), names ...string) *throttle {
	t := &throttle{
		every:   every,
		emit:    emit,
		names:   map[string]bool{},
		last:    map[string]time.Time{},
		pending: map[string]Event{},
	}
	for _, n := range names {
		t.names[n] = true
	}
	return t
}

// send emits ev now, or later when its property was sent too recently.
// Any other event first flushes pending values, so the UI sees events in
// the order mpv produced them.
func (t *throttle) send(ev Event) {
	if ev.Kind != "prop" || !t.names[ev.Name] {
		t.flush()
		t.emit(ev)
		return
	}
	t.mu.Lock()
	now := time.Now()
	if now.Sub(t.last[ev.Name]) >= t.every {
		delete(t.pending, ev.Name)
		t.last[ev.Name] = now
		t.mu.Unlock()
		t.emit(ev)
		return
	}
	t.pending[ev.Name] = ev
	if t.timer == nil {
		t.timer = time.AfterFunc(t.every-now.Sub(t.last[ev.Name]), t.flush)
	}
	t.mu.Unlock()
}

// flush emits every pending value now.
func (t *throttle) flush() {
	t.mu.Lock()
	if t.timer != nil {
		t.timer.Stop()
		t.timer = nil
	}
	events := make([]Event, 0, len(t.pending))
	now := time.Now()
	for name, ev := range t.pending {
		events = append(events, ev)
		t.last[name] = now
	}
	clear(t.pending)
	t.mu.Unlock()
	for _, ev := range events {
		t.emit(ev)
	}
}
