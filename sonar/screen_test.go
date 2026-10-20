package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
