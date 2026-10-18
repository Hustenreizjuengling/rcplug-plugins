package main

import (
	"os"

	"github.com/Hustenreizjuengling/rcplug/sdk"
)

// savedState is the sonar's persisted configuration so a user's channel and
// tuning survive plugin restarts and reboots.
type savedState struct {
	Source     string  `json:"source"`
	Smoothing  float64 `json:"smoothing"`
	Flip       bool    `json:"flip"`
	NoiseFloor float64 `json:"noiseFloor"`
	// Channel is the console source the sonar listens to (e.g. "combo1"),
	// Channels maps console sources to audio sources.
	Channel  string            `json:"channel,omitempty"`
	Channels map[string]string `json:"channels,omitempty"`
	// The distance's calibration (dsp.Config).
	NearDb  float64 `json:"nearDb,omitempty"`
	NearCoh float64 `json:"nearCoh,omitempty"`
	FarDb   float64 `json:"farDb,omitempty"`
	FarCoh  float64 `json:"farCoh,omitempty"`
}

// statePath returns where the sonar stores its settings, in the plugin's data
// directory (RCPLUG_DATA_DIR).
func statePath() string { return sdk.StatePath("sonar.state.json") }

func (h *hub) loadState() {
	if h.statePath == "" {
		return
	}
	if _, err := os.Stat(h.statePath); err != nil {
		return // first run: keep flag defaults
	}
	var s savedState
	if sdk.LoadJSON(h.statePath, &s) != nil {
		return
	}
	h.mu.Lock()
	if s.Source != "" {
		if spec, err := validateSource(s.Source, sdk.DataDir()); err == nil {
			h.srcSpec = spec
		}
	}
	h.cfg.Smoothing = clamp(s.Smoothing, 0, 0.99)
	h.cfg.FlipSign = s.Flip
	if s.NoiseFloor != 0 {
		h.cfg.NoiseFloor = s.NoiseFloor
	}
	if s.NearDb < 0 && s.NearDb >= -120 {
		h.cfg.NearDb, h.cfg.NearCoh = s.NearDb, clamp(s.NearCoh, 0, 1)
	}
	if s.FarDb < 0 && s.FarDb >= -120 {
		h.cfg.FarDb, h.cfg.FarCoh = s.FarDb, clamp(s.FarCoh, 0, 1)
	}
	if validChannel(s.Channel) {
		h.channel = s.Channel
	}
	for ch, src := range s.Channels {
		if spec, err := validateSource(src, sdk.DataDir()); err == nil && validChannel(ch) {
			h.channels[ch] = spec
		}
	}
	h.mu.Unlock()
}

func (h *hub) saveState() {
	if h.statePath == "" {
		return
	}
	h.mu.Lock()
	s := savedState{
		Source:     h.srcSpec,
		Smoothing:  h.cfg.Smoothing,
		Flip:       h.cfg.FlipSign,
		NoiseFloor: h.cfg.NoiseFloor,
		Channel:    h.channel,
		Channels:   map[string]string{},
		NearDb:     h.cfg.NearDb, NearCoh: h.cfg.NearCoh, FarDb: h.cfg.FarDb, FarCoh: h.cfg.FarCoh,
	}
	for k, v := range h.channels {
		s.Channels[k] = v
	}
	h.mu.Unlock()
	_ = sdk.SaveJSON(h.statePath, s)
}
