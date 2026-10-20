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

// ButtonStyle is how a button is drawn.
type ButtonStyle uint8

const (
	// StyleCard is a grey gradient card with the label (and icon) inside.
	StyleCard ButtonStyle = iota
	// StyleTile is a square gradient card holding the icon, with the label
	// under the card on the page (menus, routing, pads).
	StyleTile
	// StyleLight is the light card of a chosen thing, dark text.
	StyleLight
	// StyleRound is a circle: an icon or a short word (X, tick, REC).
	StyleRound
	// StylePlain is just the icon or text, no card (header controls).
	StylePlain
)

// Button is a tappable card with an optional icon and label.
type Button struct {
	Base
	Label   string
	Icon    string
	OnTap   func()
	Toggle  bool // stays on
	On      bool
	Enabled bool
	Style   ButtonStyle
	Fill    gfx.Color // overrides the card colour (alpha 0: the style's)
	Tint    gfx.Color // overrides the icon/text colour (alpha 0: the style's)
	Font    *Font     // overrides the label font
	Big     bool      // StyleTile: a large icon
	Sub     string    // a second, dimmer line under the label (StyleCard)

	pressed bool
}

// NewButton makes a button.
func NewButton(label string, onTap func()) *Button {
	return &Button{Label: label, OnTap: onTap, Enabled: true}
}

// NewTile makes an icon tile with its label underneath.
func NewTile(label, icon string, onTap func()) *Button {
	b := NewButton(label, onTap)
	b.Icon, b.Style = icon, StyleTile
	return b
}

// SetOn changes a toggle button's state.
func (b *Button) SetOn(on bool) {
	if on != b.On {
		b.On = on
		b.MarkDirty()
	}
}

// SetLabel changes the caption.
func (b *Button) SetLabel(s string) {
	if s != b.Label {
		b.Label = s
		b.MarkDirty()
	}
}

func (b *Button) CanFocus() bool { return b.Enabled }

func (b *Button) Activate() bool {
	b.tap()
	return false
}

func (b *Button) tap() {
	if b.Toggle {
		b.On = !b.On
	}
	b.MarkDirty()
	if b.OnTap != nil {
		b.OnTap()
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
			b.tap()
		}
	}
	return true
}

// card is the tile's card area (StyleTile leaves room for the label below).
func (b *Button) card() image.Rectangle {
	r := b.bounds
	if b.Style == StyleTile && b.Label != "" {
		r.Max.Y -= 48
	}
	return r
}

