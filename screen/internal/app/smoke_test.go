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

// TestRenderAllPages builds and draws every page over a representative state,
// with no icons or text (so it needs no device resources), to catch layout
// panics and make sure a frame comes out.
func TestRenderAllPages(t *testing.T) {
	model := desk.NewOffline(nil)
	if err := model.LoadRaw(preview.BuildTree()); err != nil {
		t.Fatal(err)
	}
	ctx := ui.NewContext(ui.Dark(ui.Liberation()), nil) // nil icons: drawn as nothing
	texts := lang.New("de", nil)
	a := app.New(1280, 720, ctx, model, texts, nil)

	ids := []string{
		pages.IDSettings, pages.IDOutputs, pages.IDDisplay, pages.IDSystem,
		pages.IDRouting, pages.IDPads, pages.IDShows, pages.IDRecordings,
		pages.IDNetwork, pages.IDBluetooth, pages.IDCallMe, pages.IDInfo,
		pages.IDFirmware, pages.IDRcplug, pages.IDPlugins,
	}

	// Home first.
	if got := a.RenderOnce(); len(got) == 0 {
		t.Fatal("home rendered nothing")
	}
	// A channel page and a processor.
	a.Push(pages.NewChannel(0))
	a.RenderOnce()
	a.Push(pages.NewAssign(0))
	a.RenderOnce()

	for _, id := range ids {
		a.Home()
		a.Open(id)
		if got := a.RenderOnce(); len(got) == 0 {
			t.Errorf("page %q rendered nothing", id)
		}
	}
}

// TestRenderRcplugPages builds and draws the rcplug pages over the
// preview's host data (the system, processes, plugins, updates, catalog
// and services), so a layout panic or an empty frame shows up here.
func TestRenderRcplugPages(t *testing.T) {
	model := desk.NewOffline(nil)
	if err := model.LoadRaw(preview.BuildTree()); err != nil {
		t.Fatal(err)
	}
	ctx := ui.NewContext(ui.Dark(ui.Liberation()), nil)
	for _, code := range []string{"de", "en"} {
		a := app.New(1280, 720, ctx, model, lang.New(code, nil), preview.NewPlugins())
		a.Open(pages.IDRcplug)
		if got := a.RenderOnce(); len(got) == 0 {
			t.Fatal("rcplug menu rendered nothing")
		}
		a.Open(pages.IDPlugins)
		if got := a.RenderOnce(); len(got) == 0 {
			t.Fatal("plugin list rendered nothing")
		}
		for _, kind := range []string{"overview", "procs", "updates", "catalog", "services", "plugin:sonar", "plugin:screen", "plugin:plugrain"} {
			a.Open(pages.IDRcplug)
			a.Push(pages.RcplugSub(kind))
			if got := a.RenderOnce(); len(got) == 0 {
				t.Errorf("%s: rcplug page %q rendered nothing", code, kind)
			}
			// A second frame reads the live values again.
			a.RenderOnce()
		}
	}
}

// TestDialogAndKeyboard checks the modal overlays build and render.
func TestDialogAndKeyboard(t *testing.T) {
	model := desk.NewOffline(nil)
	_ = model.LoadRaw(preview.BuildTree())
	ctx := ui.NewContext(ui.Dark(ui.Liberation()), nil)
	a := app.New(1280, 720, ctx, model, lang.New("en", nil), nil)
	a.Confirm("Title", "Message body", "OK", true, func() {})
	if got := a.RenderOnce(); len(got) == 0 {
		t.Error("dialog rendered nothing")
	}
	a.Keyboard("Name", "hi", func(string) {})
	if got := a.RenderOnce(); len(got) == 0 {
		t.Error("keyboard rendered nothing")
	}
}
