package app_test

import (
	"image"
	"testing"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/app"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/lang"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/preview"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// TestTapCogOpensSettings taps the settings cog on the mixer and checks the
// navigation moved to the settings page, so touch routing through the shell
// and the full-bleed page works.
func TestTapCogOpensSettings(t *testing.T) {
	model := desk.NewOffline(nil)
	if err := model.LoadRaw(preview.BuildTree()); err != nil {
		t.Fatal(err)
	}
	ctx := ui.NewContext(ui.Dark(ui.Liberation()), nil)
	a := app.New(1280, 720, ctx, model, lang.New("de", nil), nil)
	a.RenderOnce() // build the mixer

	if a.Depth() != 1 {
		t.Fatalf("start depth %d, want 1", a.Depth())
	}
	// The cog is at the right end of the status bar.
	tap(a, 1244, 60)
	a.RenderOnce()
	if a.Depth() != 2 {
		t.Fatalf("after tapping the cog, depth %d, want 2", a.Depth())
	}
	if got := a.PageTitle(); got != "CONFIGURATION" {
		t.Fatalf("page title %q, want CONFIGURATION", got)
	}

	// Tapping Home returns to the mixer.
	tap(a, 1214, 58) // the home button on the header
	a.RenderOnce()
	if a.Depth() != 1 {
		t.Fatalf("after Home, depth %d, want 1", a.Depth())
	}
}

func tap(a *app.App, x, y int) {
	a.Touch(ui.Down, image.Pt(x, y))
	a.Touch(ui.Up, image.Pt(x, y))
}
