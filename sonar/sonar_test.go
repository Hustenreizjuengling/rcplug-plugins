package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Hustenreizjuengling/rcplug-plugins/sonar/internal/dsp"
)

func TestSyntheticSourceIsPacedToRealTime(t *testing.T) {
	src, err := openSource(context.Background(), "test", 48000)
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 4800*4) // 100 ms of audio
	start := time.Now()
	for i := 0; i < 3; i++ {
		if _, err := io.ReadFull(src, buf); err != nil {
			t.Fatal(err)
		}
	}
	if d := time.Since(start); d < 250*time.Millisecond {
		t.Fatalf("300 ms of audio delivered in %s: not paced", d)
	}
}

func TestValidateSource(t *testing.T) {
	data := t.TempDir()
	for spec, ok := range map[string]bool{
		"test": true, "alsa:default": true, "alsa:hw:0,0": true, "alsa:plughw:CARD=Duo,DEV=0": true,
		"wav:clip.wav": true, "wav:" + filepath.Join(data, "x.wav"): true,
		"alsa:default; rm -rf /": false, "wav:/etc/shadow": false, "wav:../../etc/passwd": false,
		"stdin": false, "http://x": false,
	} {
		if _, err := validateSource(spec, data); (err == nil) != ok {
			t.Errorf("%q: err=%v, want ok=%v", spec, err, ok)
		}
	}
	// A relative WAV path is opened inside the data directory.
	if spec, _ := validateSource("wav:clip.wav", data); spec != "wav:"+filepath.Join(data, "clip.wav") {
		t.Errorf("relative WAV resolved to %q", spec)
	}
	// A sibling directory sharing the prefix is outside.
	if _, err := validateSource("wav:"+data+"-other/x.wav", data); err == nil {
		t.Error("sibling directory accepted")
	}
}

// A broken source must back off instead of spinning.
func TestFailingSourceBacksOff(t *testing.T) {
	h := newHub(dsp.DefaultConfig(48000), 48000, 1024, "wav:/nonexistent.wav", "")
	var publishes atomic.Int32
	ch := make(chan []byte, 64)
	h.subs[ch] = struct{}{}
	go func() {
		for msg := range ch {
			if strings.Contains(string(msg), "nonexistent") {
				publishes.Add(1)
			}
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
	defer cancel()
	h.run(ctx)
	if n := publishes.Load(); n < 1 || n > 3 {
		t.Fatalf("%d error reports in 2.5 s (want a few, with backoff)", n)
	}
}

// Choosing another source restarts the capture immediately.
func TestSourceChangeRestartsCapture(t *testing.T) {
	h := newHub(dsp.DefaultConfig(48000), 48000, 1024, "wav:/nonexistent.wav", "")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.run(ctx)
	time.Sleep(200 * time.Millisecond)
	h.mu.Lock()
	h.srcSpec = "test"
	h.mu.Unlock()
	h.restart <- struct{}{}
	deadline := time.Now().Add(2 * time.Second)
	for {
		h.mu.Lock()
		name, serr := h.srcName, h.srcErr
		h.mu.Unlock()
		if strings.HasPrefix(name, "test") && serr == "" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("still on %q (%q)", name, serr)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// Choosing a console channel switches to the audio source assigned to it,
// and the assignment survives a restart.
func TestChannelAssignment(t *testing.T) {
	state := filepath.Join(t.TempDir(), "sonar.state.json")
	h := newHub(dsp.DefaultConfig(48000), 48000, 1024, "test", state)
	post := func(body string) (int, map[string]interface{}) {
		rec := httptest.NewRecorder()
		h.handleConfig(rec, httptest.NewRequest(http.MethodPost, "/api/config", strings.NewReader(body)))
		var out map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return rec.Code, out
	}
	if code, _ := post(`{"assign":{"channel":"combo1","source":"alsa:hw:1,0"}}`); code != 200 {
		t.Fatalf("assign: %d", code)
	}
	if _, out := post(`{"channel":"combo1"}`); out["source"] != "alsa:hw:1,0" || out["channel"] != "combo1" {
		t.Fatalf("channel did not switch the source: %v", out)
	}
	if _, out := post(`{"channel":"usb2"}`); out["source"] != "alsa:hw:1,0" {
		t.Fatalf("an unassigned channel must keep the current source: %v", out)
	}
	for _, bad := range []string{`{"channel":"../x"}`, `{"assign":{"channel":"combo1","source":"wav:/etc/shadow"}}`} {
		if code, _ := post(bad); code != http.StatusBadRequest {
			t.Errorf("%s: %d, want 400", bad, code)
		}
	}
	h2 := newHub(dsp.DefaultConfig(48000), 48000, 1024, "test", state)
	h2.loadState()
	if h2.channel != "usb2" || h2.channels["combo1"] != "alsa:hw:1,0" {
		t.Fatalf("not persisted: channel=%q channels=%v", h2.channel, h2.channels)
	}
}
