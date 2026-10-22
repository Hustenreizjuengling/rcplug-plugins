package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/Hustenreizjuengling/rcplug-plugins/sonar/internal/dsp"
	"github.com/Hustenreizjuengling/rcplug/sdk"
)

// Sonar's page on the device's screen (the `screen` plugin): a description
// of controls the screen draws itself, served at /screen, and the changes
// it sends back. Sonar never draws pixels on the screen; it describes the
// radar as shapes in a canvas, says what it hears and what can be set, in
// the screen's language (?lang=de or en).

// screenControl is one control of the page, as the screen plugin reads it.
type screenControl struct {
	ID      string   `json:"id"`
	Type    string   `json:"type"` // canvas value meter choice slider toggle button
	Label   string   `json:"label"`
	Text    string   `json:"text,omitempty"`
	Value   float64  `json:"value,omitempty"`
	On      bool     `json:"on,omitempty"`
	Min     float64  `json:"min,omitempty"`
	Max     float64  `json:"max,omitempty"`
	Unit    string   `json:"unit,omitempty"`
	Options []string `json:"options,omitempty"`
	Danger  bool     `json:"danger,omitempty"`
	Confirm string   `json:"confirm,omitempty"`

	// A canvas: the radar as shapes in a 100 × 100 picture the screen
	// scales and draws itself (screen 0.5.0 or later; an older screen
	// skips the control it does not know).
	W      float64       `json:"w,omitempty"`
	H      float64       `json:"h,omitempty"`
	Shapes []screenShape `json:"shapes,omitempty"`
}

// screenShape is one shape of the radar. Shapes with an id (the beam, the
// blip, the level bars) glide between two descriptions on the screen.
type screenShape struct {
	ID     string       `json:"id,omitempty"`
	Shape  string       `json:"shape"`
	X      float64      `json:"x,omitempty"`
	Y      float64      `json:"y,omitempty"`
	R      float64      `json:"r,omitempty"`
	W      float64      `json:"w,omitempty"`
	H      float64      `json:"h,omitempty"`
	Radius float64      `json:"radius,omitempty"`
	A0     float64      `json:"a0,omitempty"`
	A1     float64      `json:"a1,omitempty"`
	Width  float64      `json:"width,omitempty"`
	Points [][2]float64 `json:"points,omitempty"`
	Text   string       `json:"text,omitempty"`
	Size   string       `json:"size,omitempty"`
	Align  string       `json:"align,omitempty"`
	Color  string       `json:"color,omitempty"`
	Alpha  float64      `json:"alpha,omitempty"`
}

// screenEvery is how often a screen that polls asks for the page while it
// shows it (ms): ten pictures a second, which the screen smooths between.
// A screen since 0.6.0 asks for a stream instead and gets a picture with
// every estimate (about thirty a second).
const screenEvery = 100

// trailPt is one recent blip for the radar's trail.
type trailPt struct {
	at              time.Time
	az, level, dist float64
}

// The trail keeps the last trailFor of sound, one point per trailStep at
// most.
const (
	trailFor  = 2 * time.Second
	trailStep = 60 * time.Millisecond
	trailMax  = 40
)

// noteTrail records a reading for the trail (under h.mu).
func (h *hub) noteTrail(e dsp.Estimate, now time.Time) {
	if e.Level > 0.04 && (len(h.trail) == 0 || now.Sub(h.trail[len(h.trail)-1].at) >= trailStep) {
		h.trail = append(h.trail, trailPt{at: now, az: e.Azimuth, level: e.Level, dist: e.Dist})
	}
	drop := 0
	for drop < len(h.trail) && now.Sub(h.trail[drop].at) > trailFor {
		drop++
	}
	if len(h.trail)-drop > trailMax {
		drop = len(h.trail) - trailMax
	}
	if drop > 0 {
		h.trail = append(h.trail[:0], h.trail[drop:]...)
	}
}

