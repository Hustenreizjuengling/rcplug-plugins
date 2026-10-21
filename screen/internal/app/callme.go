package app

import "github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/pages"

// ShowCallMePopup lays a CallMe popup over the page by name (the preview).
func (a *App) ShowCallMePopup(name string) {
	if v := pages.CallMePopup(a.env(), name); v != nil {
		a.setOverlay(v)
	}
}

// watchCallMe shows the CallMe popups the desk's state asks for: an
// incoming call (while hosting is on), the one-hour warning and the rating
// after a call. A popup never covers the keyboard or a dialog; an incoming
// call popup goes when the call is taken or withdrawn elsewhere.
func (a *App) watchCallMe() {
	show, clear := a.callme.Check(a.env())
	a.mu.Lock()
	overlay := a.overlay
	a.mu.Unlock()
	kind, isPopup := pages.IsCallMePopup(overlay)
	if clear && isPopup && kind == "incoming" {
		a.clearOverlay()
		overlay, isPopup = nil, false
	}
	if show == nil {
		return
	}
	if overlay != nil && !isPopup {
		return // the keyboard or a dialog is up; the state stays visible on the pages
	}
	a.setOverlay(show)
}
