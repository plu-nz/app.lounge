// Package player drives libmpv and draws its video under the app's webview.
package player

/*
#cgo pkg-config: gtk4 mpv epoxy
#include <stdlib.h>
#include <mpv/client.h>
#include "render.h"

static char **lounge_args(int n) { return calloc(n + 1, sizeof(char *)); }
static void lounge_args_set(char **a, int i, char *s) { a[i] = s; }
static mpv_event_property *lounge_event_property(mpv_event *e) { return (mpv_event_property *)e->data; }
static mpv_event_end_file *lounge_event_end_file(mpv_event *e) { return (mpv_event_end_file *)e->data; }
static mpv_event_log_message *lounge_event_log(mpv_event *e) { return (mpv_event_log_message *)e->data; }
*/
import "C"

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
	"unsafe"
)

// Event is a player state change sent to the UI.
type Event struct {
	Kind  string          `json:"kind"`            // prop, loaded, end, idle
	Name  string          `json:"name,omitempty"`  // property name for prop
	Value json.RawMessage `json:"value,omitempty"` // property value or end reason
}

// observed lists the properties the UI follows.
var observed = []string{
	"pause", "time-pos", "duration", "paused-for-cache", "cache-buffering-state",
	"demuxer-cache-time", "track-list", "aid", "sid", "volume", "mute", "speed",
	"sub-delay", "sub-scale", "sub-pos", "idle-active", "eof-reached", "media-title",
	"video-params/w", "video-params/h", "seeking", "hwdec-current",
}

// Player owns one mpv core.
type Player struct {
	handle *C.mpv_handle
	emit   func(Event)
	once   sync.Once
}

// New creates an mpv core configured to render through the GL area.
func New(emit func(Event)) (*Player, error) {
	h := C.mpv_create()
	if h == nil {
		return nil, errors.New("mpv_create failed")
	}
	p := &Player{handle: h, emit: emit}
	options := [][2]string{
		{"vo", "libmpv"},
		// VAAPI first: it is what AMD and Intel GPUs offer on Linux, and
		// auto-safe would otherwise try Vulkan and CUDA, which this OpenGL
		// render path cannot use.
		{"hwdec", "vaapi,auto-safe"},
		// GTK draws frames on its own clock, so mpv must not block the main
		// loop waiting for a frame's target time.
		{"video-timing-offset", "0"},
		{"idle", "yes"},
		{"keep-open", "no"},
		{"terminal", "no"},
		{"input-default-bindings", "no"},
		{"input-vo-keyboard", "no"},
		{"osc", "no"},
		{"ytdl", "no"},
		{"sub-auto", "fuzzy"},
		{"cache", "yes"},
		{"demuxer-max-bytes", "150MiB"},
		{"network-timeout", "30"},
		{"user-agent", "Lounge/0.1"},
	}
	for _, o := range options {
		if err := p.setOption(o[0], o[1]); err != nil {
			C.mpv_terminate_destroy(h)
			return nil, err
		}
	}
	if err := check(C.mpv_initialize(h)); err != nil {
		C.mpv_terminate_destroy(h)
		return nil, fmt.Errorf("mpv_initialize: %w", err)
	}
	level := C.CString("warn")
	C.mpv_request_log_messages(h, level)
	C.free(unsafe.Pointer(level))
	for i, name := range observed {
		cname := C.CString(name)
		C.mpv_observe_property(h, C.uint64_t(i+1), cname, C.MPV_FORMAT_NODE)
		C.free(unsafe.Pointer(cname))
	}
	go p.loop()
	if os.Getenv("LOUNGE_DEBUG") != "" {
		go p.logStats()
	}
	return p, nil
}

// logStats prints playback health every few seconds (LOUNGE_DEBUG=1).
func (p *Player) logStats() {
	for range time.Tick(5 * time.Second) {
		var parts []string
		for _, name := range []string{"hwdec-current", "estimated-vf-fps", "frame-drop-count", "decoder-frame-drop-count", "vo-delayed-frame-count", "avsync", "vsync-jitter", "video-params/w"} {
			v, err := p.Property(name)
			if err != nil {
				continue
			}
			parts = append(parts, name+"="+string(v))
		}
		if len(parts) > 0 {
			log.Printf("mpv stats: %s", strings.Join(parts, " "))
		}
	}
}

// Attach places the video under the webview of the given GtkWindow.
// It must run on the GTK main thread.
func (p *Player) Attach(gtkWindow unsafe.Pointer) error {
	if rc := C.lounge_attach(gtkWindow, p.handle); rc != 0 {
		return fmt.Errorf("attach video surface: code %d", int(rc))
	}
	return nil
}

