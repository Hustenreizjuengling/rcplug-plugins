// Package pages builds the screens of the alternative interface: the mixer,
// the channel and processing pages, pads, the menus and settings, and the
// rcplug and plugin pages. A page turns the desk model and the device's
// icons and text into a tree of ui views; the app frames it with a header
// and drives navigation, touch and the main knob.
package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/lang"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// Env is what a page needs to build itself.
type Env struct {
	Ctx     *ui.Context
	Desk    *desk.Model
	Text    *lang.Texts
	Nav     Nav
	Plugins PluginSource // may be nil
}

// T is a shortcut for a localized label.
func (e *Env) T(key string) string { return e.Text.T(key) }

// TR is T for the few strings the device has no key for: it uses RØDE's
// translation when there is one, else the German or English fallback given
// here, by the chosen language.
func (e *Env) TR(key, de, en string) string {
	if key != "" && e.Text.Has(key) {
		return e.Text.T(key)
	}
	if e.Text.German() {
		return de
	}
	return en
}

// Nav lets a page move between screens and hand control back to RØDE.
type Nav interface {
	// Push shows a page on top of the current one (with a back button).
	Push(p Page)
	// Pop goes back one page.
	Pop()
	// Home returns to the mixer.
	Home()
	// Open shows a page by its id (see the registry), e.g. "settings".
	Open(id string)
	// Replace swaps the current page for another (the next strip, say).
	Replace(p Page)
	// Refresh rebuilds the current page's views (after a state change the
	// page keeps itself, such as a selection).
	Refresh()
	// Keyboard asks for text with the on-screen keyboard, then calls done.
	Keyboard(prompt, initial string, done func(string))
	// Confirm shows a modal dialog; onYes runs if confirmed.
	Confirm(title, message, yes string, danger bool, onYes func())
	// Toast shows a brief message.
	Toast(msg string)
	// RodeScreen hands the screen back to RØDE's own interface (ends the UI).
	RodeScreen()
}

// Page is one screen. Build lays its body into r and returns the view; Title
// is the header caption.
type Page interface {
	Title() string
	Build(env *Env, r image.Rectangle) ui.View
}

// FullBleed is an optional interface: a page that fills the whole screen with
// no standard header (the mixer).
type FullBleed interface {
	FullBleed() bool
}

// HeaderPage is an optional interface: a page that puts its own controls in
// the header bar (a strip switcher, the assign page's cross and tick). With
// ownNav the app adds no back and home buttons.
type HeaderPage interface {
	Header(env *Env, bar image.Rectangle) (views []ui.View, ownNav bool)
}

// EncoderHandler is an optional interface: a page that drives the main knob
// itself (the mixer: monitor level and the virtual faders). It returns false
// to leave the event to the app's focus navigation.
type EncoderHandler interface {
	Encoder(env *Env, ev desk.EncoderEvent) bool
}

// PluginSource gives the rcplug and plugin pages their data (implemented by
// the ext package; nil when there is none).
type PluginSource interface {
	// Plugins lists the installed plugins (for status tiles).
	Plugins() []PluginInfo
	// Page fetches a plugin's declared screen page (nil, false if none).
	Page(name string) (*PluginPage, bool)
	// Submit sends a control's new value back to a plugin.
	Submit(name, control string, value interface{}) error
	// DashboardURL is the host's dashboard address (for a QR code).
	DashboardURL() string
}

// PluginInfo is one installed plugin.
type PluginInfo struct {
	Name    string
	Title   string
	Running bool
	Icon    string
}

// PluginPage is a plugin's declared screen (a list of controls the UI draws
// in the RØDE style; the plugin never draws pixels itself).
type PluginPage struct {
	Title    string          `json:"title"`
	Controls []PluginControl `json:"controls"`
}

// PluginControl is one control on a plugin page.
type PluginControl struct {
	ID      string   `json:"id"`
	Type    string   `json:"type"` // toggle slider choice button value meter
	Label   string   `json:"label"`
	Value   float64  `json:"value"`
	On      bool     `json:"on"`
	Text    string   `json:"text"`
	Min     float64  `json:"min"`
	Max     float64  `json:"max"`
	Unit    string   `json:"unit"`
	Options []string `json:"options"`
	Danger  bool     `json:"danger"`  // a button that asks before acting
	Confirm string   `json:"confirm"` // the confirmation message
}

// FullBleedOf reports whether a page wants the whole screen.
func FullBleedOf(p Page) bool {
	fb, ok := p.(FullBleed)
	return ok && fb.FullBleed()
}
