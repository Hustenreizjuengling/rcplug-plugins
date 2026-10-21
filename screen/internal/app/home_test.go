package app_test

import (
	"testing"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/pages"
)

// TestEncoderRingFollowsPages: the main knob's ring is set as RØDE's screen
// sets it, 7 on the mixer and -1 on every other page and under a dialog.
func TestEncoderRingFollowsPages(t *testing.T) {
	a, m := newTestApp(t)
	ring := func() int64 { return m.Int("ENCODER", 0, "encoderColour") }
	if ring() != desk.EncoderRingHome {
		t.Fatalf("ring on the mixer %d, want %d", ring(), desk.EncoderRingHome)
	}
	a.Open(pages.IDSettings)
	if ring() != desk.EncoderRingOff {
		t.Fatalf("ring on a menu page %d, want %d", ring(), desk.EncoderRingOff)
	}
	a.Home()
	if ring() != desk.EncoderRingHome {
		t.Fatalf("ring back on the mixer %d, want %d", ring(), desk.EncoderRingHome)
	}
	a.Confirm("T", "M", "OK", false, nil)
	if ring() != desk.EncoderRingOff {
		t.Fatalf("ring under a dialog %d, want %d", ring(), desk.EncoderRingOff)
	}
}

// TestSetupAssistantSequence walks the assistant with the arrow: the steps
// follow the device's numbering and the per-input steps that do not apply
// (the wireless pairing of a combo input, the inputs the desk has not got)
// are skipped; the last arrow returns to the mixer.
func TestSetupAssistantSequence(t *testing.T) {
	a, m := newTestApp(t)
	a.Push(pages.NewSetupAssistant(nil))
	a.RenderOnce()
	w, ok := a.Page().(*pages.SetupAssistantPage)
	if !ok {
		t.Fatalf("top page %T", a.Page())
	}
	// The fixture: a Duo with two combo inputs carrying microphones.
	want := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 10, 11, 12, 14, 15, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e}
	for i, step := range want {
		if w.Step() != step {
			t.Fatalf("step %d: at %#x, want %#x", i, w.Step(), step)
		}
		tap(a, 1195, 635) // the arrow at the bottom right
		a.RenderOnce()
	}
	if a.Depth() != 1 {
		t.Fatalf("after the last arrow the mixer should show, depth %d", a.Depth())
	}
	// Walking through wrote nothing the user did not change: the inputs
	// keep their sources.
	if m.SourceOf(0) != 0 || m.SourceOf(1) != 1 {
		t.Fatalf("sources changed: %d %d", m.SourceOf(0), m.SourceOf(1))
	}
}

// TestSetupAssistantCross: the cross leaves the assistant at any step.
func TestSetupAssistantCross(t *testing.T) {
	a, _ := newTestApp(t)
	w := pages.NewSetupAssistant(nil).(*pages.SetupAssistantPage)
	w.SetStep(6)
	a.Push(w)
	a.RenderOnce()
	tap(a, 55, 55)
	a.RenderOnce()
	if a.Depth() != 1 {
		t.Fatalf("the cross should return to the mixer, depth %d", a.Depth())
	}
}

// TestWelcomeFlow: Get Started, then language, date and time and getting
// connected with the tick, then the assistant takes the page's place.
func TestWelcomeFlow(t *testing.T) {
	a, _ := newTestApp(t)
	a.Push(pages.NewWelcome(nil))
	a.RenderOnce()
	w := a.Page().(*pages.WelcomePage)
	tap(a, 640, 525) // Get Started!
	a.RenderOnce()
	for _, step := range []int{1, 2, 3} {
		if w.Step() != step {
			t.Fatalf("at step %d, want %d", w.Step(), step)
		}
		tap(a, 1214, 58) // the tick
		a.RenderOnce()
	}
	if _, ok := a.Page().(*pages.SetupAssistantPage); !ok || a.Depth() != 2 {
		t.Fatalf("after getting connected the assistant should show: %T depth %d", a.Page(), a.Depth())
	}
}

// TestRoutingTilesOpenMix: a routing tile opens that output's mix page, the
// mode switch writes MIXMINUSES[o].outputMixMinus.
func TestRoutingTilesOpenMix(t *testing.T) {
	a, m := newTestApp(t)
	a.Open(pages.IDRouting)
	a.RenderOnce()
	tap(a, 640, 490) // the USB 1 tile in the second row
	a.RenderOnce()
	if a.Depth() != 3 {
		t.Fatalf("depth after the tile %d, want 3", a.Depth())
	}
	// USB 1 is output 7; its switch offers Main Mix, Mix-minus, Custom.
	tap(a, 640, 200) // the middle cell: Mix-minus
	a.RenderOnce()
	if got := m.Int(desk.MixMinusNode, 7, "outputMixMinus"); got != 1 {
		t.Fatalf("outputMixMinus of USB 1 = %d, want 1", got)
	}
}
