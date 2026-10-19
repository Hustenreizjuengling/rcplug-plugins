package ui_test

import (
	"image"
	"image/png"
	"os"
	"testing"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/rode"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// TestPreview draws a demo page with the real icons for looking at:
// RCPLUG_RODE_GUI=<rc_gui_juce> RCPLUG_UI_OUT=<out.png>.
func TestPreview(t *testing.T) {
	guiPath, out := os.Getenv("RCPLUG_RODE_GUI"), os.Getenv("RCPLUG_UI_OUT")
	if guiPath == "" || out == "" {
		t.Skip("set RCPLUG_RODE_GUI and RCPLUG_UI_OUT")
	}
	res, err := rode.Open(guiPath)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Close()
	ctx := ui.NewContext(ui.Dark(ui.Liberation()), rode.NewIcons(res))
	root := ui.NewRoot(ctx, 1280, 720)
	th := ctx.Theme

	// A demo layout: a top bar, some strips with meters and faders, a row
	// of buttons, knobs and toggles.
	page := &ui.Container{}
	bar := &ui.Container{}
	bar.SetBounds(image.Rect(0, 0, 1280, 64))
	ui.Panel(root.Image(), image.Rect(0, 0, 0, 0), 0, th.Bg, image.Rect(0, 0, 0, 0))
	title := ui.NewLabel("MISCHPULT", th.Fonts.Title, th.Text, font.Left)
	title.SetBounds(image.Rect(16, 8, 400, 56))
	rec := ui.NewButton("REC", nil)
	rec.Icon, rec.On = "IconRecOn", true
	rec.SetBounds(image.Rect(1120, 12, 1264, 52))
	bar.Add(title, rec)

	strips := &ui.Container{}
	strips.SetBounds(image.Rect(0, 72, 1280, 520))
	levels := []float64{0.8, 0.55, 0.3, 0.72, 0.6, 0.2}
	for i := 0; i < 6; i++ {
		x := 16 + i*150
		sr := image.Rect(x, 80, x+140, 512)
		card := ui.NewPanelView(th.Panel)
		card.SetBounds(sr)
		name := ui.NewLabel([]string{"MIC 1", "MIC 2", "USB", "BLUETOOTH", "SOUND", "CHAT"}[i], th.Fonts.Small, th.Text, font.Center)
		name.SetBounds(image.Rect(x, 88, x+140, 118))
		lvl := levels[i]
		m := ui.NewMeter(func() float64 { return lvl })
		m.SetBounds(image.Rect(x+12, 130, x+40, 420))
		fader := ui.NewFader(func() float64 { return lvl }, nil)
		fader.SetBounds(image.Rect(x+50, 130, x+128, 460))
		mute := ui.NewButton("", nil)
		mute.Icon, mute.Toggle, mute.On = "IconMute", true, i == 2
		mute.SetBounds(image.Rect(x+12, 466, x+68, 504))
		listen := ui.NewButton("", nil)
		listen.Icon = "IconListen"
		listen.SetBounds(image.Rect(x+76, 466, x+128, 504))
		strips.Add(card, name, m, fader, mute, listen)
	}

	controls := &ui.Container{}
	controls.SetBounds(image.Rect(0, 528, 1280, 720))
	k1 := ui.NewKnob(func() float64 { return 0.65 }, nil)
	k1.Label, k1.Text = "GAIN", func(v float64) string { return "+24" }
	k1.SetBounds(image.Rect(16, 536, 136, 690))
	k2 := ui.NewKnob(func() float64 { return 0.4 }, nil)
	k2.Label, k2.Text = "COMP", func(v float64) string { return "-6 dB" }
	k2.SetBounds(image.Rect(150, 536, 270, 690))
	tog := ui.NewToggle(func() bool { return true }, nil)
	tog.SetBounds(image.Rect(300, 560, 440, 600))
	togLbl := ui.NewLabel("48V", th.Fonts.Body, th.Text, font.Left)
	togLbl.SetBounds(image.Rect(300, 610, 440, 650))
	b1 := ui.NewButton("Einstellungen", nil)
	b1.Icon = "IconSystem"
	b1.SetBounds(image.Rect(470, 560, 760, 616))
	b2 := ui.NewButton("Routing", nil)
	b2.Icon = "IconRouting"
	b2.SetBounds(image.Rect(470, 630, 760, 686))
	controls.Add(k1, k2, tog, togLbl, b1, b2)

	page.Add(bar, strips, controls)
	root.SetView(page)
	root.Render()

	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, root.Image().ToNRGBA()); err != nil {
		t.Fatal(err)
	}
	_ = gfx.Hex
}
