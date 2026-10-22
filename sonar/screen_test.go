package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Hustenreizjuengling/rcplug-plugins/sonar/internal/dsp"
)

// TestScreenPage: the page for the device's screen lists Sonar's state and
// settings in the asked language, and a change from the screen applies.
func TestScreenPage(t *testing.T) {
	h := newHub(dsp.DefaultConfig(48000), 48000, 480, "test", "")
	rec := httptest.NewRecorder()
	h.handleScreen(rec, httptest.NewRequest(http.MethodGet, "/screen?lang=de", nil))
	if rec.Code != 200 {
		t.Fatalf("GET: %d %s", rec.Code, rec.Body)
	}
	var page struct {
		Title    string          `json:"title"`
		Controls []screenControl `json:"controls"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	ids := map[string]screenControl{}
	for _, c := range page.Controls {
		ids[c.ID] = c
	}
	if page.Title != "Sonar" || ids["listen"].Type != "choice" || ids["flip"].Label != "Links und rechts tauschen" {
		t.Fatalf("page: %+v", page)
	}
	if ids["source"].Text != "Testsignal" {
		t.Fatalf("source: %q", ids["source"].Text)
	}
	// The radar: a canvas with the grid, the beam and the blip, in the
	// screen's units; the page asks to be fetched ten times a second.
	if c := ids["radar"]; c.Type != "canvas" || c.W != 100 || c.H != 100 || len(c.Shapes) < 20 {
		t.Fatalf("radar: %+v", ids["radar"])
	}
	var pageMeta struct {
		Every int    `json:"every"`
		Icon  string `json:"icon"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &pageMeta)
	if pageMeta.Every != screenEvery || pageMeta.Icon == "" {
		t.Fatalf("page meta: %+v", pageMeta)
	}
	byID := map[string]screenShape{}
	for _, sh := range ids["radar"].Shapes {
		if sh.ID != "" {
			byID[sh.ID] = sh
		}
	}
	for _, id := range []string{"beam", "glow", "blip", "l", "r"} {
		if _, ok := byID[id]; !ok {
			t.Errorf("radar without the %q shape", id)
		}
	}
	// A toggle from the screen.
	rec = httptest.NewRecorder()
	h.handleScreen(rec, httptest.NewRequest(http.MethodPost, "/screen", strings.NewReader(`{"control":"flip","value":true}`)))
	if rec.Code != 200 || !h.cfg.FlipSign {
		t.Fatalf("flip: %d %s, flip=%v", rec.Code, rec.Body, h.cfg.FlipSign)
	}
	rec = httptest.NewRecorder()
	h.handleScreen(rec, httptest.NewRequest(http.MethodPost, "/screen", strings.NewReader(`{"control":"smoothing","value":0.5}`)))
	if rec.Code != 200 || h.cfg.Smoothing != 0.5 {
		t.Fatalf("smoothing: %d %s, %v", rec.Code, rec.Body, h.cfg.Smoothing)
	}
	rec = httptest.NewRecorder()
	h.handleScreen(rec, httptest.NewRequest(http.MethodPost, "/screen", strings.NewReader(`{"control":"nope","value":1}`)))
	if rec.Code != 400 {
		t.Fatalf("an unknown control: %d", rec.Code)
	}
}

func TestSourceNames(t *testing.T) {
	en := screenTexts["en"]
	cases := map[string]string{
		"test": "Test signal", "jack:system:capture_1,system:capture_2": "Mic 1 + Mic 2",
		"jack:sip-audio-in:capture_3,sip-audio-in:capture_4": "Call 2",
		"jack:JUCEJack:out_3,JUCEJack:out_4":                 "Mixer output 3 + 4",
		"jack:foo:a,foo:b":                                   "foo:a + foo:b", "alsa:hw:1,0": "ALSA hw:1,0",
	}
	for spec, want := range cases {
		if got := sourceName(en, spec); got != want {
			t.Errorf("%s: %q, want %q", spec, got, want)
		}
	}
}