// radar is the picture: a half-disc with rings and spokes (0° ahead, −90°
// hard left, +90° hard right; near the centre is near the microphones),
// the trail of the last two seconds, the beam and the blip, then the
// readouts and the left and right level under it.
func radar(t map[string]string, e dsp.Estimate, cfg dsp.Config, trail []trailPt, now time.Time) []screenShape {
	const cx, cy, R = 50.0, 56.0, 44.0
	ang := func(az float64) float64 { return (270 + math.Max(-90, math.Min(90, az))) * math.Pi / 180 }
	rad := func(dist float64) float64 { return R * (0.12 + 0.83*math.Max(0, math.Min(1, dist))) }
	var sh []screenShape
	for k := 1; k <= 4; k++ {
		sh = append(sh, screenShape{Shape: "arc", X: cx, Y: cy, R: R * float64(k) / 4, A0: 180, A1: 360, Color: "line", Alpha: 0.6})
	}
	for _, a := range []float64{-90, -45, 0, 45, 90} {
		th := ang(a)
		sh = append(sh, screenShape{Shape: "line", Points: [][2]float64{{cx, cy}, {cx + math.Cos(th)*R, cy + math.Sin(th)*R}}, Color: "line", Alpha: 0.6})
		lx, ly := cx+math.Cos(th)*(R+5), cy+math.Sin(th)*(R+4)
		switch a {
		case -90:
			lx += 4
		case 90:
			lx -= 4
		}
		sh = append(sh, screenShape{Shape: "text", X: lx, Y: ly - 2, Text: fmt.Sprintf("%+d°", int(a)), Size: "tiny", Color: "dim"})
	}
	sh = append(sh, screenShape{Shape: "text", X: cx + R*0.14, Y: cy - 3, Text: t["nearV"], Size: "tiny", Color: "dim", Align: "left"},
		screenShape{Shape: "text", X: cx + R - 8, Y: cy - 3, Text: t["farV"], Size: "tiny", Color: "dim", Align: "right"})
	for _, p := range trail {
		age := float64(now.Sub(p.at)) / float64(trailFor)
		if a := (1 - age) * 0.5 * p.level; a > 0.03 {
			th, rr := ang(p.az), rad(p.dist)
			sh = append(sh, screenShape{Shape: "circle", X: cx + math.Cos(th)*rr, Y: cy + math.Sin(th)*rr, R: 0.9, Color: "accent", Alpha: a})
		}
	}
	level := math.Max(0, math.Min(1, e.Level))
	th, rr := ang(e.Azimuth), rad(e.Dist)
	bx, by := cx+math.Cos(th)*rr, cy+math.Sin(th)*rr
	sh = append(sh,
		screenShape{ID: "beam", Shape: "line", Points: [][2]float64{{cx, cy}, {cx + math.Cos(th)*R, cy + math.Sin(th)*R}}, Width: 0.5, Color: "accent", Alpha: 0.3 + 0.5*e.Conf},
		screenShape{ID: "glow", Shape: "glow", X: bx, Y: by, R: (1 + level*2.8) * 2.2, Color: "accent", Alpha: 0.55 + 0.4*e.Conf},
		screenShape{ID: "blip", Shape: "circle", X: bx, Y: by, R: 0.9, Color: "text"},
	)
	// The readouts.
	quiet := e.LevelDb <= cfg.NoiseFloor
	bearing, distance, db := "—", "—", "-∞"
	if !quiet {
		bearing = fmt.Sprintf("%+d°", int(math.Round(e.Azimuth)))
		switch {
		case e.Dist < 0.33:
			distance = t["nearV"]
		case e.Dist < 0.66:
			distance = t["midV"]
		default:
			distance = t["farV"]
		}
	}
	if e.LevelDb > -99 {
		db = fmt.Sprintf("%d", int(math.Round(e.LevelDb)))
	}
	readouts := [][2]string{{bearing, t["bearing"]}, {distance, t["distance"]}, {db + " dBFS", t["level"]}, {fmt.Sprintf("%d %%", int(math.Round(e.Conf*100))), t["confidence"]}}
	for i, ro := range readouts {
		x := 12.5 + 25*float64(i)
		sh = append(sh, screenShape{Shape: "text", X: x, Y: 72, Text: ro[0], Size: "title"},
			screenShape{Shape: "text", X: x, Y: 80, Text: ro[1], Size: "tiny", Color: "dim"})
	}
	for i, lv := range []float64{e.L, e.R} {
		y := 89 + 5*float64(i)
		lv = math.Max(0, math.Min(1, lv))
		sh = append(sh, screenShape{Shape: "text", X: 4, Y: y, Text: []string{"L", "R"}[i], Size: "tiny", Color: "dim", Align: "left"},
			screenShape{Shape: "rect", X: 8, Y: y - 1, W: 88, H: 2, Radius: 1, Color: "panel"},
			screenShape{ID: []string{"l", "r"}[i], Shape: "rect", X: 8, Y: y - 1, W: math.Max(0.5, 88*lv), H: 2, Radius: 1, Color: "accent"})
	}
	return sh
}

