package app

import (
	"sync"
	"time"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/pages"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// The main knob on menu pages: turning moves a green ring from control to
// control in reading order, pressing taps the control (or starts adjusting
// a value: turning then changes it, pressing again lets go), and holding
// the knob goes back a page. The mixer drives the knob itself (see
// pages.HomePage).

// holdFor is how long a press counts as a hold.
const holdFor = 900 * time.Millisecond

type focus struct {
	mu        sync.Mutex
	list      []ui.Focusable
	idx       int
	adjusting bool
	root      ui.View
}

// collect gathers the focusable views under root (after a build) and keeps
// the focus on the same index when the page only refreshed.
func (f *focus) collect(root ui.View) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.root = root
	f.list = ui.Focusables(root)
	if f.idx >= len(f.list) {
		f.idx = -1
		f.adjusting = false
	}
	if f.idx >= 0 {
		f.list[f.idx].SetFocus(true)
		if f.adjusting {
			f.list[f.idx].Activate() // back into adjusting on the fresh view
		}
	}
}

// reset drops the focus (a page change).
func (f *focus) reset() {
	f.mu.Lock()
	f.idx, f.adjusting = -1, false
	f.mu.Unlock()
}

// move shifts the focus by delta, wrapping around.
func (f *focus) move(delta int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := len(f.list)
	if n == 0 {
		return
	}
	if f.idx >= 0 && f.idx < n {
		f.list[f.idx].SetFocus(false)
	}
	switch {
	case f.idx < 0 && delta > 0:
		f.idx = 0
	case f.idx < 0:
		f.idx = n - 1
	default:
		f.idx = ((f.idx+delta)%n + n) % n
	}
	f.list[f.idx].SetFocus(true)
	ui.Reveal(f.root, f.list[f.idx])
}

// current is the focused view, if any.
func (f *focus) current() ui.Focusable {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.idx < 0 || f.idx >= len(f.list) {
		return nil
	}
	return f.list[f.idx]
}

// handleEncoder routes a knob event: to the page if it drives the knob
// itself, else through the focus.
func (a *App) handleEncoder(ev desk.EncoderEvent) {
	a.mu.Lock()
	page := a.stack[len(a.stack)-1]
	overlay := a.overlay
	depth := len(a.stack)
	a.mu.Unlock()

	if overlay == nil {
		// The page's views follow the change through their own Dirty (the
		// monitor ring, the fader caps); a page whose layout changes asks
		// for a Refresh itself. Rebuilding the whole tree per detent would
		// queue the turns behind the drawing.
		if h, ok := page.(pages.EncoderHandler); ok && h.Encoder(a.env(), ev) {
			return
		}
	}
	switch {
	case ev.Delta != 0:
		if cur := a.focus.current(); cur != nil && a.focus.isAdjusting() {
			cur.Adjust(ev.Delta)
			return
		}
		if ev.Delta > 0 {
			a.focus.move(1)
		} else {
			a.focus.move(-1)
		}
	case ev.Release && ev.Held >= holdFor:
		if a.focus.isAdjusting() {
			a.focus.setAdjusting(false)
			if cur := a.focus.current(); cur != nil {
				cur.Activate() // leave adjusting
			}
			return
		}
		if overlay != nil {
			a.clearOverlay()
		} else if depth > 1 {
			a.Pop()
		}
	case ev.Release:
		cur := a.focus.current()
		if cur == nil {
			a.focus.move(1)
			return
		}
		a.focus.setAdjusting(cur.Activate())
	}
}

func (f *focus) isAdjusting() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.adjusting
}

func (f *focus) setAdjusting(v bool) {
	f.mu.Lock()
	f.adjusting = v
	f.mu.Unlock()
}

// Encoder feeds a knob event (for tests and the browser preview).
func (a *App) Encoder(ev desk.EncoderEvent) { a.handleEncoder(ev) }
