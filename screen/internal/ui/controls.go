package ui

import (
	"image"
	"math"
	"time"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
)

// Fader is a vertical slider with a value 0…1. Dragging it calls Change;
// the value shown comes from Value so the desk stays the source of truth.
type Fader struct {
	Base
	Adjuster
	Value    func() float64
	Change   func(float64)
	Label    string  // a short caption under the track
	Unity    float64 // a detent to mark (0…1), <0 for none
	Step     float64 // an encoder step (default 1/127)
	last     float64
	dragging bool
}

// NewFader makes a fader.
func NewFader(value func() float64, change func(float64)) *Fader {
	return &Fader{Value: value, Change: change, last: math.NaN(), Unity: -1}
}

func (f *Fader) val() float64 {
	if f.Value != nil {
		return clamp01(f.Value())
	}
	return 0
}

func (f *Fader) track() image.Rectangle {
	r := inset(f.bounds, 6)
	if f.Label != "" {
		r.Max.Y -= 34
	}
	w := min(r.Dx(), 24)
	x := r.Min.X + (r.Dx()-w)/2
	return image.Rect(x, r.Min.Y+14, x+w, r.Max.Y-14)
}

func (f *Fader) set(v float64) {
	if f.Change != nil {
		f.Change(clamp01(v))
	}
	f.MarkDirty()
}

func (f *Fader) CanFocus() bool { return f.Change != nil }
func (f *Fader) Activate() bool { return f.toggleAdjust() }
func (f *Fader) Adjust(d int) {
	step := f.Step
	if step <= 0 {
		step = 1.0 / 127
	}
	f.set(f.val() + float64(d)*step)
}
func (f *Fader) SetFocus(on bool) {
	if !on {
		f.stop()
	}
	f.Base.SetFocus(on)
}

func (f *Fader) Touch(phase TouchPhase, p image.Point) bool {
	if f.Change == nil {
		return false
	}
	tr := f.track()
	switch phase {
	case Down:
		f.dragging = true
		fallthrough
	case Move:
		if f.dragging && tr.Dy() > 0 {
			f.set(1 - float64(p.Y-tr.Min.Y)/float64(tr.Dy()))
		}
	case Up:
		f.dragging = false
	}
	return true
}

// Dirty redraws when the desk's value moved, too.
func (f *Fader) Dirty() []image.Rectangle {
	if v := f.val(); v != f.last {
		f.last = v
		f.MarkDirty()
	}
	return f.Base.Dirty()
}

func (f *Fader) Draw(dst *gfx.Image, c *Context, clip image.Rectangle) {
	t := c.Theme
	tr := f.track()
	Panel(dst, tr, tr.Dx()/2, t.Black, clip)
	PanelOutline(dst, tr, tr.Dx()/2, 1, t.Line, clip)
	v := f.val()
	y := tr.Max.Y - int(v*float64(tr.Dy()))
	Panel(dst, image.Rect(tr.Min.X, y, tr.Max.X, tr.Max.Y), tr.Dx()/2, t.Accent, clip)
	if f.Unity >= 0 {
		uy := tr.Max.Y - int(f.Unity*float64(tr.Dy()))
		Fill(dst, image.Rect(tr.Min.X-10, uy-1, tr.Min.X-3, uy+1), t.TextDim, clip)
		Fill(dst, image.Rect(tr.Max.X+3, uy-1, tr.Max.X+10, uy+1), t.TextDim, clip)
	}
	// The cap: a white pill, hollow at zero.
	cap := image.Rect(tr.Min.X-14, y-13, tr.Max.X+14, y+13)
	if v <= 0 {
		Panel(dst, cap, 13, t.Black, clip)
		PanelOutline(dst, cap, 13, 3, t.Text, clip)
	} else {
		Panel(dst, cap, 13, t.Text, clip)
		Fill(dst, image.Rect(cap.Min.X+8, y-1, cap.Max.X-8, y+1), gfx.Hex(0x9a9a9a), clip)
	}
	if f.Label != "" {
		Text(dst, t.Fonts.Small, f.Label, image.Rect(f.bounds.Min.X, f.bounds.Max.Y-32, f.bounds.Max.X, f.bounds.Max.Y), font.Center, t.TextDim, clip)
	}
	if f.Focused() {
		Focus(dst, f.bounds, t.Radius, f.Adjusting(), t, clip)
	}
}

// KnobStyle is how a knob looks.
type KnobStyle uint8

