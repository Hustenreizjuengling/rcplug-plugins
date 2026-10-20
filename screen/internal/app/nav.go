package app

import (
	"time"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/pages"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// The navigation surface (pages.Nav). Each change marks the view tree for a
// rebuild before the next frame.

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
}

// Pop goes back one page.
func (a *App) Pop() {
	a.mu.Lock()
	if len(a.stack) > 1 {
		a.stack = a.stack[:len(a.stack)-1]
	}
	a.overlay = nil
	a.dirty = true
	a.mu.Unlock()
}

// Home returns to the mixer.
func (a *App) Home() {
	a.mu.Lock()
	a.stack = a.stack[:1]
	a.overlay = nil
	a.dirty = true
	a.mu.Unlock()
}

// Open shows a top-level page by its id.
func (a *App) Open(id string) {
	if p, ok := pages.Lookup(id); ok {
		a.Push(p)
	} else {
		a.Toast("—")
	}
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

// Toast shows a brief message near the bottom.
func (a *App) Toast(msg string) {
	a.mu.Lock()
	a.toast = msg
	a.toastUntil = time.Now().Add(2500 * time.Millisecond)
	a.dirty = true
	a.mu.Unlock()
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
}

func (a *App) clearOverlay() {
	a.mu.Lock()
	a.overlay = nil
	a.dirty = true
	a.mu.Unlock()
}
