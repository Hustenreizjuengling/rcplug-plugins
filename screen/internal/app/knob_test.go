package app_test

import (
	"testing"
	"time"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/app"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/lang"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/pages"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/preview"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

func newTestApp(t *testing.T) (*app.App, *desk.Model) {
	t.Helper()
	model := desk.NewOffline(nil)
	if err := model.LoadRaw(preview.BuildTree()); err != nil {
		t.Fatal(err)
	}
	ctx := ui.NewContext(ui.Dark(ui.Liberation()), nil)
	a := app.New(1280, 720, ctx, model, lang.New("de", nil), nil)
	a.RenderOnce()
	return a, model
}

// TestAssignOnlyWritesOnTick: marking a source on the assignment page changes
// nothing until the tick; the cross discards.
func TestAssignOnlyWritesOnTick(t *testing.T) {
	a, model := newTestApp(t)
	before := model.SourceOf(0)
	if before < 0 {
		t.Fatal("fixture strip 0 should carry a source")
	}
	a.Push(pages.NewAssign(0))
	a.RenderOnce()
	tap(a, 840, 664) // "not assigned" at the bottom of the grid
	a.RenderOnce()
	if got := model.SourceOf(0); got != before {
		t.Fatalf("a tap on the list wrote the source (%d); it must wait for the tick", got)
	}
	tap(a, 66, 58) // the cross: discard
	a.RenderOnce()
	if a.Depth() != 1 || model.SourceOf(0) != before {
		t.Fatalf("the cross must leave the source as it was (depth %d, source %d)", a.Depth(), model.SourceOf(0))
	}
	a.Push(pages.NewAssign(0))
	a.RenderOnce()
	tap(a, 840, 664)
	tap(a, 1214, 58) // the tick: apply
	a.RenderOnce()
	if got := model.SourceOf(0); got != -1 {
		t.Fatalf("after the tick the source is %d, want -1", got)
	}
}

// TestKnobOnMenus: turning focuses the first control (back), a short press
// taps it, a hold goes back.
func TestKnobOnMenus(t *testing.T) {
	a, _ := newTestApp(t)
	a.Open(pages.IDSettings)
	a.RenderOnce()
	a.Encoder(desk.EncoderEvent{Delta: 1})
	a.RenderOnce()
	a.Encoder(desk.EncoderEvent{Release: true, Held: 100 * time.Millisecond}) // taps "back"
	a.RenderOnce()
	if a.Depth() != 1 {
		t.Fatalf("pressing the knob on the back button did not go back (depth %d)", a.Depth())
	}
	a.Open(pages.IDSettings)
	a.RenderOnce()
	a.Encoder(desk.EncoderEvent{Release: true, Held: 2 * time.Second}) // a hold = back
	a.RenderOnce()
	if a.Depth() != 1 {
		t.Fatalf("holding the knob did not go back (depth %d)", a.Depth())
	}
}

// TestKnobOnMixer: on the mixer the knob sets the monitor level and a press
// moves on to the virtual faders.
func TestKnobOnMixer(t *testing.T) {
	a, model := newTestApp(t)
	lv := model.MonitorLevel()
	a.Encoder(desk.EncoderEvent{Delta: 2})
	if got := model.MonitorLevel(); got <= lv {
		t.Fatalf("turning on the mixer did not raise the monitor level (%v → %v)", lv, got)
	}
	a.Encoder(desk.EncoderEvent{Release: true, Held: 50 * time.Millisecond})
	strip := model.PhysicalFaders()
	before := model.FaderLevel(strip)
	a.Encoder(desk.EncoderEvent{Delta: -3})
	if got := model.FaderLevel(strip); got >= before {
		t.Fatalf("after a press the knob should move virtual strip %d (%d → %d)", strip, before, got)
	}
	if a.Depth() != 1 {
		t.Fatal("the mixer's knob must not navigate")
	}
}

// TestChannelButton: the button above a fader opens that strip's channel
// page, a second press on the same page returns to the mixer, and another
// strip's button switches the page.
func TestChannelButton(t *testing.T) {
	a, _ := newTestApp(t)
	a.ChannelButton(1)
	if a.Depth() != 2 {
		t.Fatalf("depth after the button: %d, want 2", a.Depth())
	}
	if ch, ok := a.Page().(*pages.ChannelPage); !ok || ch.Strip() != 1 {
		t.Fatalf("top page after the button: %T", a.Page())
	}
	a.ChannelButton(2)
	if ch, ok := a.Page().(*pages.ChannelPage); !ok || ch.Strip() != 2 || a.Depth() != 2 {
		t.Fatalf("another strip's button should switch the page: %T depth %d", a.Page(), a.Depth())
	}
	a.ChannelButton(2)
	if a.Depth() != 1 {
		t.Fatalf("the same button again should return to the mixer, depth %d", a.Depth())
	}
}
