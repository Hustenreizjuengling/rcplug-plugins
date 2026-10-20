package app

import (
	"context"
	"image"
	"time"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/pages"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/panel"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// Run draws the interface and reads touches and the main knob until ctx ends
// or the screen is handed back to RØDE. out shows the page (landscape);
// touch carries finger events in the panel's portrait coordinates; turn is
// how the page lies on the panel (see the panel package).
func (a *App) Run(ctx context.Context, out panel.Output, touch <-chan panel.TouchEvent, turn int) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	a.mu.Lock()
	a.quit = cancel
	a.mu.Unlock()

	pageW, pageH := out.Size()
	if pageW != a.w || pageH != a.h {
		a.w, a.h = pageW, pageH
		a.root = ui.NewRoot(a.ctx, a.w, a.h)
		a.Rebuild()
	}
	panelW, panelH := pageH, pageW // the panel is the page turned

	refresh := 60
	if s, ok := out.(interface{ Refresh() int }); ok && s.Refresh() > 0 {
		refresh = s.Refresh()
	}
	frame := time.Second / time.Duration(refresh)
	tick := time.NewTicker(frame)
	defer tick.Stop()

	wasDown := false
	lastOnline, lastStrips := false, -1
	knob := a.model.Encoder()
	button := a.model.ChannelButton()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-touch:
			if !ok {
				touch = nil
				continue
			}
			p := panel.PageOf(ev.At, panelW, panelH, turn)
			switch {
			case ev.Down && !wasDown:
				a.root.Touch(ui.Down, p)
			case ev.Down && wasDown:
				a.root.Touch(ui.Move, p)
			case !ev.Down:
				a.root.Touch(ui.Up, p)
			}
			wasDown = ev.Down
		case ev := <-knob:
			a.handleEncoder(ev)
		case n := <-button:
			a.channelButton(n)
		case <-tick.C:
			// The device coming online (or its strip count changing) needs a
			// fresh layout.
			if on, n := a.model.Online(), a.model.StripCount(); on != lastOnline || n != lastStrips {
				lastOnline, lastStrips = on, n
				a.Rebuild()
			}
			a.expireToast()
			a.mu.Lock()
			rebuild := a.dirty
			a.mu.Unlock()
			if rebuild {
				a.build()
			}
			changed := a.root.Render()
			if len(changed) > 0 {
				if err := out.Present(a.root.Image(), changed); err != nil {
					return err
				}
			}
		}
	}
}

// expireToast clears a toast or popup that has timed out, and watches the
// headphone knobs: a turn shows the level for a moment.
func (a *App) expireToast() {
	a.mu.Lock()
	if a.toast != "" && time.Now().After(a.toastUntil) {
		a.toast = ""
		a.dirty = true
	}
	if a.popup != nil && time.Now().After(a.popupUntil) {
		a.popup = nil
		a.dirty = true
	}
	a.mu.Unlock()
	a.watchHeadphones()
}

// watchHeadphones shows the headphone popup when a knob's level changes
// (not on the first reading, nor while the device is offline).
func (a *App) watchHeadphones() {
	n := a.model.Headphones()
	first := a.hpLast == nil
	if first {
		a.hpLast = make([]float64, n)
	}
	for i := 0; i < n && i < len(a.hpLast); i++ {
		v := a.model.HeadphoneFrac(i)
		if v != a.hpLast[i] && !first && a.model.Online() {
			a.Popup(pages.HeadphonePopup(a.env(), i, a.w, a.h), pages.PopupFor)
		}
		a.hpLast[i] = v
	}
}

// Popup lays a passing view over the page for a while (the headphone level);
// a new one replaces the old.
func (a *App) Popup(v ui.View, keep time.Duration) {
	a.mu.Lock()
	a.popup = v
	a.popupUntil = time.Now().Add(keep)
	a.dirty = true
	a.mu.Unlock()
}

// RenderOnce builds and renders a single frame (for the PNG preview and
// tests). It returns the rendered image's bounds as changed.
func (a *App) RenderOnce() []image.Rectangle {
	a.build()
	return a.root.Render()
}

// Frame builds the view if it changed, renders, and returns the current
// picture (for the browser preview).
func (a *App) Frame() *gfx.Image {
	a.expireToast()
	a.mu.Lock()
	rebuild := a.dirty
	a.mu.Unlock()
	if rebuild {
		a.build()
	}
	a.root.Render()
	return a.root.Image()
}

// Touch routes a touch at a page point (for the browser preview).
func (a *App) Touch(phase ui.TouchPhase, p image.Point) { a.root.Touch(phase, p) }
