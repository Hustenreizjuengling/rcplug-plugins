// Package ui is the look of the screen and its building blocks: a theme, a
// drawing context that holds the fonts and the device's icons, a tree of
// views that lay out and draw themselves and take touches (and the encoder's
// focus), and the common controls (cards, labels, buttons, knobs, meters,
// steppers, lists, tabs). It draws with package gfx and knows nothing of the
// device or the desk.
package ui

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
)

// Theme is the colours, sizes and fonts of the look. They follow RØDE's own
// screen (a 1280 × 720 page on a 5" panel, so everything is large): black
// behind the mixer, a dark grey gradient behind the menus, grey gradient
// cards, white text, a light card for the chosen option and green for what
// is live or in focus.
type Theme struct {
	// Colours.
	Bg          gfx.Color // the screen behind the mixer
	PageTop     gfx.Color // the gradient behind menu pages, top …
	PageBottom  gfx.Color // … and bottom
	Panel       gfx.Color // a flat card
	PanelHi     gfx.Color // a pressed card
	CardTop     gfx.Color // a tile's gradient, top …
	CardBottom  gfx.Color // … and bottom
	LightTop    gfx.Color // the chosen option's light card, top …
	LightBottom gfx.Color // … and bottom
	Black       gfx.Color // a black inset (meters, value fields)
	Line        gfx.Color // a divider
	Text        gfx.Color // the main text
	TextDim     gfx.Color // a label or a lesser value
	TextMuted   gfx.Color // disabled
	TextOnLight gfx.Color // text on a light card
	Accent      gfx.Color // RØDE's green: live, chosen, in focus
	Record      gfx.Color // recording red
	Warn        gfx.Color // amber
	Blue        gfx.Color // the blue of a chosen segment (routing)
	Knob        gfx.Color // a control's track
	KnobFill    gfx.Color // a control's filled part
	MeterTrack  gfx.Color // behind a meter

	// Sizes in pixels.
	Pad    int // the gap at the screen's edge and between cards
	Gap    int // a small gap inside a card
	Radius int // a card's corner
	Row    int // a list row's height
	Button int // a square button's side (back, home)

	Fonts Fonts
}

// Fonts are the type sizes the look uses.
type Fonts struct {
	Tiny  *Font // 20: meter scales, hints
	Small *Font // 24: captions
	Body  *Font // 30: text, tile labels
	Label *Font // 28 bold: button captions
	Title *Font // 44: page titles
	Big   *Font // 56 bold: the clock
	Huge  *Font // 72 bold: a stepper's value
}

// Dark is the standard look.
func Dark(f *FontSet) *Theme {
	return &Theme{
		Bg:          gfx.Hex(0x000000),
		PageTop:     gfx.Hex(0x343434),
		PageBottom:  gfx.Hex(0x0c0c0c),
		Panel:       gfx.Hex(0x2a2a2a),
		PanelHi:     gfx.Hex(0x4a4a4a),
		CardTop:     gfx.Hex(0x474747),
		CardBottom:  gfx.Hex(0x2b2b2b),
		LightTop:    gfx.Hex(0xf6f6f6),
		LightBottom: gfx.Hex(0xcacaca),
		Black:       gfx.Hex(0x000000),
		Line:        gfx.Hex(0x4d4d4d),
		Text:        gfx.Hex(0xffffff),
		TextDim:     gfx.Hex(0xb4b4b4),
		TextMuted:   gfx.Hex(0x6a6a6a),
		TextOnLight: gfx.Hex(0x141414),
		Accent:      gfx.Hex(0x3bd33b),
		Record:      gfx.Hex(0xd92020),
		Warn:        gfx.Hex(0xf0a000),
		Blue:        gfx.Hex(0x1e62e6),
		Knob:        gfx.Hex(0x3a3a3a),
		KnobFill:    gfx.Hex(0xffffff),
		MeterTrack:  gfx.Hex(0x161616),
		Pad:         30,
		Gap:         16,
		Radius:      12,
		Row:         96,
		Button:      72,
		Fonts: Fonts{
			Tiny:  f.Regular.Sized(20),
			Small: f.Regular.Sized(24),
			Body:  f.Regular.Sized(30),
			Label: f.Bold.Sized(28),
			Title: f.Regular.Sized(44),
			Big:   f.Bold.Sized(56),
			Huge:  f.Bold.Sized(72),
		},
	}
}

// inset shrinks a rectangle on every side.
func inset(r image.Rectangle, by int) image.Rectangle {
	return image.Rect(r.Min.X+by, r.Min.Y+by, r.Max.X-by, r.Max.Y-by)
}

// Inset is inset for the pages.
func Inset(r image.Rectangle, by int) image.Rectangle { return inset(r, by) }
