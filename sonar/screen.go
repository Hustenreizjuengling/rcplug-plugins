package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/Hustenreizjuengling/rcplug/sdk"
)

// Sonar's page on the device's screen (the `screen` plugin): a description
// of controls the screen draws itself, served at /screen, and the changes
// it sends back. Sonar never draws on the screen; it says what it hears
// and what can be set, in the screen's language (?lang=de or en).

// screenControl is one control of the page, as the screen plugin reads it.
type screenControl struct {
	ID      string   `json:"id"`
	Type    string   `json:"type"` // value meter choice slider toggle button
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
}

// screenTexts are the page's words in English and German.
var screenTexts = map[string]map[string]string{
	"en": {
		"source": "Listening to", "bearing": "Direction", "level": "Level", "distance": "Distance",
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
		"source": "Hört auf", "bearing": "Richtung", "level": "Pegel", "distance": "Entfernung",
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
// name, then the test signal.
func (h *hub) screenSources(r *http.Request) []string {
	var out []string
	for _, p := range listJackPairs(r.Context()) {
		if _, ok := namedPairs[p]; ok {
			out = append(out, "jack:"+p)
		}
	}
	return append(out, "test")
}

func (h *hub) handleScreen(w http.ResponseWriter, r *http.Request) {
	t := screenLang(r)
	if r.Method == http.MethodPost {
		h.handleScreenChange(w, r, t)
		return
	}
	sources := h.screenSources(r)
	h.mu.Lock()
	e, spec, srcErr, cfg := h.last, h.srcSpec, h.srcErr, h.cfg
	h.mu.Unlock()

	source := sourceName(t, spec)
	if srcErr != "" {
		source += " · " + srcErr
	}
	bearing := t["quiet"]
	if e.Conf > 0.2 && e.LevelDb > cfg.NoiseFloor {
		deg := int(math.Round(math.Abs(e.Azimuth)))
		switch {
		case deg < 5:
			bearing = t["centre"]
		case e.Azimuth < 0:
			bearing = fmt.Sprintf("%d° %s", deg, t["left"])
		default:
			bearing = fmt.Sprintf("%d° %s", deg, t["right"])
		}
	}
	distance := t["none"]
	if cfg.Calibrated() && e.LevelDb > cfg.NoiseFloor {
		switch {
		case e.Dist < 0.33:
			distance = t["nearV"]
		case e.Dist < 0.66:
			distance = t["midV"]
		default:
			distance = t["farV"]
		}
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
		{ID: "source", Type: "value", Label: t["source"], Text: source},
		{ID: "bearing", Type: "value", Label: t["bearing"], Text: bearing},
		{ID: "level", Type: "meter", Label: t["level"], Value: e.Level, Min: 0, Max: 1},
		{ID: "distance", Type: "value", Label: t["distance"], Text: distance},
		{ID: "listen", Type: "choice", Label: t["listen"], Options: options, Value: float64(sel)},
		{ID: "smoothing", Type: "slider", Label: t["smoothing"], Value: cfg.Smoothing, Min: 0, Max: 0.99},
		{ID: "flip", Type: "toggle", Label: t["flip"], On: cfg.FlipSign},
		{ID: "near", Type: "button", Label: t["near"]},
		{ID: "far", Type: "button", Label: t["far"]},
		{ID: "reset", Type: "button", Label: t["reset"], Danger: true, Confirm: t["resetConfirm"]},
	}
	sdk.WriteJSON(w, http.StatusOK, map[string]interface{}{"title": "Sonar", "controls": controls})
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
		sources := h.screenSources(r)
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
