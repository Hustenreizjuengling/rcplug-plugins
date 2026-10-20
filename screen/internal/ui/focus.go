package ui

// The encoder's focus. The device's main knob reaches controls without the
// touchscreen: turning it moves a green ring from control to control,
// pressing it taps the control or (for a value) starts adjusting, turning
// then changes the value and pressing again lets go — as RØDE's screen does.
// Controls implement Focusable; the app walks the view tree in order and
// keeps the index of the focused one.

// Focusable is a view the encoder can reach.
type Focusable interface {
	View
	// SetFocus gives or takes the focus (the view draws a ring).
	SetFocus(on bool)
	// Focused reports whether the view has the focus.
	Focused() bool
	// CanFocus reports whether the view can take the focus now.
	CanFocus() bool
	// Activate is a press while focused: it taps a button, flips a switch,
	// or for a value enters adjusting (returns true) and leaves it again
	// (returns false).
	Activate() bool
	// Adjust is a turn while adjusting: delta steps, negative = left.
	Adjust(delta int)
}

// Walker is a view with children.
type Walker interface {
	Walk(fn func(View))
}

// Walk visits v and every view under it, in drawing order.
func Walk(v View, fn func(View)) {
	if v == nil {
		return
	}
	fn(v)
	if w, ok := v.(Walker); ok {
		w.Walk(fn)
	}
}

// Focusables lists the views under root that can take the focus, in order.
func Focusables(root View) []Focusable {
	var out []Focusable
	Walk(root, func(v View) {
		if f, ok := v.(Focusable); ok && f.CanFocus() {
			out = append(out, f)
		}
	})
	return out
}

// Revealer is a scrolling view that can bring a child into sight.
type Revealer interface {
	Reveal(v View)
}

// Reveal asks every scrolling view under root to show v if it holds it.
func Reveal(root, v View) {
	Walk(root, func(w View) {
		if r, ok := w.(Revealer); ok {
			r.Reveal(v)
		}
	})
}

// Adjuster is embedded by value controls to share the adjusting state.
type Adjuster struct {
	adjusting bool
}

// Adjusting reports whether the encoder is changing the control's value.
func (a *Adjuster) Adjusting() bool { return a.adjusting }

// toggleAdjust flips adjusting and returns the new state.
func (a *Adjuster) toggleAdjust() bool {
	a.adjusting = !a.adjusting
	return a.adjusting
}

// stop leaves adjusting (when the focus moves away).
func (a *Adjuster) stop() { a.adjusting = false }
