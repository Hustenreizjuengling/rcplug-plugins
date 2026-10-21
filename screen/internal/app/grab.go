package app

import "github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"

// grab moves the knob's focus to a view that was touched (the view must be
// in the current page's focus list), optionally straight into adjusting, so
// the next turn of the knob changes that view. A touch on a view that is not
// in the list leaves the focus alone.
func (f *focus) grab(v ui.Focusable, adjusting bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	at := -1
	for i, c := range f.list {
		if c == v {
			at = i
			break
		}
	}
	if at < 0 {
		return
	}
	if f.idx >= 0 && f.idx < len(f.list) && f.idx != at {
		f.list[f.idx].SetFocus(false)
	}
	f.idx = at
	f.list[at].SetFocus(true)
	if adjusting != f.adjusting {
		// Activate toggles the view's adjusting state; it starts from the
		// state the view keeps itself, so a touch that already toggled it
		// is not undone here.
		f.adjusting = adjusting
	}
}