func (b *Button) Draw(dst *gfx.Image, c *Context, clip image.Rectangle) {
	t := c.Theme
	fg := t.Text
	if b.Tint.A() != 0 {
		fg = b.Tint
	}
	card := b.card()
	light := b.Style == StyleLight || (b.On && b.Toggle && b.Style != StyleRound)
	switch {
	case !b.Enabled:
		fg = t.TextMuted
	case light:
		fg = t.TextOnLight
	}
	switch b.Style {
	case StylePlain:
		if b.pressed {
			Panel(dst, card, t.Radius, t.PanelHi.WithAlpha(0.5), clip)
		}
	case StyleRound:
		rr := Square(card)
		cx, cy := centre(rr).X, centre(rr).Y
		fill := t.CardTop
		if b.Fill.A() != 0 {
			fill = b.Fill
		}
		if b.On {
			fill = t.Accent
		}
		if b.pressed {
			fill = gfx.Mix(fill, t.Text, 0.2)
		}
		Circle(dst, cx, cy, float64(rr.Dx())/2, fill, clip)
	default:
		switch {
		case b.pressed:
			Panel(dst, card, t.Radius, t.PanelHi, clip)
		case light:
			LightCard(dst, card, t, clip)
		case b.Fill.A() != 0:
			Panel(dst, card, t.Radius, b.Fill, clip)
		case !b.Enabled:
			Panel(dst, card, t.Radius, t.Panel, clip)
		default:
			Card(dst, card, t, clip)
		}
	}
	f := t.Fonts.Body
	if b.Font != nil {
		f = b.Font
	}
	switch b.Style {
	case StyleTile:
		ic := inset(card, 22)
		if b.Big {
			ic = inset(card, 14)
		}
		if b.Icon != "" {
			c.Icon(dst, b.Icon, Square(ic), fg)
		}
		if b.Label != "" {
			// The caption must fit the tile's width (the tile is clipped to its
			// bounds); a long one takes a smaller face, then an ellipsis.
			lr := image.Rect(b.bounds.Min.X, card.Max.Y, b.bounds.Max.X, b.bounds.Max.Y)
			lf := f
			for _, smaller := range []*Font{t.Fonts.Small, t.Fonts.Tiny} {
				if TextWidth(lf, b.Label) <= lr.Dx() {
					break
				}
				lf = smaller
			}
			Text(dst, lf, b.Label, lr, font.Center, t.Text, clip)
		}
	case StyleRound:
		rr := Square(card)
		if b.Icon != "" {
			c.Icon(dst, b.Icon, inset(rr, rr.Dx()/4), fg)
		} else {
			Text(dst, f, b.Label, rr, font.Center, fg, clip)
		}
	default:
		r := inset(card, t.Gap)
		if b.Icon != "" && b.Label == "" {
			c.Icon(dst, b.Icon, Square(inset(card, 14)), fg)
			break
		}
		if b.Icon != "" {
			side := min(r.Dy(), 48)
			ir := image.Rect(r.Min.X, r.Min.Y+(r.Dy()-side)/2, r.Min.X+side, r.Min.Y+(r.Dy()-side)/2+side)
			c.Icon(dst, b.Icon, ir, fg)
			r.Min.X = ir.Max.X + t.Gap
			if b.Sub == "" {
				Text(dst, f, b.Label, r, font.Left, fg, clip)
				break
			}
		}
		if b.Sub != "" {
			h := r.Dy() / 2
			Text(dst, f, b.Label, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+h), font.Left, fg, clip)
			Text(dst, t.Fonts.Small, b.Sub, image.Rect(r.Min.X, r.Min.Y+h, r.Max.X, r.Max.Y), font.Left, t.TextDim, clip)
		} else {
			Text(dst, f, b.Label, r, font.Center, fg, clip)
		}
	}
	if b.Focused() {
		fr := card
		rad := t.Radius
		if b.Style == StyleRound {
			fr = Square(card)
			rad = fr.Dx() / 2
		}
		Focus(dst, fr, rad, false, t, clip)
	}
}

// Toggle is a switch: a pill that slides, as on RØDE's settings pages.
type Toggle struct {
	Base
	On      func() bool
	Set     func(bool)
	Label   string // a caption to the left of the pill (optional)
	value   bool
	pressed bool
}

// NewToggle makes a switch bound to getter/setter.
func NewToggle(on func() bool, set func(bool)) *Toggle {
	return &Toggle{On: on, Set: set}
}

func (t *Toggle) CanFocus() bool { return true }
func (t *Toggle) Activate() bool {
	t.flip()
	return false
}

func (t *Toggle) flip() {
	if t.Set != nil {
		t.Set(!t.state())
	}
	t.MarkDirty()
}