// Detach frees the render context. It must run on the GTK main thread.
func (p *Player) Detach() { C.lounge_detach() }

// Command runs an mpv command such as {"loadfile", url}.
func (p *Player) Command(args ...string) error {
	if len(args) == 0 {
		return errors.New("empty command")
	}
	cargs := C.lounge_args(C.int(len(args)))
	defer C.free(unsafe.Pointer(cargs))
	for i, a := range args {
		s := C.CString(a)
		defer C.free(unsafe.Pointer(s))
		C.lounge_args_set(cargs, C.int(i), s)
	}
	if err := check(C.mpv_command(p.handle, cargs)); err != nil {
		return fmt.Errorf("%s: %w", args[0], err)
	}
	return nil
}

// SetProperty sets a property from its string form.
func (p *Player) SetProperty(name, value string) error {
	cname, cvalue := C.CString(name), C.CString(value)
	defer C.free(unsafe.Pointer(cname))
	defer C.free(unsafe.Pointer(cvalue))
	if err := check(C.mpv_set_property_string(p.handle, cname, cvalue)); err != nil {
		return fmt.Errorf("set %s: %w", name, err)
	}
	return nil
}

// Property returns a property value as JSON.
func (p *Player) Property(name string) (json.RawMessage, error) {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	var node C.mpv_node
	if err := check(C.mpv_get_property(p.handle, cname, C.MPV_FORMAT_NODE, unsafe.Pointer(&node))); err != nil {
		return nil, fmt.Errorf("get %s: %w", name, err)
	}
	defer C.mpv_free_node_contents(&node)
	return nodeJSON(&node), nil
}

// Close stops playback and destroys the core. Call Detach first.
//
// Teardown runs off the calling (GTK main) thread, which mpv may still
// need while it shuts down, and gives up after a short wait because the
// process is exiting anyway.
func (p *Player) Close() {
	p.once.Do(func() {
		done := make(chan struct{})
		go func() {
			C.mpv_terminate_destroy(p.handle)
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			log.Print("mpv did not shut down in time")
		}
	})
}

func (p *Player) setOption(name, value string) error {
	cname, cvalue := C.CString(name), C.CString(value)
	defer C.free(unsafe.Pointer(cname))
	defer C.free(unsafe.Pointer(cvalue))
	if err := check(C.mpv_set_option_string(p.handle, cname, cvalue)); err != nil {
		return fmt.Errorf("option %s=%s: %w", name, value, err)
	}
	return nil
}

func (p *Player) loop() {
	for {
		ev := C.mpv_wait_event(p.handle, -1)
		switch ev.event_id {
		case C.MPV_EVENT_SHUTDOWN:
			return
		case C.MPV_EVENT_PROPERTY_CHANGE:
			prop := C.lounge_event_property(ev)
			value := json.RawMessage("null")
			if prop.format == C.MPV_FORMAT_NODE && prop.data != nil {
				value = nodeJSON((*C.mpv_node)(prop.data))
			}
			p.emit(Event{Kind: "prop", Name: C.GoString(prop.name), Value: value})
		case C.MPV_EVENT_FILE_LOADED:
			p.emit(Event{Kind: "loaded"})
		case C.MPV_EVENT_END_FILE:
			end := C.lounge_event_end_file(ev)
			reason := map[C.mpv_end_file_reason]string{
				C.MPV_END_FILE_REASON_EOF:      "eof",
				C.MPV_END_FILE_REASON_STOP:     "stop",
				C.MPV_END_FILE_REASON_QUIT:     "quit",
				C.MPV_END_FILE_REASON_ERROR:    "error",
				C.MPV_END_FILE_REASON_REDIRECT: "redirect",
			}[end.reason]
			detail := ""
			if end.reason == C.MPV_END_FILE_REASON_ERROR {
				detail = C.GoString(C.mpv_error_string(end.error))
			}
			value, _ := json.Marshal(map[string]string{"reason": reason, "error": detail})
			p.emit(Event{Kind: "end", Value: value})
		case C.MPV_EVENT_LOG_MESSAGE:
			msg := C.lounge_event_log(ev)
			log.Printf("mpv [%s] %s", C.GoString(msg.prefix), strings.TrimSpace(C.GoString(msg.text)))
		}
	}
}

func nodeJSON(node *C.mpv_node) json.RawMessage {
	s := C.lounge_node_json(node)
	defer C.free(unsafe.Pointer(s))
	return json.RawMessage(C.GoString(s))
}

func check(rc C.int) error {
	if rc < 0 {
		return errors.New(C.GoString(C.mpv_error_string(rc)))
	}
	return nil
}
