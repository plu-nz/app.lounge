package player

import (
	"encoding/json"
	"sync"
	"testing"
	"time"
)

type recorder struct {
	mu     sync.Mutex
	events []Event
}

func (r *recorder) emit(ev Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, ev)
}

func (r *recorder) values() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.events))
	for i, ev := range r.events {
		out[i] = ev.Name + "=" + string(ev.Value)
	}
	return out
}

func prop(name, value string) Event {
	return Event{Kind: "prop", Name: name, Value: json.RawMessage(value)}
}

func TestThrottleKeepsTheLatestValue(t *testing.T) {
	r := &recorder{}
	th := newThrottle(50*time.Millisecond, r.emit, "time-pos")
	for _, v := range []string{"1", "2", "3", "4"} {
		th.send(prop("time-pos", v))
	}
	if got := r.values(); len(got) != 1 || got[0] != "time-pos=1" {
		t.Fatalf("first burst = %v, want only time-pos=1", got)
	}
	time.Sleep(120 * time.Millisecond)
	if got := r.values(); len(got) != 2 || got[1] != "time-pos=4" {
		t.Fatalf("after the interval = %v, want the latest value delivered", got)
	}
}

func TestThrottleFlushesBeforeOtherEvents(t *testing.T) {
	r := &recorder{}
	th := newThrottle(time.Hour, r.emit, "time-pos")
	th.send(prop("time-pos", "1"))
	th.send(prop("time-pos", "2"))
	th.send(prop("pause", "true"))
	th.send(Event{Kind: "end"})
	want := []string{"time-pos=1", "time-pos=2", "pause=true", "="}
	got := r.values()
	if len(got) != len(want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("events = %v, want %v", got, want)
		}
	}
}
