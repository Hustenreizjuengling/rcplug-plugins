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
	"path/filepath"
	"strconv"
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
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/web"
	"github.com/Hustenreizjuengling/rcplug/sdk"
)

// version is the plugin's own (its plugin.json), release the rcplug-plugins
// release it came with; both are set at build time.
var version, release = "dev", ""

// The panel is turned against the page as RØDE's screen program draws it: the
// page's top on the panel's right (the Duo), which is a 90° clockwise turn.
const panelTurn = 90

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
		if err := runDevice(ctx, *card, *touchDev, *guiPath, *langCode); err != nil && !errors.Is(err, context.Canceled) {
			log.Fatalf("screen: %v", err)
		}
		log.Printf("screen: done")
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

// runDevice is the real plugin: the desk model and the browser interface
// run for as long as the plugin does; the device's screen is taken and given
// back on request (the host's marker says who has it).
func runDevice(ctx context.Context, card, touchDev, guiPath, langCode string) error {
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
	table, _ := res.Translations(rode.Language(code))
	texts := lang.New(code, table)

	ctxUI := ui.NewContext(ui.Dark(ui.Liberation()), rode.NewIcons(res))
	reg := ext.New("", sdk.Name(), log.Printf)
	reg.SetLang(code)
	go reg.Run(ctx)
	go pages.WarmMicPictures(ctxUI) // the icon cache is safe to fill from here

	// Follow the desk's language: a start before the desk had loaded began
	// in English, and the language page changes it while running.
	var loader func(code string) *lang.Texts
	if langCode == "" {
		loader = func(code string) *lang.Texts {
			table, _ := res.Translations(rode.Language(code))
			reg.SetLang(code)
			log.Printf("screen: language %s (%d texts)", code, len(table))
			return lang.New(code, table)
		}
	}
	current := func() *lang.Texts { return texts }

	// The browser interface, on the port the host gave us.
	if addr := sdk.ListenAddr(""); addr != "" {
		srv := web.New(ctxUI, model, current, loader, reg, log.Printf)
		go func() {
			if err := sdk.Serve(ctx, addr, srv.Handler()); err != nil && !errors.Is(err, context.Canceled) {
				log.Printf("screen: web: %v", err)
			}
		}()
		log.Printf("screen: browser interface on %s", addr)
	}
	log.Printf("screen: up (%s, %d texts, %d strips)", code, texts.Count(), model.StripCount())

	// The device's screen, whenever the host's marker names us.
	pc := &panelController{card: card, ctxUI: ctxUI, model: model, texts: current, loader: loader, reg: reg}
	pc.run(ctx)
	return nil
}

// panelController opens the device's screen while the host's marker names
// this plugin and closes it when the marker changes (a handover, a release)
// or "Back to RØDE" asks for it.
type panelController struct {
	card   string
	ctxUI  *ui.Context
	model  *desk.Model
	texts  func() *lang.Texts
	loader func(string) *lang.Texts
	reg    *ext.Registry
}

// rodeScreenRunning reports whether RØDE's screen program runs (the device
// has finished starting its display).
func rodeScreenRunning() bool {
	dirs, _ := filepath.Glob("/proc/[0-9]*/comm")
	for _, d := range dirs {
		if b, err := os.ReadFile(d); err == nil && strings.TrimSpace(string(b)) == "rc_gui_juce" {
			return true
		}
	}
	return false
}

func (pc *panelController) run(ctx context.Context) {
	self := sdk.Name()
	var (
		cancel  context.CancelFunc
		done    chan struct{}
		retryAt time.Time
		askedAt time.Time
	)
	started := time.Now()
	tick := time.NewTicker(300 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			if cancel != nil {
				cancel()
				<-done
			}
			return
		case <-tick.C:
		}
		holder := ext.ScreenHolder()
		held := self != "" && holder == self
		running := done != nil
		// Nobody has the screen but RØDE, and the interface belongs on it:
		// ask the host (once RØDE's program is up, as the host does at
		// boot; again every half minute if the host said no).
		if !held && holder == "" && pc.reg != nil && ext.PanelWish() && time.Since(started) > 3*time.Second &&
			time.Since(askedAt) > 30*time.Second && rodeScreenRunning() {
			askedAt = time.Now()
			go func() {
				if err := pc.reg.TakeScreen(); err != nil {
					log.Printf("screen: asking for the device's screen: %v", err)
				}
			}()
		}
		if running {
			select {
			case <-done:
				done, cancel = nil, nil
				running = false
			default:
			}
		}
		switch {
		case held && !running && time.Now().After(retryAt):
			pctx, c := context.WithCancel(ctx)
			d := make(chan struct{})
			cancel, done = c, d
			go func() {
				defer close(d)
				if err := pc.once(pctx); err != nil && !errors.Is(err, context.Canceled) {
					log.Printf("screen: panel: %v", err)
					retryAt = time.Now().Add(3 * time.Second)
				}
			}()
		case !held && running:
			cancel()
			<-done
			done, cancel = nil, nil
			log.Printf("screen: the device's screen went to %q; the panel is closed", ext.ScreenHolder())
		}
	}
}

