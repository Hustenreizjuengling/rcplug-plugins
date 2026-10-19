// Package ui is the look of the screen and its building blocks: a theme, a
// drawing context that holds the fonts and the device's icons, a tree of
// views that lay out and draw themselves and take touches, and the common
// controls (panels, labels, buttons, faders, knobs, meters, lists, tabs).
// It draws with package gfx and knows nothing of the device or the desk.
package ui

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
)

// Theme is the colours, sizes and fonts of the look. The colours follow
// RØDE's dark screen; they are refined against screenshots as screens are
// built.
type Theme struct {
	// Colours.
	Bg        gfx.Color // the screen behind everything
	Panel     gfx.Color // a card or strip
	PanelHi   gfx.Color // a raised or selected card
	Line      gfx.Color // a divider
	Text      gfx.Color // the main text
	TextDim   gfx.Color // a label or a lesser value
	TextMuted gfx.Color // disabled
	Accent    gfx.Color // the green of RØDE's highlights
	Record    gfx.Color // recording red
	Warn      gfx.Color // a warning
	Knob      gfx.Color // a control's track
	KnobFill  gfx.Color // a control's filled part

	// Sizes in pixels.
	Pad    int // the gap at the screen's edge and between cards
	Gap    int // a small gap inside a card
	Radius int // a card's corner
	Row    int // a list row's height

	Fonts Fonts
}

// Fonts are the type sizes the look uses.
type Fonts struct {
	Tiny, Small, Body, Label, Title, Big *Font
}

// Dark is the standard look.
func Dark(f *FontSet) *Theme {
	return &Theme{
		Bg:        gfx.Hex(0x121212),
		Panel:     gfx.Hex(0x1f1f1f),
		PanelHi:   gfx.Hex(0x2c2c2c),
		Line:      gfx.Hex(0x3a3a3a),
		Text:      gfx.Hex(0xffffff),
		TextDim:   gfx.Hex(0xb0b0b0),
		TextMuted: gfx.Hex(0x6a6a6a),
		Accent:    gfx.Hex(0x7ad43a),
		Record:    gfx.Hex(0xe5322d),
		Warn:      gfx.Hex(0xf0a830),
		Knob:      gfx.Hex(0x3a3a3a),
		KnobFill:  gfx.Hex(0xffffff),
		Pad:       16,
		Gap:       8,
		Radius:    10,
		Row:       64,
		Fonts: Fonts{
			Tiny:  f.Regular.Sized(18),
			Small: f.Regular.Sized(22),
			Body:  f.Regular.Sized(26),
			Label: f.Bold.Sized(22),
			Title: f.Bold.Sized(32),
			Big:   f.Bold.Sized(44),
		},
	}
}

// inset shrinks a rectangle on every side.
func inset(r image.Rectangle, by int) image.Rectangle {
	return image.Rect(r.Min.X+by, r.Min.Y+by, r.Max.X-by, r.Max.Y-by)
}

// rows cuts a rectangle into n rows of height h with a gap; the rest is
// left below.
func rowsOf(r image.Rectangle, h, gap int) func() image.Rectangle {
	y := r.Min.Y
	return func() image.Rectangle {
		row := image.Rect(r.Min.X, y, r.Max.X, y+h)
		y += h + gap
		return row
	}
}