// screenTexts are the page's words in English and German.
var screenTexts = map[string]map[string]string{
	"en": {
		"source": "Listening to", "bearing": "Direction", "level": "Level", "distance": "Distance", "confidence": "Confidence",
		"listen": "Listen to", "smoothing": "Smoothing", "flip": "Swap left and right",
		"near": "Calibrate: this is near", "far": "Calibrate: this is far", "reset": "Reset the calibration",
		"resetConfirm": "Forget the near and far points?", "left": "left", "right": "right", "centre": "centre",
		"quiet": "quiet", "nearV": "near", "midV": "middle", "farV": "far", "none": "—",
		"sMic12": "Mic 1 + Mic 2", "sMic23": "Mic 2 + Mic 3", "sIn34": "Inputs 3 + 4", "sUsb1": "USB 1 from the computer",
		"sUsb2": "USB 2 from the computer", "sChat": "Chat", "sCall": "Call %d", "sWireless": "RØDE wireless",
		"sToUsb1": "Mix to the computer (USB 1)", "sToChat": "Chat to the computer", "sToUsb2": "USB 2 to the computer",
		"sMixOut": "Mixer output %s + %s", "sTest": "Test signal",
	},
	"de": {
		"source": "Hört auf", "bearing": "Richtung", "level": "Pegel", "distance": "Entfernung", "confidence": "Sicherheit",
		"listen": "Hören auf", "smoothing": "Glättung", "flip": "Links und rechts tauschen",
		"near": "Kalibrieren: das ist nah", "far": "Kalibrieren: das ist fern", "reset": "Kalibrierung zurücksetzen",
		"resetConfirm": "Nah- und Fernpunkt vergessen?", "left": "links", "right": "rechts", "centre": "Mitte",
		"quiet": "still", "nearV": "nah", "midV": "mittel", "farV": "fern", "none": "—",
		"sMic12": "Mic 1 + Mic 2", "sMic23": "Mic 2 + Mic 3", "sIn34": "Eingänge 3 + 4", "sUsb1": "USB 1 vom Computer",
		"sUsb2": "USB 2 vom Computer", "sChat": "Chat", "sCall": "Anruf %d", "sWireless": "RØDE Wireless",
		"sToUsb1": "Mix zum Computer (USB 1)", "sToChat": "Chat zum Computer", "sToUsb2": "USB 2 zum Computer",
		"sMixOut": "Mixer-Ausgang %s + %s", "sTest": "Testsignal",
	},
}

// namedPairs are the desk's JACK pairs by name (a Duo with firmware 1.7.6).
var namedPairs = map[string]string{
	"system:capture_1,system:capture_2":             "sMic12",
	"system:capture_2,system:capture_3":             "sMic23",
	"system:capture_3,system:capture_4":             "sIn34",
	"usb1in:capture_1,usb1in:capture_2":             "sUsb1",
	"usb2in:capture_1,usb2in:capture_2":             "sUsb2",
	"usb1-chat-in:capture_1,usb1-chat-in:capture_2": "sChat",
	"sip-audio-in:capture_1,sip-audio-in:capture_2": "sCall1",
	"sip-audio-in:capture_3,sip-audio-in:capture_4": "sCall2",
	"sip-audio-in:capture_5,sip-audio-in:capture_6": "sCall3",
	"RodeRfSPI:TX1,RodeRfSPI:TX2":                   "sWireless",
	"JUCEJack:out_15,JUCEJack:out_16":               "sToUsb1",
	"JUCEJack:out_17,JUCEJack:out_18":               "sToChat",
	"JUCEJack:out_19,JUCEJack:out_20":               "sToUsb2",
}

func screenLang(r *http.Request) map[string]string {
	if strings.HasPrefix(strings.ToLower(r.URL.Query().Get("lang")), "de") {
		return screenTexts["de"]
	}
	return screenTexts["en"]
}

// sourceName names an audio source spec for the page.
func sourceName(t map[string]string, spec string) string {
	switch {
	case spec == "" || spec == "test":
		return t["sTest"]
	case strings.HasPrefix(spec, "jack:"):
		pair := strings.TrimPrefix(spec, "jack:")
		if key, ok := namedPairs[pair]; ok {
			if strings.HasPrefix(key, "sCall") {
				return fmt.Sprintf(t["sCall"], key[len("sCall")]-'0')
			}
			return t[key]
		}
		var a, b string
		if n, _ := fmt.Sscanf(pair, "JUCEJack:out_%s", &a); n == 1 {
			if i := strings.IndexByte(a, ','); i > 0 {
				a, b = a[:i], strings.TrimPrefix(a[i+1:], "JUCEJack:out_")
				return fmt.Sprintf(t["sMixOut"], a, b)
			}
		}
		return strings.ReplaceAll(pair, ",", " + ")
	case strings.HasPrefix(spec, "alsa:"):
		return "ALSA " + strings.TrimPrefix(spec, "alsa:")
	}
	return spec
}

// screenSources lists the sources the page offers: the desk's JACK pairs by
// name, then the test signal. The list is asked of the host and JACK at
// most every few seconds; the pictures between use the last one.
func (h *hub) screenSources(ctx context.Context) []string {
	h.mu.Lock()
	if h.sources != nil && time.Since(h.sourcesAt) < 5*time.Second {
		out := h.sources
		h.mu.Unlock()
		return out
	}
	h.mu.Unlock()
	var out []string
	for _, p := range listJackPairs(ctx) {
		if _, ok := namedPairs[p]; ok {
			out = append(out, "jack:"+p)
		}
	}
	out = append(out, "test")
	h.mu.Lock()
	h.sources, h.sourcesAt = out, time.Now()
	h.mu.Unlock()
	return out
}