// once runs the interface on the device's screen until the marker moves on,
// the plugin ends or "Back to RØDE" is pressed (then the screen is handed
// back after the panel closed, so RØDE's program finds the display free).
func (pc *panelController) once(ctx context.Context) error {
	defer panel.Light(log.Printf)()
	scr, err := openScreenWait(ctx, pc.card, log.Printf)
	if err != nil {
		return err
	}
	w, h := scr.Size()
	tctx, tcancel := context.WithCancel(ctx)
	defer tcancel()
	touches := make(chan panel.TouchEvent, 32)
	panelW, panelH := h, w
	go panel.ReadTouch(tctx, panelW, panelH, touches, log.Printf)

	a := app.New(w, h, pc.ctxUI, pc.model, pc.texts(), pc.reg)
	if pc.loader != nil {
		a.FollowLanguage(pc.loader)
	}
	release := false
	a.OnRode = func() { release = true; a.Quit() }
	ext.SetPanelWish(true)
	// A desk nobody has set up yet starts with the welcome flow and the
	// setup assistant, as RØDE's screen does on its first start.
	if firstRun() {
		// The marker means "shown once": leaving the flow any other way
		// (the home button, a channel button) must not bring it back at
		// the next start.
		markWelcomeDone()
		a.Push(pages.NewWelcome(func() { a.Home() }))
		log.Printf("screen: first start, showing the welcome flow")
	}
	log.Printf("screen: on the device's screen (%d strips)", pc.model.StripCount())
	err = a.Run(ctx, scr, touches, panelTurn)
	scr.Close() // black, and the display free before RØDE's program starts
	if release {
		log.Printf("screen: back to RØDE")
		if err := pc.reg.ReleaseScreen(); err != nil {
			log.Printf("screen: giving the screen back: %v", err)
		}
		return nil
	}
	return err
}

