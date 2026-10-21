package app_test

import (
	"testing"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/pages"
)

// TestRoutingMixSelectsOutput: a routing mix page on the stack tells the
// desk its output (SYSTEM.systemMixSelected), as RØDE's RoutingMixPage does;
// switching to the next output follows, leaving writes 13.
func TestRoutingMixSelectsOutput(t *testing.T) {
	a, model := newTestApp(t)
	sel := func() int64 { return model.Int(desk.SystemNode, 0, "systemMixSelected") }
	if sel() != desk.MixSelectedNone {
		t.Fatalf("fixture starts at %d, want %d", sel(), desk.MixSelectedNone)
	}
	a.Push(pages.RoutingMix(4))
	a.RenderOnce()
	if sel() != 4 {
		t.Fatalf("with the monitor's mix open the desk reads %d, want 4", sel())
	}
	a.Replace(pages.RoutingMix(7))
	a.RenderOnce()
	if sel() != 7 {
		t.Fatalf("after switching to USB 1 the desk reads %d, want 7", sel())
	}
	a.Pop()
	a.RenderOnce()
	if sel() != desk.MixSelectedNone {
		t.Fatalf("after leaving the desk reads %d, want %d", sel(), desk.MixSelectedNone)
	}
}

// TestOperatingModeWritesMainMode: the Sync card asks first and then writes
// systemMainMode 2; Podcast from Sync mode asks and pulses the sync-mode
// request; Podcast from Karaoke writes 0 at once.
func TestOperatingModeWritesMainMode(t *testing.T) {
	a, model := newTestApp(t)
	mode := func() int64 { return model.Int(desk.SystemNode, 0, "systemMainMode") }
	a.Push(pages.Detail("opmode"))
	a.RenderOnce()
	tap(a, 1098, 252) // RØDECaster Sync (BETA)
	a.RenderOnce()
	if mode() != desk.MainModePodcast {
		t.Fatalf("the mode was written (%d) before the confirmation", mode())
	}
	tap(a, 830, 490) // Confirm
	a.RenderOnce()
	if mode() != desk.MainModeSync {
		t.Fatalf("after Confirm the mode is %d, want %d", mode(), desk.MainModeSync)
	}
	tap(a, 182, 252) // Podcast, while in Sync mode
	a.RenderOnce()
	if mode() != desk.MainModeSync || model.Bool(desk.SystemNode, 0, "rcSyncModeRequest") {
		t.Fatal("leaving Sync mode must wait for the confirmation")
	}
	tap(a, 830, 490) // Confirm
	a.RenderOnce()
	if !model.Bool(desk.SystemNode, 0, "rcSyncModeRequest") {
		t.Fatal("leaving Sync mode pulses rcSyncModeRequest")
	}
	if mode() != desk.MainModeSync {
		t.Fatalf("the desk changes the mode itself; the screen wrote %d", mode())
	}
	model.SetMainMode(desk.MainModeKaraoke)
	tap(a, 182, 252) // Podcast, from Karaoke: no question
	a.RenderOnce()
	if mode() != desk.MainModePodcast {
		t.Fatalf("Podcast from Karaoke writes 0, got %d", mode())
	}
}

// TestDimAfterWritesSeconds: the Dim After buttons write the seconds as a
// number, as RØDE's screen does, whatever type the desk started with.
func TestDimAfterWritesSeconds(t *testing.T) {
	a, model := newTestApp(t)
	a.Push(pages.Detail("brightness"))
	a.RenderOnce()
	tap(a, 799, 464) // 1 min
	a.RenderOnce()
	v, ok := model.Get(desk.GUINode, 0, "screenDimAfterSeconds")
	if !ok || v.Kind == 'b' || v.Int() != 60 {
		t.Fatalf("Dim After wrote %+v, want the number 60", v)
	}
	tap(a, 1195, 464) // Never
	a.RenderOnce()
	if v, _ := model.Get(desk.GUINode, 0, "screenDimAfterSeconds"); v.Kind == 'b' || v.Int() != 0 {
		t.Fatalf("Never wrote %+v, want the number 0", v)
	}
}

// TestProcessorPagesSetMeterSource: the compressor, de-esser, noise gate and
// panning pages put their reading on the assignable meter when they come on
// screen; the channel page's gain panel puts the input back when they go.
func TestProcessorPagesSetMeterSource(t *testing.T) {
	a, model := newTestApp(t)
	src := func() int64 { return model.Int(desk.SystemNode, 0, "assignableMeterSource") }
	model.SetInt(desk.SystemNode, 0, "assignableMeterSource", 2)
	a.Push(pages.NewChannel(0))
	a.RenderOnce()
	if src() != desk.MeterSourceInput {
		t.Fatalf("the channel page shows the input: %d", src())
	}
	for name, want := range map[string]int64{"comp": desk.MeterSourceCompressor, "deesser": desk.MeterSourceDeesser, "gate": desk.MeterSourceNoiseGate, "pan": desk.MeterSourcePan} {
		a.Push(pages.Processor(0, name))
		a.RenderOnce()
		if src() != want {
			t.Fatalf("%s page: meter source %d, want %d", name, src(), want)
		}
		a.Pop()
		a.RenderOnce()
		if src() != desk.MeterSourceInput {
			t.Fatalf("back from %s: meter source %d, want 0", name, src())
		}
	}
	a.Push(pages.Processor(0, "comp"))
	a.RenderOnce()
	a.Push(pages.Processor(0, "eq"))
	a.RenderOnce()
	if src() != desk.MeterSourceCompressor {
		t.Fatalf("the EQ page leaves the meter source: %d", src())
	}
}

// TestLeavingWirelessChannelCancelsPairing: a wireless channel page that
// goes off screen while its slot is pairing writes RADIO.radioPair -1, as
// RØDE's wireless panel does when hidden.
func TestLeavingWirelessChannelCancelsPairing(t *testing.T) {
	a, model := newTestApp(t)
	strip := -1
	for i := 0; i < model.StripCount(); i++ {
		if model.SourceKindOf(model.SourceOf(i)) == desk.KindWireless {
			strip = i
			break
		}
	}
	if strip < 0 {
		t.Skip("the fixture has no wireless strip")
	}
	a.Push(pages.NewChannel(strip))
	a.RenderOnce()
	model.StartPairing(model.SourceOf(strip))
	if !model.Pairing() {
		t.Fatal("pairing should have started")
	}
	a.Pop()
	a.RenderOnce()
	if model.Pairing() || model.Int(desk.RadioNode, 0, "radioPair") != -1 {
		t.Fatalf("leaving the page leaves radioPair at %d", model.Int(desk.RadioNode, 0, "radioPair"))
	}
}
