// Package app ties the screen interface together: it owns the drawing root,
// the desk model and the navigation stack, frames each page with a header
// and handles the modal keyboard and dialogs, and runs the draw-and-touch
// loop against an output and a touch source.
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

const headerH = 56

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
	dirty      bool // the view tree must be rebuilt
	quit       func()
}

// New makes an app for a w × h screen.
func New(w, h int, ctx *ui.Context, model *desk.Model, text *lang.Texts, plugins pages.PluginSource) *App {
	a := &App{ctx: ctx, model: model, text: text, plugins: plugins, w: w, h: h}
	a.root = ui.NewRoot(ctx, w, h)
	a.stack = []pages.Page{pages.NewHome()}
	a.dirty = true
	return a
}

// env builds the page environment with this app as the navigator.
func (a *App) env() *pages.Env {
	return &pages.Env{Ctx: a.ctx, Desk: a.model, Text: a.text, Nav: a, Plugins: a.plugins}
}

// Root is the drawing root (for tests and the preview).
func (a *App) Root() *ui.Root { return a.root }

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
	a.dirty = false
	a.mu.Unlock()

	full := image.Rect(0, 0, a.w, a.h)
	shell := &ui.Container{}
	shell.SetBounds(full)

	if pages.FullBleedOf(page) {
		body := page.Build(a.env(), full)
		shell.Add(body)
	} else {
		shell.Add(a.header(page.Title(), depth > 1)...)
		content := image.Rect(a.ctx.Theme.Pad, headerH+a.ctx.Theme.Pad, a.w-a.ctx.Theme.Pad, a.h-a.ctx.Theme.Pad)
		body := page.Build(a.env(), content)
		shell.Add(body)
	}
	if toastOn && toast != "" {
		shell.Add(a.toastView(toast))
	}

	var rootView ui.View = shell
	if overlay != nil {
		overlay.SetBounds(full)
		top := &ui.Container{}
		top.SetBounds(full)
		top.Add(shell, overlay)
		rootView = top
	}
	a.root.SetView(rootView)
}

// header is the back button and the page title.
func (a *App) header(title string, back bool) []ui.View {
	th := a.ctx.Theme
	bar := image.Rect(0, 0, a.w, headerH)
	bg := ui.NewPanelView(th.Panel)
	bg.Radius = 0
	bg.SetBounds(bar)
	out := []ui.View{bg}
	x := 8
	if back {
		b := ui.NewButton("‹", func() { a.Pop() })
		b.Fill = th.Panel
		b.SetBounds(image.Rect(x, 6, x+56, headerH-6))
		out = append(out, b)
		x += 64
	}
	home := ui.NewButton("", func() { a.Home() })
	home.Icon = "IconHome"
	home.Fill = th.Panel
	home.SetBounds(image.Rect(a.w-58, 6, a.w-6, headerH-6))
	t := ui.NewLabel(a.text.T(title), th.Fonts.Title, th.Text, ui.AlignLeft)
	t.SetBounds(image.Rect(x+4, 0, a.w-70, headerH))
	out = append(out, t, home)
	return out
}

// toastView is a brief message strip near the bottom.
func (a *App) toastView(msg string) ui.View {
	th := a.ctx.Theme
	w := 600
	r := image.Rect((a.w-w)/2, a.h-120, (a.w+w)/2, a.h-64)
	c := &ui.Container{}
	c.SetBounds(r)
	card := ui.NewPanelView(th.PanelHi)
	card.SetBounds(r)
	lbl := ui.NewLabel(msg, th.Fonts.Body, th.Text, ui.AlignCenter)
	lbl.SetBounds(r)
	c.Add(card, lbl)
	return c
}