// runServe shows the interface in a browser over a representative state.
func runServe(ctx context.Context, addr, resources, langCode string) error {
	a, model, err := offlineApp(resources, langCode)
	if err != nil {
		return err
	}
	srv := web.New(a.Ctx(), model, a.Texts, nil, preview.NewPlugins(), log.Printf)
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
	case page == "", page == "home":
	case page == "channel":
		a.Push(pages.NewChannel(0))
	case strings.HasPrefix(page, "channel:"):
		n, _ := strconv.Atoi(strings.TrimPrefix(page, "channel:"))
		a.Push(pages.NewChannel(n))
	case strings.HasPrefix(page, "channelfx:"):
		n, _ := strconv.Atoi(strings.TrimPrefix(page, "channelfx:"))
		a.Push(pages.NewChannelTab(n, 1))
	case strings.HasPrefix(page, "proc:") && strings.Contains(page, ":") && strings.Count(page, ":") == 2:
		parts := strings.Split(page, ":")
		n, _ := strconv.Atoi(parts[1])
		a.Push(pages.Processor(n, parts[2]))
	case page == "wireless-info":
		a.Push(pages.NewChannel(6))
		a.Push(pages.WirelessInfo())
	case strings.HasPrefix(page, "fx:"):
		parts := strings.Split(strings.TrimPrefix(page, "fx:"), ":")
		n, _ := strconv.Atoi(parts[0])
		idx := 0
		if len(parts) > 1 {
			idx, _ = strconv.Atoi(parts[1])
		}
		a.Push(pages.NewEffectsRack(n, idx))
	case page == "assign":
		a.Push(pages.NewAssign(0))
	case strings.HasPrefix(page, "proc:"):
		a.Push(pages.Processor(0, strings.TrimPrefix(page, "proc:")))
	case strings.HasPrefix(page, "presets:"):
		n, _ := strconv.Atoi(strings.TrimPrefix(page, "presets:"))
		a.Push(pages.NewPresets(n))
	case page == "master":
		a.Push(pages.NewMasterRack())
	case page == "fx":
		a.Push(pages.NewEffectsRack(0, 0))
	case strings.HasPrefix(page, "plugin:"):
		a.Open(pages.IDRcplug)
		a.Open(pages.IDPlugins)
		n := strings.TrimPrefix(page, "plugin:")
		a.Push(pages.PluginPageFor(n, strings.ToUpper(n[:1])+n[1:]))
	case page == "headphones":
		a.ShowHeadphone(0)
	case page == "transfer":
		a.Push(pages.NewTransfer())
	case page == "bluetooth-out":
		a.Push(pages.Detail("bluetooth"))
	case page == "welcome" || strings.HasPrefix(page, "welcome:"):
		n, _ := strconv.Atoi(strings.TrimPrefix(page, "welcome:"))
		w := pages.NewWelcome(nil).(*pages.WelcomePage)
		w.SetStep(n)
		a.Push(w)
	case page == "wizard" || strings.HasPrefix(page, "wizard:"):
		n, _ := strconv.Atoi(strings.TrimPrefix(page, "wizard:"))
		w := pages.NewSetupAssistant(nil).(*pages.SetupAssistantPage)
		w.SetStep(n)
		a.Push(w)
	case strings.HasPrefix(page, "callme:"):
		// callme:<hosting|activate|advanced|dial|quickdial|quickdial-saved|sipdial|fader|assign>
		a.Open(pages.IDCallMe)
		a.Push(pages.CallMePage(strings.TrimPrefix(page, "callme:")))
	case strings.HasPrefix(page, "popup:"):
		// popup:<incoming|conflict|recording|welcome|rating|timelimit>
		a.Open(pages.IDCallMe)
		a.ShowCallMePopup(strings.TrimPrefix(page, "popup:"))
	case page == "network-join", page == "network-advanced", page == "network-dns", page == "network-proxy":
		a.Open(pages.IDNetwork)
		a.Push(pages.CallMePage(page))
	case page == "keyboard":
		a.Keyboard("Name", "Mic 1", nil)
	case page == "password":
		a.Password("Enter password for Studio", nil)
	case page == "dialog":
		a.Confirm("CLEAR", "Are you sure you want to clear the pad?", "CLEAR", true, nil)
	case page == "metering", page == "listen", page == "faders", page == "language", page == "datetime", page == "brightness", page == "haptics", page == "monitor", page == "multitrack",
		page == "headphones-page", page == "recbutton", page == "compression", page == "midi", page == "info", page == "info-device", page == "regulatory",
		page == "analytics", page == "firmware-page", page == "opmode", page == "storage", page == "timezone", page == "cloud", page == "numpad":
		a.Push(pages.Detail(strings.TrimSuffix(page, "-page")))
	case page == "mix":
		a.Open(pages.IDRouting)
		a.Push(pages.RoutingMix(0))
	case strings.HasPrefix(page, "mix:"):
		n, _ := strconv.Atoi(strings.TrimPrefix(page, "mix:"))
		a.Open(pages.IDRouting)
		a.Push(pages.RoutingMix(n))
	case strings.HasPrefix(page, "pad:"):
		// pad:<idx>[:options|new|colour|mixer:<mode>|fx:<effect>]
		parts := strings.Split(strings.TrimPrefix(page, "pad:"), ":")
		n, _ := strconv.Atoi(parts[0])
		a.Open(pages.IDPads)
		if len(parts) > 1 {
			a.Push(pages.PadPage(n, strings.Join(parts[1:], ":")))
		} else {
			a.Push(pages.PadEdit(n))
			a.RenderOnce() // the opener swaps in the editor
		}
	case strings.HasPrefix(page, "show:"), page == "player", page == "storage":
		// show:new|import|import:<n>|export|icon|loading, player, storage
		a.Open(pages.IDShows)
		a.Push(pages.ShowPage(strings.TrimPrefix(page, "show:")))
		a.RenderOnce()
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
	a := app.New(1280, 720, ctxUI, model, texts, preview.NewPlugins())
	return a, model, nil
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
