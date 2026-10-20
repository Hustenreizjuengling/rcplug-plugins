package pages

import (
	"strconv"
	"strings"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
)

// parseColour reads an ARGB or RGB hex colour as the desk stores it (e.g.
// "ffEE7722" or "#EE7722"), returning fallback when it cannot.
func parseColour(s string, fallback gfx.Color) gfx.Color {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if s == "" {
		return fallback
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return fallback
	}
	switch len(s) {
	case 8: // AARRGGBB
		return gfx.Color(uint32(v))
	case 6: // RRGGBB
		return gfx.Color(0xff000000 | uint32(v))
	}
	return fallback
}

// dbText formats a dB number the way the device shows it (one decimal, a
// sign, "−∞" at the bottom).
func dbText(db float64) string {
	if db <= -100 {
		return "−∞"
	}
	sign := ""
	if db > 0 {
		sign = "+"
	}
	return sign + strconv.FormatFloat(db, 'f', 1, 64) + " dB"
}

// pct formats a 0..1 fraction as a percentage.
func pct(v float64) string {
	return strconv.Itoa(int(v*100+0.5)) + "%"
}

// intText formats a whole number.
func intText(v int64) string { return strconv.FormatInt(v, 10) }

// sourceIcon maps a source id to the device's strip icon (its own names), or
// "" when there is none (the UI then shows the label alone).
var sourceIcons = map[string]string{
	"combo1": "IconStripMic1", "combo2": "IconStripMic2", "combo3": "IconStripMic3", "combo4": "IconStripMic4",
	"combo1_2": "IconStripMics12", "combo2_3": "IconStripMics23", "combo3_4": "IconStripMics34",
	"usb1": "IconStripUsb1Main", "chat": "IconStripUsb1Comms", "usb2": "IconStripUsb2Main",
	"bluetooth": "bluetooth", "soundpad": "smart_pads_sound",
	"game": "VirtualInput_Game", "music": "VirtualInput_Music",
	"virtuala": "IconStripUsb2Main", "virtualb": "IconStripUsb2Main",
	"callme1": "IconCallMe1", "callme2": "IconCallMe2", "callme3": "IconCallMe3",
}

func sourceIcon(id string) string {
	if ic, ok := sourceIcons[id]; ok {
		return ic
	}
	if strings.HasPrefix(id, "combo") {
		return "IconStripMic1"
	}
	return ""
}
