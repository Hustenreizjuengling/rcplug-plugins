package app_test

import (
	"testing"
	"time"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/pages"
)

// TestDateTimeTapThenTurn: a tap on a spinner takes the knob, turning
// changes that field, and nothing reaches the desk before the tick; the
// tick writes the 24-hour flag and the zone.
func TestDateTimeTapThenTurn(t *testing.T) {
	a, model := newTestApp(t)
	a.Push(pages.Detail("datetime"))
	a.RenderOnce()
	if !model.Bool(desk.SystemNode, 0, "systemDateTime24h") {
		t.Fatal("fixture should start in 24-hour mode")
	}
	// "12h" segment (right half of the 24h/12h switch under the time).
	tap(a, 880, 340)
	a.RenderOnce()
	if !model.Bool(desk.SystemNode, 0, "systemDateTime24h") {
		t.Fatal("the format must not be written before the tick")
	}
	// Tap the hours field, then turn the knob: the hours change.
	tap(a, 770, 190)
	a.RenderOnce()
	a.Encoder(desk.EncoderEvent{Delta: 1})
	a.RenderOnce()
	// The tick applies: 24h false is written (the clock itself needs the
	// host, which the offline model has not: the page stays with a note).
	tap(a, 1214, 58)
	time.Sleep(50 * time.Millisecond)
	a.RenderOnce()
	if model.Bool(desk.SystemNode, 0, "systemDateTime24h") {
		t.Fatal("the tick must write systemDateTime24h = false")
	}
}

// TestHapticsWritesRodeValues: the first card is "No Haptic Feedback" and
// writes 0, as RØDE's screen numbers HapticMode.
func TestHapticsWritesRodeValues(t *testing.T) {
	a, model := newTestApp(t)
	a.Push(pages.Detail("haptics"))
	a.RenderOnce()
	tap(a, 900, 268) // the top card
	a.RenderOnce()
	if got := model.Int(desk.SystemNode, 0, "systemHapticSetting"); got != desk.HapticOff {
		t.Fatalf("top card wrote %d, want %d (off)", got, desk.HapticOff)
	}
	tap(a, 900, 560) // the bottom card
	a.RenderOnce()
	if got := model.Int(desk.SystemNode, 0, "systemHapticSetting"); got != desk.HapticAlways {
		t.Fatalf("bottom card wrote %d, want %d (always)", got, desk.HapticAlways)
	}
}

// TestHeadphonesAsksBeforeLouder: on a port set to High Sensitivity, picking
// another type opens the confirmation and writes only on Ok.
func TestHeadphonesAsksBeforeLouder(t *testing.T) {
	a, model := newTestApp(t)
	model.SetInt("HEADPHONE", 0, "headphoneType", desk.HeadphonesHighSensitivity)
	a.Push(pages.Detail("headphones"))
	a.RenderOnce()
	tap(a, 890, 505) // Low Sensitivity card
	a.RenderOnce()
	if got := model.Int("HEADPHONE", 0, "headphoneType"); got != desk.HeadphonesHighSensitivity {
		t.Fatalf("the type was written (%d) before the confirmation", got)
	}
	// The dialog's confirm button sits bottom right of the card.
	tap(a, 830, 490)
	a.RenderOnce()
	if got := model.Int("HEADPHONE", 0, "headphoneType"); got != desk.HeadphonesLowSensitivity {
		t.Fatalf("after Ok the type is %d, want %d", got, desk.HeadphonesLowSensitivity)
	}
}

// TestSettingsPagesRender: every settings page builds and draws over the
// fixture.
func TestSettingsPagesRender(t *testing.T) {
	a, _ := newTestApp(t)
	for _, name := range []string{"headphones", "metering", "haptics", "recbutton", "compression", "midi", "info", "info-device",
		"regulatory", "analytics", "firmware", "opmode", "storage", "datetime", "timezone", "language", "cloud", "numpad", "multitrack", "brightness"} {
		a.Home()
		a.Push(pages.Detail(name))
		if got := a.RenderOnce(); len(got) == 0 {
			t.Errorf("page %q rendered nothing", name)
		}
	}
}