const (
	// KnobRing is RØDE's channel knob: a dark disc with a white arc around
	// it and an icon inside (Depth, Sparkle, Punch), the label underneath.
	KnobRing KnobStyle = iota
	// KnobRack is the processor rack's knob: a black knob with a green arc,
	// the name and value to its right.
	KnobRack
)

// Knob is a rotary control for a value 0…1. Dragging up or down changes it;
// so does the encoder once the knob is activated.
type Knob struct {
	Base
	Adjuster
	Value   func() float64
	Change  func(float64)
	Label   string
	Icon    string
	Style   KnobStyle
	Text    func(v float64) string // the value, shown under the label / beside the knob
	Step    float64                // an encoder step (default 1/100)
	OnLight bool                   // drawn on a light panel (dark text)
	Dim     bool                   // greyed out and inert (the inactive half of a page)
	last    float64
	lastY   int
	drag    bool
}

// NewKnob makes a knob.
func NewKnob(value func() float64, change func(float64)) *Knob {
	return &Knob{Value: value, Change: change, last: math.NaN()}
}

func (k *Knob) val() float64 {
	if k.Value != nil {
		return clamp01(k.Value())
	}
	return 0
}

func (k *Knob) set(v float64) {
	if k.Change != nil {
		k.Change(clamp01(v))
	}
	k.MarkDirty()
}

func (k *Knob) CanFocus() bool { return k.Change != nil && !k.Dim }
func (k *Knob) Activate() bool { return k.toggleAdjust() }
func (k *Knob) Adjust(d int) {
	step := k.Step
	if step <= 0 {
		step = 0.01
	}
	k.set(k.val() + float64(d)*step)
}
func (k *Knob) SetFocus(on bool) {
	if !on {
		k.stop()
	}
	k.Base.SetFocus(on)
}

func (k *Knob) Touch(phase TouchPhase, p image.Point) bool {
	if k.Change == nil || k.Dim {
		return false
	}
	switch phase {
	case Down:
		k.drag = true
		k.lastY = p.Y
	case Move:
		if k.drag {
			dv := float64(k.lastY-p.Y) / 240
			k.lastY = p.Y
			k.set(k.val() + dv)
		}
	case Up:
		k.drag = false
	}
	return true
}

func (k *Knob) Dirty() []image.Rectangle {
	if v := k.val(); v != k.last {
		k.last = v
		k.MarkDirty()
	}
	return k.Base.Dirty()
}

// disc is the knob's circle.
func (k *Knob) disc() image.Rectangle {
	r := k.bounds
	switch k.Style {
	case KnobRack:
		s := min(r.Dy(), 100)
		return image.Rect(r.Min.X, r.Min.Y+(r.Dy()-s)/2, r.Min.X+s, r.Min.Y+(r.Dy()-s)/2+s)
	default:
		if k.Label != "" {
			r.Max.Y -= 44
		}
		if k.Text != nil {
			r.Max.Y -= 30
		}
		return Square(r)
	}
}

