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

// TestPluginPageControls: a plugin's declared page renders, the knob walks
// its controls, and a chosen option and a flipped toggle reach the plugin
// as the protocol says (the option's index, the toggle's state).
func TestPluginPageControls(t *testing.T) {
	model := desk.NewOffline(nil)
	if err := model.LoadRaw(preview.BuildTree()); err != nil {
		t.Fatal(err)
	}
	plugins := preview.NewPlugins()
	ctx := ui.NewContext(ui.Dark(ui.Liberation()), nil)
	a := app.New(1280, 720, ctx, model, lang.New("de", nil), plugins)
	a.Push(pages.PluginPageFor("sonar", "Sonar"))
	a.RenderOnce()

	// Walk with the knob to the second option of the choice ("USB 1 …")
	// and press. Each turn moves one control: the header's back and home
	// come first, then the option rows (value and meter rows take no
	// focus), the slider, the toggle and the buttons.
	for i := 0; i < 4; i++ {
		a.Encoder(desk.EncoderEvent{Delta: 1})
	}
	a.Encoder(desk.EncoderEvent{Release: true})
	a.RenderOnce()
	if n := len(plugins.Submitted); n != 1 || plugins.Submitted[0].Control != "listen" || plugins.Submitted[0].Value != float64(1) {
		t.Fatalf("submissions after choosing an option: %+v", plugins.Submitted)
	}
	// On to the toggle: two more options, the slider, then the toggle.
	for i := 0; i < 4; i++ {
		a.Encoder(desk.EncoderEvent{Delta: 1})
	}
	a.Encoder(desk.EncoderEvent{Release: true})
	if n := len(plugins.Submitted); n != 2 || plugins.Submitted[1].Control != "flip" || plugins.Submitted[1].Value != true {
		t.Fatalf("submissions after the toggle: %+v", plugins.Submitted)
	}
	if a.Depth() != 2 {
		t.Fatalf("depth %d, want 2", a.Depth())
	}
}
