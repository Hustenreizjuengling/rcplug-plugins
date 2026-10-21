package ui

// GrabFocus asks the app to move the main knob's focus to a view after a
// touch (RØDE's "tap a field, then turn the encoder"); with adjusting the
// knob changes the view's value at once. The app installs the hook; without
// one (a bare render) the call does nothing.
var GrabFocus func(v Focusable, adjusting bool)

// grabFocus is GrabFocus made safe to call.
func grabFocus(v Focusable, adjusting bool) {
	if GrabFocus != nil {
		GrabFocus(v, adjusting)
	}
}