func (k *Knob) Draw(dst *gfx.Image, c *Context, clip image.Rectangle) {
	t := c.Theme
	d := k.disc()
	cx, cy := centre(d).X, centre(d).Y
	rad := float64(d.Dx()) / 2
	v := k.val()
	const start, sweep = math.Pi * 0.75, math.Pi * 1.5
	switch k.Style {
	case KnobRack:
		// A black knob with a lighter rim and a green arc at its foot; a
		// dimmed knob is drawn in greys.
		body, rim, arc, dot := gfx.Hex(0x0e0e0e), gfx.Hex(0x3a3a3a), t.Accent, t.Text
		if k.Dim {
			body, rim, arc, dot = gfx.Hex(0x9a9a9a), gfx.Hex(0xb8b8b8), gfx.Hex(0xc8e8a0), gfx.Hex(0xdddddd)
		}
		Circle(dst, cx, cy, rad-2, body, clip)
		Ring(dst, cx, cy, rad-5, 3, rim, clip)
		Circle(dst, cx, cy, rad*0.55, gfx.Mix(body, rim, 0.3), clip)
		Arc(dst, cx, cy, rad-2, 5, start, start+sweep*math.Max(v, 0.005), arc, clip)
		// The pointer.
		a := start + sweep*v
		px, py := cx+math.Cos(a)*rad*0.5, cy+math.Sin(a)*rad*0.5
		Circle(dst, px, py, 5, dot, clip)
		tx := d.Max.X + 18
		name, val := t.Text, t.TextDim
		if k.OnLight {
			name, val = t.TextOnLight, gfx.Hex(0x555555)
		}
		if k.Dim {
			name, val = gfx.Hex(0x9a9a9a), gfx.Hex(0xaaaaaa)
		}
		if k.Label != "" {
			Text(dst, t.Fonts.Label, k.Label, image.Rect(tx, d.Min.Y, k.bounds.Max.X, d.Min.Y+d.Dy()/2), font.Left, name, clip)
		}
		if k.Text != nil {
			Text(dst, t.Fonts.Body, k.Text(v), image.Rect(tx, d.Min.Y+d.Dy()/2, k.bounds.Max.X, d.Max.Y), font.Left, val, clip)
		}
	default:
		Circle(dst, cx, cy, rad-6, gfx.Hex(0x2d2d2d), clip)
		Arc(dst, cx, cy, rad-5, 5, start, start+sweep, gfx.Hex(0x1a1a1a), clip)
		Arc(dst, cx, cy, rad-5, 5, start, start+sweep*math.Max(v, 0.003), t.KnobFill, clip)
		if k.Icon != "" {
			c.Icon(dst, k.Icon, inset(d, d.Dx()/4), t.Text)
		}
		y := d.Max.Y + 4
		if k.Label != "" {
			Text(dst, t.Fonts.Body, k.Label, image.Rect(k.bounds.Min.X-10, y, k.bounds.Max.X+10, y+40), font.Center, t.Text, clip)
			y += 40
		}
		if k.Text != nil {
			Text(dst, t.Fonts.Small, k.Text(v), image.Rect(k.bounds.Min.X-10, y, k.bounds.Max.X+10, y+30), font.Center, t.TextDim, clip)
		}
	}
	if k.Focused() {
		Focus(dst, d, d.Dx()/2, k.Adjusting(), t, clip)
	}
}

// Meter is a level bar. Level gives the display fraction 0…1 (dB-scaled by
// the caller); Peak, when set, gives the peak-hold fraction, otherwise the
// meter holds its own peak for a moment. Horizontal meters fill from the
// left, vertical ones from the bottom.
type Meter struct {
	Base
	Level     func() float64 // 0…1
	Peak      func() float64 // 0…1, optional
	Reduction func() float64 // gain reduction 0…1 from the top, optional
	Vertical  bool
	Thin      bool // no rounded ends (a strip meter)
	last      float64
	lastPeak  float64
	holdPeak  float64
	holdAt    time.Time
	shown     float64
	shownAt   time.Time
}

// MeterFall is how fast a drawn level falls after the reading dropped
// (scale fractions per second), so 20 Hz readings look smooth.
const MeterFall = 1.0 / 3

// NewMeter makes a level meter.
func NewMeter(level func() float64) *Meter { return &Meter{Level: level, Vertical: true, last: -1} }

func (m *Meter) lvl() float64 {
	target := 0.0
	if m.Level != nil {
		target = clamp01(m.Level())
	}
	now := time.Now()
	dt := now.Sub(m.shownAt).Seconds()
	if dt > 0.5 {
		dt = 0.5
	}
	m.shownAt = now
	m.shown = math.Max(target, m.shown-MeterFall*dt)
	return m.shown
}

// peak is the peak-hold fraction: the caller's, or a 1.5 s hold of the
// level with a slow fall.
func (m *Meter) peak(v float64) float64 {
	if m.Peak != nil {
		return clamp01(m.Peak())
	}
	now := time.Now()
	if v >= m.holdPeak {
		m.holdPeak, m.holdAt = v, now
	} else if since := now.Sub(m.holdAt); since > 1500*time.Millisecond {
		m.holdPeak = math.Max(v, m.holdPeak-float64(since-1500*time.Millisecond)/float64(time.Second)*0.4)
		m.holdAt = now.Add(-1500 * time.Millisecond)
	}
	return m.holdPeak
}

func (m *Meter) Dirty() []image.Rectangle {
	v := m.lvl()
	p := m.peak(v)
	// Redraw when the bar moved by a pixel.
	px := float64(max(m.bounds.Dx(), m.bounds.Dy()))
	if math.Abs(v-m.last)*px >= 1 || math.Abs(p-m.lastPeak)*px >= 1 {
		m.last, m.lastPeak = v, p
		m.MarkDirty()
	}
	return m.Base.Dirty()
}

