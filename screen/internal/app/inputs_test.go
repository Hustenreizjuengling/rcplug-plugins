package app_test

import (
	"image"
	"testing"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/pages"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// TestAdvancedButton: the Advanced button writes channelAdvancedProcessing
// and drops the preset, as RØDE's does; a second tap asks (Cancel / Save
// Preset / Clear) and "Clear" leaves advanced mode.
func TestAdvancedButton(t *testing.T) {
	a, model := newTestApp(t)
	a.Push(pages.NewChannel(0))
	a.RenderOnce()
	if model.Bool(desk.ChannelNode, 0, "channelAdvancedProcessing") {
		t.Fatal("fixture strip 0 should start in simple mode")
	}
	tap(a, 1081, 652) // Advanced
	a.RenderOnce()
	if !model.Bool(desk.ChannelNode, 0, "channelAdvancedProcessing") || model.CurrentPreset(0) != -1 {
		t.Fatalf("advanced %v, preset %d; want true and -1", model.Bool(desk.ChannelNode, 0, "channelAdvancedProcessing"), model.CurrentPreset(0))
	}
	tap(a, 1081, 652) // Advanced again: the question
	a.RenderOnce()
	if !model.Bool(desk.ChannelNode, 0, "channelAdvancedProcessing") {
		t.Fatal("the question must not change the mode by itself")
	}
	tap(a, 929, 494) // "Clear"
	a.RenderOnce()
	if model.Bool(desk.ChannelNode, 0, "channelAdvancedProcessing") {
		t.Fatal("Clear leaves advanced mode")
	}
}

// TestVoxLabKnobDrives: dragging the Depth knob in simple mode writes the
// macro and the EQ gains it drives.
func TestVoxLabKnobDrives(t *testing.T) {
	a, model := newTestApp(t)
	a.Push(pages.NewChannel(0))
	a.RenderOnce()
	before := model.Num(desk.ChannelNode, 0, "eqLowGain")
	a.Touch(ui.Down, image.Pt(770, 350))
	a.Touch(ui.Move, image.Pt(770, 290))
	a.Touch(ui.Up, image.Pt(770, 290))
	a.RenderOnce()
	if d := model.Num(desk.ChannelNode, 0, "channelDepth"); d <= 0.5 {
		t.Fatalf("depth %v after an upward drag, want more than 0.5", d)
	}
	if after := model.Num(desk.ChannelNode, 0, "eqLowGain"); after <= before {
		t.Fatalf("eqLowGain %v → %v: the macro must drive it", before, after)
	}
}

// TestWirelessPair: "Pair New Device" on an unpaired wireless channel asks
// the receiver to pair that slot (RADIO.radioPair).
func TestWirelessPair(t *testing.T) {
	a, model := newTestApp(t)
	if model.SourceOf(8) != desk.SourceWireless2 {
		t.Fatalf("fixture strip 8 should carry wireless 2, has %d", model.SourceOf(8))
	}
	a.Push(pages.NewChannel(8))
	a.RenderOnce()
	tap(a, 328, 645)
	a.RenderOnce()
	if got := model.Int(desk.RadioNode, 0, "radioPair"); got != 1 {
		t.Fatalf("radioPair = %d after Pair New Device on wireless 2, want 1", got)
	}
	if !model.Pairing() {
		t.Fatal("the receiver should be pairing")
	}
}

// TestWirelessGainAssist: the GainAssist button steps Auto → Dynamic → Off
// → Auto on a paired transmitter.
func TestWirelessGainAssist(t *testing.T) {
	a, model := newTestApp(t)
	a.Push(pages.NewChannel(6))
	a.RenderOnce()
	tx := model.WirelessTx(desk.SourceWireless1)
	if tx < 0 || model.Int(desk.RadioTxNode, tx, "txGainAssist") != desk.GainAssistAuto {
		t.Fatalf("fixture transmitter %d should start on Auto", tx)
	}
	tap(a, 185, 645)
	a.RenderOnce()
	if got := model.Int(desk.RadioTxNode, tx, "txGainAssist"); got != desk.GainAssistDynamic {
		t.Fatalf("txGainAssist = %d, want Dynamic", got)
	}
	tap(a, 185, 645)
	tap(a, 185, 645)
	a.RenderOnce()
	if got := model.Int(desk.RadioTxNode, tx, "txGainAssist"); got != desk.GainAssistAuto {
		t.Fatalf("txGainAssist = %d after three taps, want Auto again", got)
	}
}

// TestFadersDrag: dropping a strip's tile on another swaps their sources.
func TestFadersDrag(t *testing.T) {
	a, model := newTestApp(t)
	a.Push(pages.Detail("faders"))
	a.RenderOnce()
	s0, s1 := model.SourceOf(0), model.SourceOf(1)
	a.Touch(ui.Down, image.Pt(90, 368))
	a.Touch(ui.Move, image.Pt(150, 368))
	a.Touch(ui.Move, image.Pt(220, 368))
	a.Touch(ui.Up, image.Pt(220, 368))
	a.RenderOnce()
	if model.SourceOf(0) != s1 || model.SourceOf(1) != s0 {
		t.Fatalf("after the drag strips 0/1 carry %d/%d, want %d/%d", model.SourceOf(0), model.SourceOf(1), s1, s0)
	}
	// A plain tap opens the channel page.
	depth := a.Depth()
	tap(a, 90, 368)
	a.RenderOnce()
	if a.Depth() != depth+1 {
		t.Fatal("a tap on a tile should open the channel page")
	}
}

// TestEqDotDrag: dragging a band's dot on the EQ graph writes its gain.
func TestEqDotDrag(t *testing.T) {
	a, model := newTestApp(t)
	a.Push(pages.Processor(0, "eq"))
	a.RenderOnce()
	before := model.Num(desk.ChannelNode, 0, "eqLowGain")
	a.Touch(ui.Down, image.Pt(118, 413))
	a.Touch(ui.Move, image.Pt(118, 375))
	a.Touch(ui.Up, image.Pt(118, 375))
	a.RenderOnce()
	if after := model.Num(desk.ChannelNode, 0, "eqLowGain"); after <= before+0.1 {
		t.Fatalf("eqLowGain %v → %v, want a rise of a few dB", before, after)
	}
}

// TestAssignHeadsetAndLink: the headset tile marks the Duo's third source
// and asks before taking it from the strip that has it; the chain link
// marks the stereo pair of inputs 1 and 2.
func TestAssignHeadsetAndLink(t *testing.T) {
	a, model := newTestApp(t)
	if model.SourceOf(7) != desk.SourceHeadset {
		t.Fatalf("fixture strip 7 should carry the headset, has %d", model.SourceOf(7))
	}
	a.Push(pages.NewAssign(0))
	a.RenderOnce()
	tap(a, 191, 597) // the headset tile
	tap(a, 1120, 50) // the tick: the headset is on strip 7 → the question
	a.RenderOnce()
	if model.SourceOf(0) != 0 {
		t.Fatal("nothing changes before the question is answered")
	}
	tap(a, 834, 484) // Re-assign
	a.RenderOnce()
	if model.SourceOf(0) != desk.SourceHeadset || model.SourceOf(7) != -1 {
		t.Fatalf("after Re-assign strips 0/7 carry %d/%d, want 2/-1", model.SourceOf(0), model.SourceOf(7))
	}
	a.Push(pages.NewAssign(1))
	a.RenderOnce()
	tap(a, 316, 318) // the chain link between inputs 1 and 2
	tap(a, 1120, 50)
	a.RenderOnce()
	if model.SourceOf(1) != desk.SourceComboPair {
		t.Fatalf("after the link strip 1 carries %d, want the pair (4)", model.SourceOf(1))
	}
}

// TestPresetsPage: a preset row applies the preset; the "+" opens the
// keyboard without writing anything.
func TestPresetsPage(t *testing.T) {
	a, model := newTestApp(t)
	a.Push(pages.NewPresets(0))
	a.RenderOnce()
	tap(a, 950, 198) // "Neutral"
	a.RenderOnce()
	if model.CurrentPreset(0) != 0 {
		t.Fatalf("current preset %d after a tap on Neutral, want 0", model.CurrentPreset(0))
	}
	n := len(model.Presets())
	tap(a, 1100, 652) // "+"
	a.RenderOnce()
	if len(model.Presets()) != n || a.Depth() != 2 {
		t.Fatal("the + only opens the keyboard")
	}
}
