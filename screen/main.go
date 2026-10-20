// Command screen is an rcplug plugin that puts an alternative interface on
// the RØDECaster's touchscreen: a mixer, channel and processing pages, pads,
// the menus and settings, and pages for rcplug and other plugins — built to
// look and work like RØDE's own screen, and open to extension.
//
// It draws with the kernel's DRM interface (package panel), reads the
// touchscreen through evdev and controls the desk through the console plugin
// (package desk). It ships no RØDE assets: the icons and text are read from
// the device's own screen program at run time (package rode), and the font
// is Liberation Sans (OFL, the font the device uses).
//
// With --serve it runs in a web browser against a representative state, and
// with --png it draws one frame to a file, both for development without a
// device (the icons and text then come from a folder given with
// --resources).
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"image/png"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/app"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ext"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/lang"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/pages"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/panel"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/preview"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/rode"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
	"github.com/Hustenreizjuengling/rcplug/sdk"
)

// version is the plugin's own (its plugin.json), release the rcplug-plugins
// release it came with; both are set at build time.
var version, release = "dev", ""

// The panel is turned against the page as RØDE's screen program draws it: the
// page's top on the panel's right (the Duo), which is a 90° clockwise turn.
const panelTurn = 90

// screenMarker is where rcplugd writes the name of the plugin it gave the
// screen to (rcplug 0.8.11 and later).
const screenMarker = "/run/rcplug-screen"

func main() {
	var (
		pngOut    = flag.String("png", "", "draw one frame into this PNG file and exit (development)")
		pngPage   = flag.String("page", "", "with --png: navigate to this page first (settings, outputs, display, system, pads, routing, rcplug, channel)")
		serve     = flag.String("serve", "", "run the interface in a web browser on this address, e.g. 127.0.0.1:8099 (development)")
		resources = flag.String("resources", "", "folder of the device's extracted icons and text, for --png/--serve (development)")
		card      = flag.String("card", "/dev/dri/card0", "the display's DRM device")
		touchDev  = flag.String("touch", "", "the touchscreen's event device (default: found by its name)")
		langCode  = flag.String("lang", "", "language: en | de | … (default: the device's)")
		guiPath   = flag.String("rode", rode.GUIProgram, "RØDE's screen program, read for its icons and text")
		noCheck   = flag.Bool("no-handover-check", false, "skip the check that rcplug handed the screen over (development)")
		showVer   = flag.Bool("version", false, "print the version and exit")
	)
	flag.Parse()
	if *showVer {
		fmt.Println(version)
		return
	}
	log.SetFlags(0)
	ctx, stop := sdk.SignalContext()
	defer stop()

	switch {
	case *pngOut != "":
		if err := runPNG(*pngOut, *resources, *langCode, *pngPage); err != nil {
			log.Fatalf("screen: %v", err)
		}
	case *serve != "":
		if err := runServe(ctx, *serve, *resources, *langCode); err != nil && !errors.Is(err, context.Canceled) {
			log.Fatalf("screen: %v", err)
		}
	default:
		if err := runDevice(ctx, *card, *touchDev, *guiPath, *langCode, *noCheck); err != nil && !errors.Is(err, context.Canceled) {
			log.Fatalf("screen: %v; RØDE's screen comes back", err)
		}
		log.Printf("screen: done; RØDE's screen comes back")
	}
}

// resolveLang picks the language: the flag, else the device's GUI setting,
// else English.
func resolveLang(flagVal, deskVal string) string {
	if flagVal != "" {
		return flagVal
	}
	if deskVal != "" {
		return deskVal
	}
	return "en"
}

// langFile is the resource file for a language code.
func langFile(code string) string {
	m := map[string]string{
		"en": "english.txt", "de": "german.txt", "es": "spanish.txt", "fr": "french.txt",
		"it": "italian.txt", "pt": "portuguese.txt", "ru": "russian.txt", "ja": "japanese.txt",
		"zh": "chinese.txt", "ko": "korean.txt", "hi": "hindi.txt",
	}
	if f, ok := m[code]; ok {
		return f
	}
	return "english.txt"
}

