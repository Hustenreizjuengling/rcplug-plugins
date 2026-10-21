// Package app ties the screen interface together: it owns the drawing root,
// the desk model and the navigation stack, frames each page with a header
// and handles the modal keyboard and dialogs, drives the main knob's focus,
// and runs the draw-and-touch loop against an output and a touch source.
package app

import (
	"image"
	"sync"
	"time"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/lang"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/pages"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// The header bar of a menu page and the content area under it, as on the
// device: square back and home buttons at the corners, the title between.
const (
	headerH    = 116
	contentTop = 128
	sidePad    = 30
	bottomPad  = 20
)

// App is the running interface.
type App struct {
	ctx     *ui.Context
	model   *desk.Model
	text    *lang.Texts
	root    *ui.Root
	plugins pages.PluginSource
	w, h    int

	mu         sync.Mutex
	stack      []pages.Page
	overlay    ui.View
	toast      string
	toastUntil time.Time
	popup      ui.View // a passing view over the page (the headphone level)
	popupUntil time.Time
	diskAsked  int       // the volume the unrecognised-disk question was asked for (+1), 0 for none
	hpLast     []float64 // the headphone levels as last seen
	callme     pages.CallMeWatch
	dirty      bool // the view tree must be rebuilt
	quit       func()

	// textLoader builds the texts of a language code; set, the interface
	// follows the desk's language setting (GUI.lang) while it runs.
	textLoader func(code string) *lang.Texts

	focus focus
}

// FollowLanguage makes the interface follow the desk's language setting:
// when GUI.lang differs from the texts' language, load builds the new texts
// and the pages are rebuilt. That covers a start before the desk had loaded
// (the texts began as English) and a change on the language page, which
// RØDE's screen also applies at once.
func (a *App) FollowLanguage(load func(code string) *lang.Texts) {
	a.textLoader = load
}

// followLanguage switches the texts when the desk's language changed.
func (a *App) followLanguage() {
	if a.textLoader == nil || !a.model.Online() {
		return
	}
	code := a.model.Str(desk.GUINode, 0, "lang")
	if code == "" || code == a.text.Code() {
		return
	}
	if t := a.textLoader(code); t != nil {
		a.text = t
		a.Rebuild()
	}
}

// Texts are the interface's current texts.
func (a *App) Texts() *lang.Texts { return a.text }

// New makes an app for a w × h screen.
func New(w, h int, ctx *ui.Context, model *desk.Model, text *lang.Texts, plugins pages.PluginSource) *App {
	a := &App{ctx: ctx, model: model, text: text, plugins: plugins, w: w, h: h}
	a.root = ui.NewRoot(ctx, w, h)
	a.stack = []pages.Page{pages.NewHome()}
	a.dirty = true
	a.focus.idx = -1
	ui.GrabFocus = a.focus.grab
	return a
}

// env builds the page environment with this app as the navigator.
func (a *App) env() *pages.Env {
	return &pages.Env{Ctx: a.ctx, Desk: a.model, Text: a.text, Nav: a, Plugins: a.plugins}
}

// ShowHeadphone shows the headphone popup (for the preview).
func (a *App) ShowHeadphone(i int) {
	a.Popup(pages.HeadphonePopup(a.env(), i, a.w, a.h), pages.PopupFor)
}

// Root is the drawing root (for tests and the preview).
func (a *App) Root() *ui.Root { return a.root }

// PageTitle is the current page's title (for tests and diagnostics).
func (a *App) PageTitle() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.stack[len(a.stack)-1].Title()
}

// Page is the current page (for tests).
func (a *App) Page() pages.Page {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.stack[len(a.stack)-1]
}

// Depth is how deep the navigation stack is (for tests).
func (a *App) Depth() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.stack)
}

// Rebuild forces the view tree to be rebuilt before the next frame.
func (a *App) Rebuild() {
	a.mu.Lock()
	a.dirty = true
	a.mu.Unlock()
}

