package pages

import (
	"fmt"
	"image"
	"strconv"
	"strings"
	"time"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
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

// dbNum is dbText without the unit (the stepper draws the unit itself).
func dbNum(db float64) string {
	if db <= -100 {
		return "−∞"
	}
	return strconv.FormatFloat(db, 'f', 0, 64)
}

// pct formats a 0..1 fraction as a percentage.
func pct(v float64) string {
	return strconv.Itoa(int(v*100+0.5)) + "%"
}

// intText formats a whole number.
func intText(v int64) string { return strconv.FormatInt(v, 10) }

// clockText formats a time as the device shows it (1.7.x): "04:39:52" with
// a small "am"/"pm" beside it on the 12-hour clock, "16:39:52" on the
// 24-hour clock.
func clockText(t time.Time, h24 bool) (main, suffix string) {
	if h24 {
		return t.Format("15:04:05"), ""
	}
	return t.Format("03:04:05"), t.Format("pm")
}

// dateText formats a date as the device shows it: "07 February 2025". The
// month names are not translated (the device's screen keeps the English
// names in every language).
func dateText(t time.Time) string {
	return t.Format("02 January 2006")
}

// hoursText shortens a duration to the device's "12+ hrs" / "45 min".
func hoursText(d time.Duration) string {
	switch {
	case d <= 0:
		return "—"
	case d >= 12*time.Hour:
		return "12+ hrs"
	case d >= time.Hour:
		return strconv.Itoa(int(d.Hours())) + " hrs"
	default:
		return strconv.Itoa(int(d.Minutes())) + " min"
	}
}

// recTime formats a recording's length as HH:MM:SS.
func recTime(d time.Duration) string {
	s := int(d.Seconds())
	return fmt.Sprintf("%02d:%02d:%02d", s/3600, (s%3600)/60, s%60)
}

// sourceIcon maps a source id to the device's strip icon (its own names), or
// "" when there is none (the UI then shows the label alone).
var sourceIcons = map[string]string{
	"combo1": "IconStripMic1", "combo2": "IconStripMic2", "combo3": "IconStripMic3", "combo4": "IconStripMic4",
	"combo1_2": "IconStripMics12", "combo2_3": "IconStripMics23", "combo3_4": "IconStripMics34",
	"usb1": "IconStripUsb1Main", "chat": "IconStripUsb1Comms", "usb2": "IconStripUsb2Main",
	"bluetooth": "bluetooth", "soundpad": "smart_pads_sound",
	"game": "VirtualInput_Game", "music": "VirtualInput_Music",
	"virtuala": "Letter_A", "virtualb": "Letter_B",
	"callme1": "CallMeFaderIcon1", "callme2": "CallMeFaderIcon2", "callme3": "CallMeFaderIcon3",
	"wireless1": "IconStripWireless1", "wireless2": "IconStripWireless2", "headset": "headset",
}

func sourceIcon(id string) string {
	if ic, ok := sourceIcons[id]; ok {
		return ic
	}
	if strings.HasPrefix(id, "combo") {
		return "IconStripMic1"
	}
	if strings.HasPrefix(id, "sync") {
		return "iconRCSync"
	}
	return ""
}

// bigSourceIcon is the larger picture of a source for the channel and
// assignment pages (the strip icons carry the number; these are plainer).
var bigSourceIcons = map[string]string{
	"combo1": "IconMic1", "combo2": "IconMic2", "combo3": "IconMic3", "combo4": "IconMic4",
	"combo1_2": "IconMics12", "combo2_3": "IconMics23", "combo3_4": "IconMics34",
	"usb1": "IconUsb1Main", "chat": "IconUsb1Comms", "usb2": "IconUsb2Main",
	"bluetooth": "bluetooth", "soundpad": "smart_pads_sound",
	"callme1": "CallMe_1", "callme2": "CallMe_2", "callme3": "CallMe_3",
	"wireless1": "IconWireless1", "wireless2": "IconWireless2", "headset": "headset",
}

func sourceIconBig(id string) string {
	if ic, ok := bigSourceIcons[id]; ok {
		return ic
	}
	return sourceIcon(id)
}

// outputIcon is the routing picture of an output bus.
var outputIcons = map[string]string{
	"headphone1": "IconHeadphones", "headphone2": "IconHeadphones", "headphone3": "IconHeadphones", "headphone4": "IconHeadphones",
	"speaker": "routing_monitor", "recording": "routing_record", "bluetooth": "routing_bluetooth",
	"usb1": "routing_usb1_main", "chat": "routing_usb1_chat", "usb2": "routing_usb2_main",
	"callme1": "CallMe_1", "callme2": "CallMe_2", "callme3": "CallMe_3",
}

func outputIcon(id string) string {
	if ic, ok := outputIcons[id]; ok {
		return ic
	}
	return "IconRouting"
}

// padPalette are the SMART-pad colours by index (padColourIndex 0…11), as
// RØDE's screen program keeps them in its colour table at 0xbc93b0 (the
// four headphone colours come first; the twelve pad colours follow): red,
// orange, amber, yellow, lime, green, teal, cyan, blue, violet, purple,
// magenta.
var padPalette = []gfx.Color{
	gfx.Hex(0xdd1919), gfx.Hex(0xff7a00), gfx.Hex(0xffb800), gfx.Hex(0xffe600),
	gfx.Hex(0x7ecc00), gfx.Hex(0x00b800), gfx.Hex(0x00daa6), gfx.Hex(0x00cbf8),
	gfx.Hex(0x0047ff), gfx.Hex(0x7000ff), gfx.Hex(0xad00ff), gfx.Hex(0xff00e5),
}

func padColour(i int64) gfx.Color {
	if i >= 0 && int(i) < len(padPalette) {
		return padPalette[i]
	}
	return gfx.Hex(0x9a9a9a)
}

// padTypeIcon is the picture of a pad's kind (RØDE's tile icons by padType).
func padTypeIcon(t int64) string {
	switch t {
	case 2:
		return "smart_pads_effect"
	case 3:
		return "smart_pads_mixer"
	case 4:
		return "smart_pads_trigger"
	case 5:
		return "smart_pads_quick_dial"
	case 6:
		return "smart_pads_video"
	case 7:
		return "pads_icecast"
	}
	return "smart_pads_sound"
}

// updater is an invisible view that runs get each frame (for labels/toggles
// whose source changes). It draws nothing and takes no space.
type updater struct {
	ui.Base
	get func()
}

func (u *updater) Dirty() []image.Rectangle {
	if u.get != nil {
		u.get()
	}
	return nil
}
func (u *updater) Draw(*gfx.Image, *ui.Context, image.Rectangle) {}
func (u *updater) Touch(ui.TouchPhase, image.Point) bool         { return false }

// strip switching, shared by the channel and rack pages.

// stripLabel is the caption of a strip: its source, or "not assigned".
func stripLabel(env *Env, strip int) string {
	n := env.Desk.SourceName(env.Desk.SourceOf(strip))
	if n.Label == "" {
		return env.T("NOT ASSIGNED")
	}
	return n.Label
}

// stripColour is the colour of a strip's source (the accent if none).
func stripColour(env *Env, strip int) gfx.Color {
	th := env.Ctx.Theme
	if env.Desk.SourceOf(strip) < 0 {
		return th.TextMuted
	}
	return parseColour(env.Desk.InputColour(strip), th.Text)
}