// runDevice is the real plugin: take the screen and run until stopped.
func runDevice(ctx context.Context, card, touchDev, guiPath, langCode string, noCheck bool) error {
	if !noCheck {
		if err := handoverOK(sdk.Name()); err != nil {
			return err
		}
	}
	res, err := rode.Open(guiPath)
	if err != nil {
		return fmt.Errorf("reading the device's icons and text from %s: %w", guiPath, err)
	}
	defer res.Close()

	model := desk.New("", log.Printf)
	go model.Run(ctx)
	// Wait briefly for the first load so the first frame is not empty.
	waitOnline(ctx, model, 3*time.Second)

	code := langCode
	if code == "" {
		code = model.Str(desk.GUINode, 0, "lang")
	}
	code = resolveLang(langCode, code)
	table, _ := res.Translations(code)
	texts := lang.New(code, table)

	ctxUI := ui.NewContext(ui.Dark(ui.Liberation()), rode.NewIcons(res))
	reg := ext.New("", sdk.Name(), log.Printf)
	go reg.Run(ctx)

	defer panel.Light(log.Printf)()
	scr, err := openScreenWait(ctx, card, log.Printf)
	if err != nil {
		return err
	}
	defer scr.Close()
	w, h := scr.Size()

	touches := make(chan panel.TouchEvent, 32)
	panelW, panelH := h, w
	go panel.ReadTouch(ctx, panelW, panelH, touches, log.Printf)

	a := app.New(w, h, ctxUI, model, texts, reg)
	log.Printf("screen: interface up (%s, %d texts, %d strips)", code, texts.Count(), model.StripCount())
	return a.Run(ctx, scr, touches, panelTurn)
}

// runServe shows the interface in a browser over a representative state.
func runServe(ctx context.Context, addr, resources, langCode string) error {
	a, _, err := offlineApp(resources, langCode)
	if err != nil {
		return err
	}
	srv := preview.NewServer(a)
	log.Printf("screen: preview at http://%s/", addr)
	return sdk.Serve(ctx, addr, srv.Handler())
}

// runPNG draws one frame of the interface to a file.
func runPNG(out, resources, langCode, page string) error {
	a, _, err := offlineApp(resources, langCode)
	if err != nil {
		return err
	}
	switch {
	case page == "":
	case page == "channel":
		a.Push(pages.NewChannel(0))
	case strings.HasPrefix(page, "proc:"):
		a.Push(pages.Processor(0, strings.TrimPrefix(page, "proc:")))
	case page == "keyboard":
		a.Keyboard("Name", "Mic 1", nil)
	case page == "dialog":
		a.Confirm("CLEAR", "Are you sure you want to clear the pad?", "CLEAR", true, nil)
	default:
		a.Open(page)
	}
	a.RenderOnce()
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, a.Root().Image().ToNRGBA())
}

// offlineApp builds an app over the offline model and the resources folder
// (for --png and --serve).
func offlineApp(resources, langCode string) (*app.App, *desk.Model, error) {
	if resources == "" {
		return nil, nil, errors.New("--resources must point to a folder of the device's extracted icons and text")
	}
	if _, err := os.Stat(resources); err != nil {
		return nil, nil, fmt.Errorf("--resources: %w", err)
	}
	code := resolveLang(langCode, "de")
	icons := preview.NewDirIcons(resources)
	table := preview.LoadTexts(resources, langFile(code))
	texts := lang.New(code, table)
	ctxUI := ui.NewContext(ui.Dark(ui.Liberation()), icons)

	model := desk.NewOffline(log.Printf)
	if err := model.LoadRaw(preview.BuildTree()); err != nil {
		return nil, nil, err
	}
	a := app.New(1280, 720, ctxUI, model, texts, nil)
	return a, model, nil
}

// handoverOK checks that rcplug stopped RØDE's screen for this plugin.
func handoverOK(name string) error {
	b, err := os.ReadFile(screenMarker)
	if err != nil || name == "" || strings.TrimSpace(string(b)) != name {
		return errors.New("rcplug did not hand the screen over (this plugin needs rcplug 0.8.11 or later and the \"screen\" permission)")
	}
	return nil
}

// openScreenWait takes the display, giving RØDE's screen program a moment to
// let go of it after it was stopped.
func openScreenWait(ctx context.Context, card string, logf func(string, ...interface{})) (panel.Output, error) {
	var lastErr error
	for wait := 5 * time.Second; wait > 0; wait -= 250 * time.Millisecond {
		scr, err := panel.OpenScreen(card, panelTurn, logf)
		if err == nil {
			return scr, nil
		}
		lastErr = err
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
	return nil, fmt.Errorf("opening the display: %w", lastErr)
}

// waitOnline waits up to d for the model's first load.
func waitOnline(ctx context.Context, model *desk.Model, d time.Duration) {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if model.Online() {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(100 * time.Millisecond):
		}
	}
}