func (m *Meter) Draw(dst *gfx.Image, c *Context, clip image.Rectangle) {
	t := c.Theme
	r := m.bounds
	rad := 0
	if !m.Thin {
		rad = min(r.Dx(), r.Dy()) / 2
	}
	Panel(dst, r, rad, t.MeterTrack, clip)
	v := m.lvl()
	p := m.peak(v)
	if m.Vertical {
		h := int(v * float64(r.Dy()))
		bar := image.Rect(r.Min.X, r.Max.Y-h, r.Max.X, r.Max.Y)
		meterPaint(dst, bar, r, true, rad, clip)
		if p > v && p > 0 {
			py := r.Max.Y - int(p*float64(r.Dy()))
			Fill(dst, image.Rect(r.Min.X, py-2, r.Max.X, py+2), meterColour(p), clip)
		}
	} else {
		w := int(v * float64(r.Dx()))
		bar := image.Rect(r.Min.X, r.Min.Y, r.Min.X+w, r.Max.Y)
		meterPaint(dst, bar, r, false, rad, clip)
		if p > v && p > 0 {
			px := r.Min.X + int(p*float64(r.Dx()))
			Fill(dst, image.Rect(px-2, r.Min.Y, px+2, r.Max.Y), meterColour(p), clip)
		}
	}
	if m.Reduction != nil {
		gr := clamp01(m.Reduction())
		if m.Vertical {
			h := int(gr * float64(r.Dy()))
			Fill(dst, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+h), t.Warn.WithAlpha(0.7), clip)
		}
	}
}

// meterStops is the level colouring: green up to −9 dBFS, yellow to −3,
// red above (on a 60 dB scale).
var meterStops = []gfx.Stop{
	{Offset: 0, Color: gfx.Hex(0x2fd12f)}, {Offset: 0.84, Color: gfx.Hex(0x2fd12f)},
	{Offset: 0.86, Color: gfx.Hex(0xf0d020)}, {Offset: 0.94, Color: gfx.Hex(0xf0d020)},
	{Offset: 0.96, Color: gfx.Hex(0xe82020)}, {Offset: 1, Color: gfx.Hex(0xe82020)},
}

// meterColour is the colour of the meter at fraction v.
func meterColour(v float64) gfx.Color {
	switch {
	case v >= 0.95:
		return gfx.Hex(0xe82020)
	case v >= 0.85:
		return gfx.Hex(0xf0d020)
	}
	return gfx.Hex(0x2fd12f)
}

// meterPaint fills bar with the level colouring over the full extent full.
func meterPaint(dst *gfx.Image, bar, full image.Rectangle, vertical bool, rad int, clip image.Rectangle) {
	if bar.Empty() {
		return
	}
	var p0, p1 gfx.Pt
	if vertical {
		p0 = gfx.Pt{X: 0, Y: float64(full.Max.Y)}
		p1 = gfx.Pt{X: 0, Y: float64(full.Min.Y)}
	} else {
		p0 = gfx.Pt{X: float64(full.Min.X)}
		p1 = gfx.Pt{X: float64(full.Max.X)}
	}
	g := gfx.NewLinear(p0, p1, gfx.Identity, meterStops)
	path := &gfx.Path{}
	if rad > 0 {
		path.RoundRect(float64(full.Min.X), float64(full.Min.Y), float64(full.Dx()), float64(full.Dy()), float64(rad), float64(rad))
	} else {
		path.Rect(float64(bar.Min.X), float64(bar.Min.Y), float64(bar.Dx()), float64(bar.Dy()))
	}
	dst.Fill(path, gfx.Identity, gfx.NonZero, g, clip.Intersect(bar))
}

// DbFrac maps a linear level (0…1 = −∞…0 dBFS) onto a 60 dB meter scale.
func DbFrac(v float64) float64 {
	if v <= 0 {
		return 0
	}
	return clamp01((20*math.Log10(v) + 60) / 60)
}

// MeterScale draws the dBFS scale of a 60 dB meter beside r: "0 15 30 45
// 60" with dot ticks every 3 dB, the labels to the left of x.
func MeterScale(dst *gfx.Image, r image.Rectangle, x int, c *Context, clip image.Rectangle) {
	t := c.Theme
	for db := 0; db <= 60; db += 3 {
		y := r.Min.Y + int(float64(db)/60*float64(r.Dy()))
		if db%15 == 0 {
			Text(dst, t.Fonts.Tiny, itoa(db), image.Rect(x-40, y-14, x-4, y+14), font.Right, t.TextDim, clip)
			Fill(dst, image.Rect(x-2, y-1, x+3, y+1), t.TextDim, clip)
		} else {
			Fill(dst, image.Rect(x, y-1, x+2, y+1), gfx.Hex(0x707070), clip)
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [8]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
