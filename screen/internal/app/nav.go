package app

import (
	"time"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/pages"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// The navigation surface (pages.Nav). Each change marks the view tree for a
// rebuild before the next frame; a page change also drops the knob's focus.

// Push shows a page on top of the current one.
func (a *App) Push(p pages.Page) {
	if p == nil {
		return
	}
	a.mu.Lock()
	a.stack = append(a.stack, p)
	a.overlay = nil
	a.dirty = true
	a.mu.Unlock()
	a.focus.reset()
	a.syncChannel()
	a.enter(p)
}

// Pop goes back one page.
func (a *App) Pop() {
	a.mu.Lock()
	var gone pages.Page
	if len(a.stack) > 1 {
		gone = a.stack[len(a.stack)-1]
		a.stack = a.stack[:len(a.stack)-1]
	}
	a.overlay = nil
	a.dirty = true
	a.mu.Unlock()
	a.focus.reset()
	a.syncChannel()
	if gone != nil {
		a.leave(gone)
	}
}

// Home returns to the mixer.
func (a *App) Home() {
	a.mu.Lock()
	gone := append([]pages.Page(nil), a.stack[1:]...)
	a.stack = a.stack[:1]
	a.overlay = nil
	a.dirty = true
	a.mu.Unlock()
	a.focus.reset()
	a.syncChannel()
	a.leave(gone...)
}

// Open shows a top-level page by its id.
func (a *App) Open(id string) {
	if p, ok := pages.Lookup(id); ok {
		a.Push(p)
	} else {
		a.Toast("—")
	}
}

// Replace swaps the current page for another (keeps the depth).
func (a *App) Replace(p pages.Page) {
	if p == nil {
		return
	}
	a.mu.Lock()
	gone := a.stack[len(a.stack)-1]
	a.stack[len(a.stack)-1] = p
	a.overlay = nil
	a.dirty = true
	a.mu.Unlock()
	a.focus.reset()
	a.syncChannel()
	a.leave(gone)
	a.enter(p)
}

// Refresh rebuilds the current page's views; the knob's focus stays where
// it was.
func (a *App) Refresh() {
	a.mu.Lock()
	a.dirty = true
	a.mu.Unlock()
}

// Keyboard asks for text with the on-screen keyboard.
func (a *App) Keyboard(prompt, initial string, done func(string)) {
	kb := ui.NewKeyboard(prompt, initial, func(s string) {
		a.clearOverlay()
		if done != nil {
			done(s)
		}
	}, a.clearOverlay)
	a.setOverlay(kb)
}

// Confirm shows a modal dialog; onYes runs if confirmed.
func (a *App) Confirm(title, message, yes string, danger bool, onYes func()) {
	d := ui.NewDialog(title, message, yes, func() {
		a.clearOverlay()
		if onYes != nil {
			onYes()
		}
	}, a.text.T("CANCEL"), a.clearOverlay)
	d.Danger = danger
	a.setOverlay(d)
}

// Toast shows a brief message near the bottom. It stays four seconds, as
// RØDE's notification popup does ("Marker added at …", "TX Disconnected").
func (a *App) Toast(msg string) {
	a.mu.Lock()
	a.toast = msg
	a.toastUntil = time.Now().Add(4000 * time.Millisecond)
	a.dirty = true
	a.mu.Unlock()
}

// Modal lays a view over the page until Dismiss (the CallMe popups).
func (a *App) Modal(v ui.View) { a.setOverlay(v) }

// Dismiss removes the modal view, keyboard or dialog.
func (a *App) Dismiss() { a.clearOverlay() }

// Overlay is the modal view currently up, or nil (for tests).
func (a *App) Overlay() ui.View {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.overlay
}

// RodeScreen hands the screen back to RØDE's interface (ends the UI).
func (a *App) RodeScreen() {
	a.mu.Lock()
	q := a.quit
	a.mu.Unlock()
	if q != nil {
		q()
	}
}

func (a *App) setOverlay(v ui.View) {
	a.mu.Lock()
	a.overlay = v
	a.dirty = true
	a.mu.Unlock()
	a.focus.reset()
}

func (a *App) clearOverlay() {
	a.mu.Lock()
	a.overlay = nil
	a.dirty = true
	a.mu.Unlock()
	a.focus.reset()
}

// channelOpen is the strip whose channel pages are open (the channel page
// itself or one of its pages above it), or -1.
func (a *App) channelOpen() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i := len(a.stack) - 1; i >= 0; i-- {
		if ch, ok := a.stack[i].(*pages.ChannelPage); ok {
			return ch.Strip()
		}
	}
	return -1
}

// syncChannel tells the desk which channel's pages are open after a page
// change, as RØDE's screen does: the button above that fader lights, and
// goes out when the pages close.
func (a *App) syncChannel() {
	a.model.SelectChannel(a.channelOpen())
}

// channelButton answers the desk's selected channel, which the buttons
// above the physical faders change: a press selects the strip and opens its
// channel page, the next press (or another strip's button) deselects it and
// the pages close, as on RØDE's screen. The button's light is the desk's.
func (a *App) channelButton(strip int) {
	a.mu.Lock()
	overlay := a.overlay
	a.mu.Unlock()
	if overlay != nil {
		return
	}
	if strip < 0 {
		if a.channelOpen() >= 0 {
			a.Home()
		}
		return
	}
	if strip >= a.model.StripCount() || a.channelOpen() == strip {
		return
	}
	a.mu.Lock()
	gone := append([]pages.Page(nil), a.stack[1:]...)
	a.stack = append(a.stack[:1], pages.NewChannel(strip))
	a.overlay = nil
	a.dirty = true
	a.mu.Unlock()
	a.focus.reset()
	a.syncChannel()
	a.leave(gone...)
}

// ChannelButton feeds a change of the selected channel (for tests and the
// preview): a strip's number, or -1.
func (a *App) ChannelButton(strip int) { a.channelButton(strip) }