// TestTrail: the trail keeps the last two seconds of sound, thinned to one
// point per 60 ms and at most trailMax points, and quiet readings add
// nothing.
func TestTrail(t *testing.T) {
	h := newHub(dsp.DefaultConfig(48000), 48000, 480, "test", "")
	now := time.Now()
	for i := 0; i < 200; i++ {
		h.noteTrail(dsp.Estimate{Level: 0.5, Azimuth: float64(i)}, now.Add(time.Duration(i)*20*time.Millisecond))
	}
	if n := len(h.trail); n == 0 || n > trailMax {
		t.Fatalf("%d points after 4 s of sound", n)
	}
	last := now.Add(199 * 20 * time.Millisecond)
	if p := h.trail[len(h.trail)-1]; last.Sub(p.at) >= trailStep {
		t.Errorf("the newest point is %v old, want one within %v", last.Sub(p.at), trailStep)
	}
	if first := h.trail[0]; last.Sub(first.at) > trailFor+trailStep {
		t.Errorf("a point older than %v kept: %v", trailFor, last.Sub(first.at))
	}
	for i := 1; i < len(h.trail); i++ {
		if d := h.trail[i].at.Sub(h.trail[i-1].at); d < trailStep {
			t.Fatalf("points %v apart, want %v or more", d, trailStep)
		}
	}
	before := len(h.trail)
	h.noteTrail(dsp.Estimate{Level: 0.01}, last)
	if len(h.trail) != before {
		t.Errorf("a quiet reading changed the trail (%d → %d)", before, len(h.trail))
	}
	// The picture carries the trail as faint dots on top of the grid.
	h.mu.Lock()
	n := len(radar(screenTexts["en"], dsp.Estimate{Level: 0.5, LevelDb: -20}, h.cfg, h.trail, last))
	h.mu.Unlock()
	if n < 20+len(h.trail)/2 {
		t.Errorf("radar with %d shapes for %d trail points", n, len(h.trail))
	}
}

// TestScreenStream: asked for an event stream, the page comes as one: a
// picture at once and one with every publish, each the whole page.
func TestScreenStream(t *testing.T) {
	h := newHub(dsp.DefaultConfig(48000), 48000, 480, "test", "")
	srv := httptest.NewServer(http.HandlerFunc(h.handleScreen))
	defer srv.Close()
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/screen?lang=en", nil)
	req.Header.Set("Accept", "text/event-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("content type %q", ct)
	}
	events := make(chan []byte, 8)
	go func() {
		sc := bufio.NewScanner(resp.Body)
		sc.Buffer(make([]byte, 64<<10), 1<<20)
		for sc.Scan() {
			if line := sc.Bytes(); bytes.HasPrefix(line, []byte("data: ")) {
				events <- append([]byte(nil), line[6:]...)
			}
		}
		close(events)
	}()
	read := func() []byte {
		select {
		case ev := <-events:
			return ev
		case <-time.After(2 * time.Second):
			t.Fatal("no picture arrived")
		}
		return nil
	}
	first := read()
	for i := 0; i < 2; i++ {
		h.publish(dsp.Estimate{Azimuth: 30, Level: 0.5, Conf: 0.9, Dist: 0.5})
		time.Sleep(10 * time.Millisecond)
	}
	second := read()
	for _, ev := range [][]byte{first, second} {
		var page struct {
			Title    string          `json:"title"`
			Every    int             `json:"every"`
			Controls []screenControl `json:"controls"`
		}
		if err := json.Unmarshal(ev, &page); err != nil {
			t.Fatalf("event %s: %v", ev, err)
		}
		if page.Title != "Sonar" || page.Every != screenEvery || len(page.Controls) < 7 || page.Controls[0].Type != "canvas" {
			t.Fatalf("event is not the page: %+v", page)
		}
	}
	// A publish while the screen is still busy is not queued up: at most
	// one picture waits.
	h.mu.Lock()
	n := len(h.screens)
	h.mu.Unlock()
	if n != 1 {
		t.Fatalf("screens: %d", n)
	}
}
