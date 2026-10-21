package app_test

import (
	"testing"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/app"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/lang"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/pages"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/preview"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

func newPadsApp(t *testing.T) (*app.App, *desk.Model) {
	t.Helper()
	model := desk.NewOffline(nil)
	if err := model.LoadRaw(preview.BuildTree()); err != nil {
		t.Fatal(err)
	}
	ctx := ui.NewContext(ui.Dark(ui.Liberation()), nil)
	a := app.New(1280, 720, ctx, model, lang.New("en", nil), nil)
	a.RenderOnce()
	return a, model
}

// TestPadPagesRender builds and draws every pad, show and recording page.
func TestPadPagesRender(t *testing.T) {
	a, _ := newPadsApp(t)
	for _, name := range []string{"options", "colour", "new", "sound", "mixer", "mixer:new", "trigger", "dial", "video", "icecast",
		"fx", "fx:echo", "fx:megaphone", "fx:robot", "fx:disguise", "fx:pitch"} {
		a.Home()
		a.Open(pages.IDPads)
		idx := 3
		switch name {
		case "sound":
			idx = 2
		case "trigger":
			idx = 10
		case "dial":
			idx = 12
		case "video":
			idx = 13
		case "icecast":
			idx = 14
		case "mixer", "mixer:new":
			idx = 4
		case "fx:robot", "fx:disguise", "fx:pitch":
			idx = 7
		}
		a.Push(pages.PadPage(idx, name))
		if got := a.RenderOnce(); len(got) == 0 {
			t.Errorf("pad page %q rendered nothing", name)
		}
	}
	for _, name := range []string{"", "new", "import", "import:0", "export", "icon", "loading", "player"} {
		a.Home()
		a.Open(pages.IDShows)
		a.Push(pages.ShowPage(name))
		if got := a.RenderOnce(); len(got) == 0 {
			t.Errorf("show page %q rendered nothing", name)
		}
	}
	a.Home()
	a.Open(pages.IDRecordings)
	if got := a.RenderOnce(); len(got) == 0 {
		t.Error("recordings rendered nothing")
	}
	// The editor opener swaps itself for the pad's own editor.
	a.Home()
	a.Push(pages.PadEdit(10))
	a.RenderOnce()
	if a.PageTitle() != "MIDI" {
		t.Errorf("editing the MIDI pad opened %q", a.PageTitle())
	}
}

// TestNewPadWritesKindAndName: choosing a type on the new-pad page writes
// padType and RØDE's default name, then opens the editor.
func TestNewPadWritesKindAndName(t *testing.T) {
	a, m := newPadsApp(t)
	a.Open(pages.IDPads)
	a.Push(pages.PadPage(15, "new"))
	a.RenderOnce()
	// The Trigger tile is the fourth of five tiles across the page.
	tap(a, 870, 330)
	a.RenderOnce()
	pad := m.PadByIndex(15)
	if pad.Type != desk.PadTrigger || pad.Name != "Trigger 16" {
		t.Fatalf("after the Trigger tile: type %d name %q", pad.Type, pad.Name)
	}
	if a.PageTitle() != "MIDI" {
		t.Fatalf("the editor did not open: %q", a.PageTitle())
	}
	// RØDE's trigger page sets the mode to latching when it opens.
	if m.Int(desk.PadNode, pad.Node, "padTriggerMode") != 1 {
		t.Fatalf("padTriggerMode = %d, want 1", m.Int(desk.PadNode, pad.Node, "padTriggerMode"))
	}
}

// TestMixerModeAndTrigger: the mixer page writes the mode index and the
// trigger pair; the trigger page cycles the send mode.
func TestMixerModeAndTrigger(t *testing.T) {
	a, m := newPadsApp(t)
	a.Open(pages.IDPads)
	a.Push(pages.PadPage(4, "mixer"))
	a.RenderOnce()
	node := m.PadByIndex(4).Node
	// The Momentary half of the header pair.
	tap(a, 900, 60)
	a.RenderOnce()
	if got := m.Int(desk.PadNode, node, "padMixerTriggerMode"); got != 0 {
		t.Fatalf("padMixerTriggerMode = %d, want 0 (momentary)", got)
	}
	tap(a, 380, 60)
	a.RenderOnce()
	if got := m.Int(desk.PadNode, node, "padMixerTriggerMode"); got != 1 {
		t.Fatalf("padMixerTriggerMode = %d, want 1 (latching)", got)
	}
	// The next tile to the right of the chosen one is Trash Talk.
	tap(a, 980, 240)
	a.RenderOnce()
	if got := m.Int(desk.PadNode, node, "padMixerMode"); got != 1 {
		t.Fatalf("padMixerMode = %d, want 1 (trash talk)", got)
	}

	a.Home()
	a.Open(pages.IDPads)
	a.Push(pages.PadPage(10, "trigger"))
	a.RenderOnce()
	node = m.PadByIndex(10).Node
	m.SetBool(desk.PadNode, node, "padTriggerCustom", true)
	a.RenderOnce()
	tap(a, 1020, 580) // the Send button
	a.RenderOnce()
	if got := m.Int(desk.PadNode, node, "padTriggerSend"); got != 0 {
		t.Fatalf("padTriggerSend after one tap = %d, want 0 (press)", got)
	}
}

// TestSoundPadPlayMode: the play-mode carousel writes 0 Toggle, 1 Hold, 2
// One Shot, and Hold switches the loop off and replay on.
func TestSoundPadPlayMode(t *testing.T) {
	a, m := newPadsApp(t)
	a.Open(pages.IDPads)
	a.Push(pages.PadPage(2, "sound"))
	a.RenderOnce()
	node := m.PadByIndex(2).Node
	if !m.Bool(desk.PadNode, node, "padLoop") {
		t.Fatal("the fixture pad 2 loops")
	}
	tap(a, 390, 486) // the play-mode carousel's ›
	a.RenderOnce()
	if got := m.Int(desk.PadNode, node, "padPlayMode"); got != 1 {
		t.Fatalf("padPlayMode = %d, want 1 (hold)", got)
	}
	if m.Bool(desk.PadNode, node, "padLoop") || !m.Bool(desk.PadNode, node, "padReplay") {
		t.Fatal("hold should switch the loop off and replay on")
	}
	tap(a, 390, 486)
	a.RenderOnce()
	if got := m.Int(desk.PadNode, node, "padPlayMode"); got != 2 {
		t.Fatalf("padPlayMode = %d, want 2 (one shot)", got)
	}
}