func (h *hub) handleScreen(w http.ResponseWriter, r *http.Request) {
	t := screenLang(r)
	if r.Method == http.MethodPost {
		h.handleScreenChange(w, r, t)
		return
	}
	if strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
		h.streamScreen(w, r, t)
		return
	}
	sdk.WriteJSON(w, http.StatusOK, h.screenPage(t, h.screenSources(r.Context())))
}

// screenPage is the page: the radar, what Sonar hears and what can be set.
func (h *hub) screenPage(t map[string]string, sources []string) map[string]interface{} {
	now := time.Now()
	h.mu.Lock()
	e, spec, srcErr, cfg := h.last, h.srcSpec, h.srcErr, h.cfg
	trail := append([]trailPt(nil), h.trail...)
	h.mu.Unlock()

	source := sourceName(t, spec)
	if srcErr != "" {
		source += " · " + srcErr
	}
	options := make([]string, len(sources))
	sel := -1
	for i, s := range sources {
		options[i] = sourceName(t, s)
		if s == spec {
			sel = i
		}
	}
	if sel < 0 {
		options = append(options, sourceName(t, spec))
		sel = len(options) - 1
	}
	controls := []screenControl{
		{ID: "radar", Type: "canvas", W: 100, H: 100, Shapes: radar(t, e, cfg, trail, now)},
		{ID: "source", Type: "value", Label: t["source"], Text: source},
		{ID: "listen", Type: "choice", Label: t["listen"], Options: options, Value: float64(sel)},
		{ID: "smoothing", Type: "slider", Label: t["smoothing"], Value: cfg.Smoothing, Min: 0, Max: 0.99},
		{ID: "flip", Type: "toggle", Label: t["flip"], On: cfg.FlipSign},
		{ID: "near", Type: "button", Label: t["near"]},
		{ID: "far", Type: "button", Label: t["far"]},
		{ID: "reset", Type: "button", Label: t["reset"], Danger: true, Confirm: t["resetConfirm"]},
	}
	return map[string]interface{}{"title": "Sonar", "icon": "IconListen", "every": screenEvery, "controls": controls}
}

// streamScreen sends the page as an event stream: one picture now and one
// with every estimate (the web's rate), each the whole page, so the screen
// shows the sound as it happens instead of asking ten times a second.
func (h *hub) streamScreen(w http.ResponseWriter, r *http.Request, t map[string]string) {
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	ch := make(chan struct{}, 1)
	h.mu.Lock()
	if len(h.screens) >= maxStreams {
		h.mu.Unlock()
		sdk.WriteError(w, sdk.NewError(http.StatusServiceUnavailable, "too_many_clients", "too many live connections"))
		return
	}
	h.screens[ch] = struct{}{}
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		delete(h.screens, ch)
		h.mu.Unlock()
	}()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	send := func() bool {
		payload, err := json.Marshal(h.screenPage(t, h.screenSources(r.Context())))
		if err != nil {
			return false
		}
		if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
			return false
		}
		fl.Flush()
		return true
	}
	if !send() {
		return
	}
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ch:
			if !send() {
				return
			}
		}
	}
}

// handleScreenChange applies one control's new value from the screen.
func (h *hub) handleScreenChange(w http.ResponseWriter, r *http.Request, t map[string]string) {
	var in struct {
		Control string      `json:"control"`
		Value   interface{} `json:"value"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&in); err != nil {
		sdk.WriteError(w, sdk.NewError(http.StatusBadRequest, "bad_request", err.Error()))
		return
	}
	num, _ := in.Value.(float64)
	on, _ := in.Value.(bool)
	var p configPatch
	str := func(s string) *string { return &s }
	switch in.Control {
	case "listen":
		sources := h.screenSources(r.Context())
		i := int(num)
		if i < 0 || i >= len(sources) {
			sdk.WriteError(w, sdk.NewError(http.StatusBadRequest, "bad_request", "no such source"))
			return
		}
		p.Source = str(sources[i])
	case "smoothing":
		p.Smoothing = &num
	case "flip":
		p.Flip = &on
	case "near", "far", "reset":
		p.Calibrate = str(in.Control)
	default:
		sdk.WriteError(w, sdk.NewError(http.StatusBadRequest, "bad_request", "unknown control"))
		return
	}
	if err := h.apply(p); err != nil {
		sdk.WriteError(w, err)
		return
	}
	sdk.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
	_ = t
}