// build assembles the shell for the current page and sets it on the root.
func (a *App) build() {
	a.mu.Lock()
	page := a.stack[len(a.stack)-1]
	depth := len(a.stack)
	overlay := a.overlay
	toast := a.toast
	toastOn := time.Now().Before(a.toastUntil)
	popup := a.popup
	if !time.Now().Before(a.popupUntil) {
		popup = nil
	}
	a.dirty = false
	a.mu.Unlock()
	if a.model.TransferMode() {
		page, depth = pages.NewTransfer(), 1
	}
	// A card or drive the desk cannot read: RØDE's popup asks to format it,
	// once per insertion.
	if overlay == nil {
		if vol, title, msg, yes, onYes, ok := pages.UnrecognisedDisk(a.env()); ok && a.diskAsked != vol+1 {
			a.diskAsked = vol + 1
			a.Confirm(title, msg, yes, true, onYes)
			a.mu.Lock()
			overlay = a.overlay
			a.mu.Unlock()
		} else if !ok {
			a.diskAsked = 0
		}
	}

	full := image.Rect(0, 0, a.w, a.h)
	var shell *ui.Backdrop
	if pages.FullBleedOf(page) {
		shell = &ui.Backdrop{Plain: true}
		shell.Fill = a.ctx.Theme.Bg
		shell.SetBounds(full)
		shell.Add(page.Build(a.env(), full))
	} else {
		shell = &ui.Backdrop{}
		shell.SetBounds(full)
		shell.Add(a.header(page, depth > 1)...)
		content := image.Rect(sidePad, contentTop, a.w-sidePad, a.h-bottomPad)
		shell.Add(page.Build(a.env(), content))
	}
	if popup != nil {
		shell.Add(popup)
	}
	if toastOn && toast != "" {
		shell.Add(a.toastView(toast))
	}

	var rootView ui.View = shell
	var focusRoot ui.View = shell
	if overlay != nil {
		overlay.SetBounds(full)
		top := &ui.Container{}
		top.SetBounds(full)
		top.Add(shell, overlay)
		rootView = top
		focusRoot = overlay
	}
	a.root.SetView(rootView)
	a.focus.collect(focusRoot)
	if a.model.Online() {
		a.syncEncoderRing() // the first frame after the desk came up
	}
}

// header is the bar across the top: back and home, the title, or the page's
// own header controls.
func (a *App) header(page pages.Page, back bool) []ui.View {
	th := a.ctx.Theme
	bar := image.Rect(0, 0, a.w, headerH)
	var out []ui.View
	ownNav := false
	if hp, ok := page.(pages.HeaderPage); ok {
		var views []ui.View
		views, ownNav = hp.Header(a.env(), bar)
		out = append(out, views...)
	} else if title := a.text.T(page.Title()); title != "" {
		t := ui.NewLabel(title, th.Fonts.Title, th.Text, ui.AlignCenter)
		t.SetBounds(image.Rect(sidePad+th.Button+20, 0, a.w-sidePad-th.Button-20, headerH))
		out = append(out, t)
	}
	if ownNav {
		return out
	}
	y := (headerH - th.Button) / 2
	if back {
		b := ui.NewButton("", func() { a.Pop() })
		b.Icon = "IconBack"
		b.SetBounds(image.Rect(sidePad, y, sidePad+th.Button, y+th.Button))
		out = append([]ui.View{b}, out...)
	}
	home := ui.NewButton("", func() { a.Home() })
	home.Icon = "IconHome"
	home.SetBounds(image.Rect(a.w-sidePad-th.Button, y, a.w-sidePad, y+th.Button))
	return append(out, home)
}

// toastView is a brief message strip near the bottom.
func (a *App) toastView(msg string) ui.View {
	th := a.ctx.Theme
	w := min(900, 80+ui.TextWidth(th.Fonts.Body, msg))
	r := image.Rect((a.w-w)/2, a.h-140, (a.w+w)/2, a.h-60)
	c := &ui.Container{}
	c.SetBounds(r)
	card := ui.NewPanelView(th.PanelHi)
	card.Radius = 16
	card.SetBounds(r)
	lbl := ui.NewLabel(msg, th.Fonts.Body, th.Text, ui.AlignCenter)
	lbl.SetBounds(r)
	c.Add(card, lbl)
	return c
}
