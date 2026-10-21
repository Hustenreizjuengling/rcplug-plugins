package ui

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
)

// Spinner is RØDE's EncoderSpinner: a value in a black field with its
// caption underneath. A tap picks the field (it takes the knob's focus and
// starts adjusting, outlined in green); turning the knob then counts the
// value up or down between Min and Max, wrapping round. The page keeps the
// value until it is applied, so Value and Set are the page's.
type Spinner struct {
	Base
	Adjuster
	Label string
	Value func() int
	Set   func(int)
	Text  func(int) string // the value's text; nil shows the number (two digits when Pad)
	Min   int
	Max   int
	Pad   bool // leading zero below 10 (months, days, minutes)
	Font  *Font
	last  int
}

// NewSpinner makes a spinner over a getter and a setter.
func NewSpinner(label string, value func() int, set func(int)) *Spinner {
	return &Spinner{Label: label, Value: value, Set: set, last: -1 << 30}
}

func (s *Spinner) val() int {
	if s.Value != nil {
		return s.Value()
	}
	return 0
}

func (s *Spinner) text() string {
	v := s.val()
	if s.Text != nil {
		return s.Text(v)
	}
	if s.Pad && v >= 0 && v < 10 {
		return "0" + itoa(v)
	}
	return itoa(v)
}

func (s *Spinner) CanFocus() bool { return s.Set != nil }
func (s *Spinner) Activate() bool { return s.toggleAdjust() }

// Adjust steps the value by d, wrapping between Min and Max.
func (s *Spinner) Adjust(d int) {
	if s.Set == nil {
		return
	}
	span := s.Max - s.Min + 1
	v := s.val() + d
	if span > 0 {
		v = ((v-s.Min)%span+span)%span + s.Min
	}
	s.Set(v)
	s.MarkDirty()
}

func (s *Spinner) SetFocus(on bool) {
	if !on {
		s.stop()
	}
	s.Base.SetFocus(on)
}

// Touch: a tap takes the knob (into adjusting); a tap while adjusting lets
// go again.
func (s *Spinner) Touch(phase TouchPhase, p image.Point) bool {
	if s.Set == nil {
		return false
	}
	if phase == Up {
		adjusting := s.toggleAdjust()
		grabFocus(s, adjusting)
		s.MarkDirty()
	}
	return true
}

func (s *Spinner) Dirty() []image.Rectangle {
	if v := s.val(); v != s.last {
		s.last = v
		s.MarkDirty()
	}
	return s.Base.Dirty()
}

// field is the black value box (the caption takes the bottom 44 px).
func (s *Spinner) field() image.Rectangle {
	b := s.bounds
	if s.Label == "" {
		return b
	}
	return image.Rect(b.Min.X, b.Min.Y, b.Max.X, b.Max.Y-44)
}

func (s *Spinner) Draw(dst *gfx.Image, c *Context, clip image.Rectangle) {
	th := c.Theme
	f := s.field()
	Panel(dst, f, th.Radius, th.Black, clip)
	fnt := s.Font
	if fnt == nil {
		fnt = th.Fonts.Big
	}
	col := th.Text
	if s.Set == nil {
		col = th.TextDim
	}
	Text(dst, fnt, s.text(), f, font.Center, col, clip)
	if s.Label != "" {
		Text(dst, th.Fonts.Small, s.Label, image.Rect(s.bounds.Min.X, f.Max.Y, s.bounds.Max.X, s.bounds.Max.Y), font.Center, th.TextDim, clip)
	}
	if s.Adjusting() {
		PanelOutline(dst, f, th.Radius, 4, th.Accent, clip)
	} else if s.Focused() {
		Focus(dst, f, th.Radius, false, th, clip)
	}
}
