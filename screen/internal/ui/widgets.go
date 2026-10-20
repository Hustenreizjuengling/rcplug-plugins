package ui

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
)

// Label shows a line of text.
type Label struct {
	Base
	Text  string
	Font  *Font
	Color gfx.Color
	Align font.Align
}

// NewLabel makes a label.
func NewLabel(text string, f *Font, col gfx.Color, a font.Align) *Label {
	return &Label{Text: text, Font: f, Color: col, Align: a}
}

// Set changes the text and redraws if it changed.
func (l *Label) Set(s string) {
	if s != l.Text {
		l.Text = s
		l.MarkDirty()
	}
}

func (l *Label) Draw(dst *gfx.Image, c *Context, clip image.Rectangle) {
	Text(dst, l.Font, l.Text, l.bounds, l.Align, l.Color, clip)
}

// Icon shows one of the device's icons, in a colour or in its own colours.
type Icon struct {
	Base
	Name string
	Tint gfx.Color // alpha 0: own colours
}

// NewIcon makes an icon view.
func NewIcon(name string, tint gfx.Color) *Icon { return &Icon{Name: name, Tint: tint} }

func (i *Icon) Draw(dst *gfx.Image, c *Context, clip image.Rectangle) {
	c.Icon(dst, i.Name, i.bounds, i.Tint)
}

// Button is a tappable card with an optional icon and label.
type Button struct {
	Base
	Label      string
	Icon       string
	OnTap      func()
	Toggle     bool // stays on
	On         bool
	Enabled    bool
	TileLayout bool      // icon centred above the label (a menu tile)
	Fill       gfx.Color // override the card colour (alpha 0: the theme's)

	pressed bool
}

// NewButton makes a button.
func NewButton(label string, onTap func()) *Button {
	return &Button{Label: label, OnTap: onTap, Enabled: true}
}

// SetOn changes a toggle button's state.
func (b *Button) SetOn(on bool) {
	if on != b.On {
		b.On = on
		b.MarkDirty()
	}
}

func (b *Button) Touch(phase TouchPhase, p image.Point) bool {
	if !b.Enabled {
		return false
	}
	switch phase {
	case Down:
		b.pressed = true
		b.MarkDirty()
	case Up:
		if b.pressed {
			b.pressed = false
			if b.Toggle {
				b.On = !b.On
			}
			b.MarkDirty()
			if b.OnTap != nil {
				b.OnTap()
			}
		}
	}
	return true
}

func (b *Button) Draw(dst *gfx.Image, c *Context, clip image.Rectangle) {
	t := c.Theme
	fill := t.Panel
	switch {
	case !b.Enabled:
		fill = t.Bg
	case b.pressed:
		fill = t.PanelHi
	case b.On:
		fill = t.Accent
	case b.Fill.A() != 0:
		fill = b.Fill
	}
	Panel(dst, b.bounds, t.Radius, fill, clip)
	text := t.Text
	if !b.Enabled {
		text = t.TextMuted
	} else if b.On {
		text = gfx.Hex(0x101010)
	}
	r := inset(b.bounds, t.Gap)
	if b.TileLayout {
		// Icon centred above the label.
		side := min(min(r.Dx(), r.Dy()-24), 56)
		if b.Icon != "" {
			ir := image.Rect(r.Min.X+(r.Dx()-side)/2, r.Min.Y+8, r.Min.X+(r.Dx()-side)/2+side, r.Min.Y+8+side)
			c.Icon(dst, b.Icon, ir, text)
		}
		Text(dst, t.Fonts.Small, b.Label, image.Rect(r.Min.X, r.Max.Y-30, r.Max.X, r.Max.Y), font.Center, text, clip)
		return
	}
	if b.Icon != "" {
		side := min(r.Dy(), 40)
		ir := image.Rect(r.Min.X, r.Min.Y+(r.Dy()-side)/2, r.Min.X+side, r.Min.Y+(r.Dy()-side)/2+side)
		tint := text
		if b.Icon != "" && b.Label == "" {
			ir = image.Rect(r.Min.X+(r.Dx()-side)/2, ir.Min.Y, r.Min.X+(r.Dx()-side)/2+side, ir.Max.Y)
		}
		c.Icon(dst, b.Icon, ir, tint)
		r.Min.X = ir.Max.X + t.Gap
	}
	if b.Label != "" {
		Text(dst, t.Fonts.Label, b.Label, r, alignFor(b), text, clip)
	}
}

func alignFor(b *Button) font.Align {
	if b.Icon != "" {
		return font.Left
	}
	return font.Center
}

// Toggle is a switch (a pill that slides).
type Toggle struct {
	Base
	On      func() bool
	Set     func(bool)
	value   bool
	pressed bool
}

// NewToggle makes a switch bound to getter/setter.
func NewToggle(on func() bool, set func(bool)) *Toggle {
	return &Toggle{On: on, Set: set}
}

func (t *Toggle) Touch(phase TouchPhase, p image.Point) bool {
	switch phase {
	case Down:
		t.pressed = true
	case Up:
		if t.pressed && t.Set != nil {
			t.Set(!t.state())
			t.MarkDirty()
		}
		t.pressed = false
	}
	return true
}

func (t *Toggle) state() bool {
	if t.On != nil {
		return t.On()
	}
	return t.value
}

func (t *Toggle) Draw(dst *gfx.Image, c *Context, clip image.Rectangle) {
	th := c.Theme
	on := t.state()
	// A pill the height of the box, up to 72 wide.
	h := min(t.bounds.Dy(), 36)
	w := h * 2
	x := t.bounds.Max.X - w
	y := t.bounds.Min.Y + (t.bounds.Dy()-h)/2
	track := th.Line
	if on {
		track = th.Accent
	}
	r := image.Rect(x, y, x+w, y+h)
	Panel(dst, r, h/2, track, clip)
	knob := x + 2
	if on {
		knob = x + w - h + 2
	}
	Panel(dst, image.Rect(knob, y+2, knob+h-4, y+h-2), (h-4)/2, gfx.Hex(0xffffff), clip)
}

// PanelView is a plain rounded card.
type PanelView struct {
	Base
	Fill    gfx.Color
	Radius  int // -1: the theme's
	Outline gfx.Color
}

// NewPanelView makes a card filled with col.
func NewPanelView(col gfx.Color) *PanelView { return &PanelView{Fill: col, Radius: -1} }

func (p *PanelView) Draw(dst *gfx.Image, c *Context, clip image.Rectangle) {
	rad := p.Radius
	if rad < 0 {
		rad = c.Theme.Radius
	}
	Panel(dst, p.bounds, rad, p.Fill, clip)
	if p.Outline.A() != 0 {
		PanelOutline(dst, p.bounds, rad, 2, p.Outline, clip)
	}
}
