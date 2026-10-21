package pages

import (
	"image"
	"math"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// valueBox is RØDE's outlined value field (the MIDI page's Type, Control,
// Channel, On, Off; the fade times; the ducking depth): a caption above a
// large value in a black box with a light outline. A tap gives it the knob
// (the outline turns green and the knob changes the value); a vertical drag
// changes it too. Text renders the value; Step moves it by whole steps.
type valueBox struct {
	ui.Base
	Label     string
	Text      func() string
	Step      func(delta int)
	Enabled   func() bool
	last      string
	lastY     int
	drag      bool
	acc       int
	adjusting bool
}

func newValueBox(label string, text func() string, step func(delta int)) *valueBox {
	return &valueBox{Label: label, Text: text, Step: step}
}

func (v *valueBox) enabled() bool { return v.Enabled == nil || v.Enabled() }

func (v *valueBox) CanFocus() bool { return v.Step != nil }
func (v *valueBox) Activate() bool {
	if !v.enabled() {
		return false
	}
	v.adjusting = !v.adjusting
	v.MarkDirty()
	return v.adjusting
}

// Adjusting reports whether the knob is changing the value.
func (v *valueBox) Adjusting() bool { return v.adjusting }
func (v *valueBox) Adjust(d int) {
	if v.Step == nil || !v.enabled() {
		return
	}
	for ; d > 0; d-- {
		v.Step(1)
	}
	for ; d < 0; d++ {
		v.Step(-1)
	}
	v.MarkDirty()
}
func (v *valueBox) SetFocus(on bool) {
	if !on {
		v.adjusting = false
	}
	v.Base.SetFocus(on)
}

func (v *valueBox) Touch(phase ui.TouchPhase, p image.Point) bool {
	if v.Step == nil {
		return false
	}
	switch phase {
	case ui.Down:
		v.drag, v.lastY, v.acc = true, p.Y, 0
	case ui.Move:
		if v.drag && v.enabled() {
			v.acc += v.lastY - p.Y
			v.lastY = p.Y
			for ; v.acc >= 12; v.acc -= 12 {
				v.Step(1)
			}
			for ; v.acc <= -12; v.acc += 12 {
				v.Step(-1)
			}
			v.MarkDirty()
		}
	case ui.Up:
		v.drag = false
	}
	return true
}

func (v *valueBox) Dirty() []image.Rectangle {
	if s := v.Text(); s != v.last {
		v.last = s
		v.MarkDirty()
	}
	return v.Base.Dirty()
}

func (v *valueBox) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	r := v.Bounds()
	ui.Panel(dst, r, 14, th.Black, clip)
	outline := gfx.Hex(0xd0d0d0)
	switch {
	case !v.enabled():
		outline = th.TextMuted
	case v.Adjusting():
		outline = th.Accent
	}
	ui.PanelOutline(dst, r, 14, 3, outline, clip)
	text := th.Text
	if !v.enabled() {
		text = th.TextMuted
	}
	ui.Text(dst, th.Fonts.Body, v.Label, image.Rect(r.Min.X, r.Min.Y+10, r.Max.X, r.Min.Y+50), font.Center, text, clip)
	ui.Text(dst, th.Fonts.Huge, v.Text(), image.Rect(r.Min.X, r.Min.Y+44, r.Max.X, r.Max.Y-6), font.Center, text, clip)
	if v.Focused() && !v.Adjusting() {
		ui.Focus(dst, r, 14, false, th, clip)
	}
}

// lightButton is RØDE's wide option button that is light when chosen and
// dark otherwise (Latching, Beep Tone, Custom, the Default | Custom pair).
func lightButton(label string, on func() bool, tap func()) (*ui.Button, ui.View) {
	b := ui.NewButton(label, tap)
	b.Toggle = true
	return b, &updater{get: func() { b.SetOn(on()) }}
}

// recDot draws the red record dot of RØDE's record buttons.
func recDot(dst *gfx.Image, r image.Rectangle, clip image.Rectangle) {
	cx, cy := float64(r.Min.X+r.Max.X)/2, float64(r.Min.Y+r.Max.Y)/2
	rad := math.Min(float64(r.Dx()), float64(r.Dy())) / 2
	ui.Circle(dst, cx, cy, rad, gfx.Hex(0xffffff), clip)
	ui.Circle(dst, cx, cy, rad-4, gfx.Hex(0xe8201c), clip)
}

// roundIconButton is a round button with an icon inside (RØDE's input and
// effect buttons): a ring in the accent colour when on.
type roundIconButton struct {
	ui.Base
	Icon    string
	Label   string
	On      func() bool
	Tap     func()
	Fill    gfx.Color
	pressed bool
	last    bool
}

func newRoundIconButton(icon string, on func() bool, tap func()) *roundIconButton {
	return &roundIconButton{Icon: icon, On: on, Tap: tap}
}

func (b *roundIconButton) CanFocus() bool { return b.Tap != nil }
func (b *roundIconButton) Activate() bool {
	if b.Tap != nil {
		b.Tap()
	}
	return false
}

func (b *roundIconButton) Touch(phase ui.TouchPhase, p image.Point) bool {
	switch phase {
	case ui.Down:
		b.pressed = true
		b.MarkDirty()
	case ui.Up:
		if b.pressed && b.Tap != nil {
			b.Tap()
		}
		b.pressed = false
		b.MarkDirty()
	}
	return true
}

func (b *roundIconButton) on() bool { return b.On != nil && b.On() }

func (b *roundIconButton) Dirty() []image.Rectangle {
	if v := b.on(); v != b.last {
		b.last = v
		b.MarkDirty()
	}
	return b.Base.Dirty()
}

func (b *roundIconButton) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	r := ui.Square(b.Bounds())
	cx, cy := float64(r.Min.X+r.Max.X)/2, float64(r.Min.Y+r.Max.Y)/2
	rad := float64(r.Dx()) / 2
	fill := gfx.Hex(0x1a1a1a)
	if b.Fill.A() != 0 {
		fill = b.Fill
	}
	if b.pressed {
		fill = gfx.Mix(fill, th.Text, 0.2)
	}
	ui.Circle(dst, cx, cy, rad, fill, clip)
	ring := gfx.Hex(0x0a0a0a)
	if b.on() {
		ring = th.Accent
	}
	ui.Ring(dst, cx, cy, rad-3, 5, ring, clip)
	if b.Icon != "" {
		ic := ui.Inset(r, r.Dx()/4)
		if b.Label != "" {
			ic = image.Rect(r.Min.X+r.Dx()/5, r.Min.Y+r.Dy()/4, r.Min.X+r.Dx()*11/20, r.Max.Y-r.Dy()/4)
			ui.Text(dst, th.Fonts.Label, b.Label, image.Rect(ic.Max.X, r.Min.Y, r.Max.X-r.Dx()/8, r.Max.Y), font.Center, th.Text, clip)
		}
		c.Icon(dst, b.Icon, ic, th.Text)
	}
	if b.Focused() {
		ui.Focus(dst, r, r.Dx()/2, false, th, clip)
	}
}

// timeText formats seconds as "mm:ss.t", as RØDE's pad recorder shows its
// time.
func timeText(sec float64) string {
	if sec < 0 {
		sec = 0
	}
	m := int(sec) / 60
	s := sec - float64(m*60)
	return pad2(m) + ":" + pad2(int(s)) + "." + intText(int64(math.Mod(s*10, 10)))
}

func pad2(n int) string {
	if n < 10 {
		return "0" + intText(int64(n))
	}
	return intText(int64(n))
}