func (t *Toggle) Touch(phase TouchPhase, p image.Point) bool {
	switch phase {
	case Down:
		t.pressed = true
	case Up:
		if t.pressed {
			t.flip()
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

// Dirty redraws when the bound value changed underneath.
func (t *Toggle) Dirty() []image.Rectangle {
	if v := t.state(); v != t.value {
		t.value = v
		t.MarkDirty()
	}
	return t.Base.Dirty()
}

// pill is the switch itself, at the right of the bounds.
func (t *Toggle) pill() image.Rectangle {
	h := min(t.bounds.Dy(), 64)
	w := h*2 + h/3
	x := t.bounds.Max.X - w
	y := t.bounds.Min.Y + (t.bounds.Dy()-h)/2
	return image.Rect(x, y, x+w, y+h)
}

func (t *Toggle) Draw(dst *gfx.Image, c *Context, clip image.Rectangle) {
	th := c.Theme
	on := t.state()
	r := t.pill()
	h := r.Dy()
	if t.Label != "" {
		Text(dst, th.Fonts.Body, t.Label, image.Rect(t.bounds.Min.X, t.bounds.Min.Y, r.Min.X-16, t.bounds.Max.Y), font.Left, th.Text, clip)
	}
	track := th.Black
	if on {
		track = th.Accent
	}
	Panel(dst, r, h/2, track, clip)
	PanelOutline(dst, r, h/2, 2, th.Line, clip)
	kd := h - 10
	kx := r.Min.X + 5
	if on {
		kx = r.Max.X - 5 - kd
	}
	Circle(dst, float64(kx)+float64(kd)/2, centre(r).Y, float64(kd)/2, th.Text, clip)
	if t.Focused() {
		Focus(dst, r, h/2, false, th, clip)
	}
}

// RingToggle is RØDE's round on/off: a ring that is green when on, with an
// optional caption to its right ("Processing").
type RingToggle struct {
	Base
	On      func() bool
	Set     func(bool)
	Label   string
	value   bool
	pressed bool
}

// NewRingToggle makes a ring toggle.
func NewRingToggle(label string, on func() bool, set func(bool)) *RingToggle {
	return &RingToggle{Label: label, On: on, Set: set}
}

func (t *RingToggle) state() bool {
	if t.On != nil {
		return t.On()
	}
	return t.value
}

func (t *RingToggle) CanFocus() bool { return true }
func (t *RingToggle) Activate() bool {
	if t.Set != nil {
		t.Set(!t.state())
	}
	t.MarkDirty()
	return false
}

func (t *RingToggle) Touch(phase TouchPhase, p image.Point) bool {
	switch phase {
	case Down:
		t.pressed = true
	case Up:
		if t.pressed {
			t.Activate()
		}
		t.pressed = false
	}
	return true
}

func (t *RingToggle) Dirty() []image.Rectangle {
	if v := t.state(); v != t.value {
		t.value = v
		t.MarkDirty()
	}
	return t.Base.Dirty()
}

func (t *RingToggle) Draw(dst *gfx.Image, c *Context, clip image.Rectangle) {
	th := c.Theme
	d := min(t.bounds.Dy(), 64)
	cx := float64(t.bounds.Min.X) + float64(d)/2
	cy := centre(t.bounds).Y
	Circle(dst, cx, cy, float64(d)/2-3, th.Panel, clip)
	col := th.Line
	if t.state() {
		col = th.Accent
	}
	Ring(dst, cx, cy, float64(d)/2-4, 5, col, clip)
	switch {
	case len([]rune(t.Label)) == 1:
		Text(dst, th.Fonts.Label, t.Label, image.Rect(int(cx)-d/2, int(cy)-d/2, int(cx)+d/2, int(cy)+d/2), font.Center, th.Text, clip)
	case t.Label != "":
		Text(dst, th.Fonts.Body, t.Label, image.Rect(t.bounds.Min.X+d+18, t.bounds.Min.Y, t.bounds.Max.X, t.bounds.Max.Y), font.Left, th.Text, clip)
	}
	if t.Focused() {
		Focus(dst, t.bounds, t.bounds.Dy()/2, false, th, clip)
	}
}

// PanelView is a plain card: flat, or a gradient, with an optional outline.
type PanelView struct {
	Base
	Fill     gfx.Color
	Radius   int // -1: the theme's
	Outline  gfx.Color
	Gradient bool // the theme's card gradient instead of Fill
	Light    bool // the light card
}

// NewPanelView makes a card filled with col.
func NewPanelView(col gfx.Color) *PanelView { return &PanelView{Fill: col, Radius: -1} }

// NewCard makes a card in the theme's gradient.
func NewCard() *PanelView { return &PanelView{Radius: -1, Gradient: true} }

func (p *PanelView) Draw(dst *gfx.Image, c *Context, clip image.Rectangle) {
	rad := p.Radius
	if rad < 0 {
		rad = c.Theme.Radius
	}
	switch {
	case p.Light:
		Gradient(dst, p.bounds, rad, c.Theme.LightTop, c.Theme.LightBottom, clip)
	case p.Gradient:
		Gradient(dst, p.bounds, rad, c.Theme.CardTop, c.Theme.CardBottom, clip)
	default:
		Panel(dst, p.bounds, rad, p.Fill, clip)
	}
	if p.Outline.A() != 0 {
		PanelOutline(dst, p.bounds, rad, 2, p.Outline, clip)
	}
}

// Spacer is an empty view (it draws nothing and takes no touch).
type Spacer struct{ Base }

func (*Spacer) Draw(*gfx.Image, *Context, image.Rectangle) {}
