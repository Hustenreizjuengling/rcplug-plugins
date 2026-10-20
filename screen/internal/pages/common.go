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

// sourceIcon maps a source id to one of the device's icons, or "" when there
// is no obvious one (the UI then shows the label alone).
func sourceIcon(id string) string {
	switch {
	case strings.HasPrefix(id, "combo"):
		return "IconMicrophone"
	case strings.HasPrefix(id, "usb"):
		return "IconUSB"
	case id == "bluetooth":
		return "IconBluetooth"
	case id == "soundpad":
		return "IconSmartPad"
	case id == "chat":
		return "IconChat"
	case strings.HasPrefix(id, "callme"):
		return "IconCallMe"
	case id == "game", id == "music", strings.HasPrefix(id, "virtual"):
		return "IconUSB"
	}
	return ""
}
