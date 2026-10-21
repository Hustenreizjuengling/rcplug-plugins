package app

import "github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/pages"

// Page lifecycle: pages that ask for it hear when they come on screen and
// when they leave the stack (the Bluetooth page's discoverability, the
// Wi-Fi scan).

func (a *App) enter(p pages.Page) {
	if e, ok := p.(pages.Enterer); ok {
		e.Enter(a.env())
	}
}

func (a *App) leave(ps ...pages.Page) {
	for _, p := range ps {
		if l, ok := p.(pages.Leaver); ok {
			l.Leave(a.env())
		}
	}
}
