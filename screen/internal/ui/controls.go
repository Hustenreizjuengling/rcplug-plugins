package ui

import (
	"image"
	"math"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
)

// Fader is a vertical slider with a value 0…1. Dragging it calls Change;
// the value shown comes from Value so the desk stays the source of truth.
type Fader struct {
	Base
	Value    func() float64
	Change   func(float64)
	Label    string // a short caption under the track
	last     float64
	dragging bool
}

// NewFader makes a fader.
func NewFader(value func() float64, change func(float64)) *Fader {
	return &Fader{Value: value, Change: change, last: math.NaN()}
}

func (f *Fader) val() float64 {
	if f.Value != nil {
		return clamp01(f.Value())
	}
	return 0
}

func (f *Fader) track() image.Rectangle {
	r := inset(f.bounds, 4)
	if f.Label != "" {
		r.Max.Y -= 22
	}
	w := min(r.Dx(), 22)
	x := r.Min.X + (r.Dx()-w)/2
	return image.Rect(x, r.Min.Y, x+w, r.Max.Y)
}

func (f *Fader) Touch(phase TouchPhase, p image.Point) bool {
	tr := f.track()
	switch phase {
	case Down:
		f.dragging = true
		fallthrough
	case Move:
		if f.dragging {
			v := 1 - float64(p.Y-tr.Min.Y)/float64(tr.Dy())
			if f.Change != nil {
				f.Change(clamp01(v))
			}
			f.MarkDirty()
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
	Panel(dst, tr, tr.Dx()/2, t.Knob, clip)
	v := f.val()
	fillTop := tr.Max.Y - int(v*float64(tr.Dy()))
	Panel(dst, image.Rect(tr.Min.X, fillTop, tr.Max.X, tr.Max.Y), tr.Dx()/2, t.Accent, clip)
	// The handle: a cap wider than the track, with a centre line.
	hy := tr.Max.Y - int(v*float64(tr.Dy()))
	hh, over := 18, 10
	handle := image.Rect(tr.Min.X-over, hy-hh/2, tr.Max.X+over, hy+hh/2)
	Panel(dst, handle, 5, gfx.Hex(0xf2f2f2), clip)
	Fill(dst, image.Rect(handle.Min.X+4, hy-1, handle.Max.X-4, hy+1), t.Line, clip)
	if f.Label != "" {
		Text(dst, t.Fonts.Tiny, f.Label, image.Rect(f.bounds.Min.X, tr.Max.Y+2, f.bounds.Max.X, f.bounds.Max.Y), font.Center, t.TextDim, clip)
	}
}

// Knob is a rotary control for a value 0…1, drawn as an arc. Dragging up or
// down changes it.
type Knob struct {
	Base
	Value  func() float64
	Change func(float64)
	Label  string
	Text   func(v float64) string // the value shown in the middle
	last   float64
	lastY  int
	drag   bool
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

func (k *Knob) Touch(phase TouchPhase, p image.Point) bool {
	switch phase {
	case Down:
		k.drag = true
		k.lastY = p.Y
	case Move:
		if k.drag {
			dv := float64(k.lastY-p.Y) / 200
			k.lastY = p.Y
			if k.Change != nil {
				k.Change(clamp01(k.val() + dv))
			}
			k.MarkDirty()
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

func (k *Knob) Draw(dst *gfx.Image, c *Context, clip image.Rectangle) {
	t := c.Theme
	r := k.bounds
	if k.Label != "" {
		r.Max.Y -= 22
	}
	cx, cy := centre(r).X, centre(r).Y
	rad := float64(min(r.Dx(), r.Dy()))/2 - 6
	const start, sweep = math.Pi * 0.75, math.Pi * 1.5
	// The track.
	bg := &gfx.Path{}
	bg.Arc(cx, cy, rad, start, start+sweep)
	dst.StrokePath(bg, gfx.Identity, gfx.Stroke{Width: 6, Cap: gfx.RoundCap}, gfx.Solid(t.Knob), clip)
	// The filled part.
	v := k.val()
	fg := &gfx.Path{}
	fg.Arc(cx, cy, rad, start, start+sweep*v)
	dst.StrokePath(fg, gfx.Identity, gfx.Stroke{Width: 6, Cap: gfx.RoundCap}, gfx.Solid(t.Accent), clip)
	if k.Text != nil {
		Text(dst, t.Fonts.Small, k.Text(v), r, font.Center, t.Text, clip)
	}
	if k.Label != "" {
		Text(dst, t.Fonts.Tiny, k.Label, image.Rect(k.bounds.Min.X, r.Max.Y, k.bounds.Max.X, k.bounds.Max.Y), font.Center, t.TextDim, clip)
	}
}

// Meter is a level bar (0…1), drawn with a green-to-red gradient and a cap
// that shrinks. It is read every frame from Level.
type Meter struct {
	Base
	Level     func() float64 // 0…1
	Reduction func() float64 // gain reduction 0…1 from the top, optional
	Vertical  bool
	last      float64
}

// NewMeter makes a level meter.
func NewMeter(level func() float64) *Meter { return &Meter{Level: level, Vertical: true, last: -1} }

func (m *Meter) lvl() float64 {
	if m.Level != nil {
		return clamp01(m.Level())
	}
	return 0
}

func (m *Meter) Dirty() []image.Rectangle {
	if v := m.lvl(); v != m.last {
		m.last = v
		m.MarkDirty()
	}
	return m.Base.Dirty()
}

func (m *Meter) Draw(dst *gfx.Image, c *Context, clip image.Rectangle) {
	t := c.Theme
	r := m.bounds
	Fill(dst, r, gfx.Hex(0x0a0a0a), clip)
	v := m.lvl()
	if m.Vertical {
		h := int(v * float64(r.Dy()))
		bar := image.Rect(r.Min.X, r.Max.Y-h, r.Max.X, r.Max.Y)
		meterPaint(dst, bar, r, true, clip)
	} else {
		w := int(v * float64(r.Dx()))
		bar := image.Rect(r.Min.X, r.Min.Y, r.Min.X+w, r.Max.Y)
		meterPaint(dst, bar, r, false, clip)
	}
	if m.Reduction != nil {
		gr := clamp01(m.Reduction())
		h := int(gr * float64(r.Dy()))
		Fill(dst, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+h), t.Warn.WithAlpha(0.6), clip)
	}
}

// meterPaint fills bar with a green→yellow→red gradient over the full
// extent full.
func meterPaint(dst *gfx.Image, bar, full image.Rectangle, vertical bool, clip image.Rectangle) {
	stops := []gfx.Stop{{Offset: 0, Color: gfx.Hex(0x3ec46d)}, {Offset: 0.7, Color: gfx.Hex(0xd8d84a)}, {Offset: 0.88, Color: gfx.Hex(0xe5322d)}}
	var p0, p1 gfx.Pt
	if vertical {
		p0 = gfx.Pt{X: 0, Y: float64(full.Max.Y)}
		p1 = gfx.Pt{X: 0, Y: float64(full.Min.Y)}
	} else {
		p0 = gfx.Pt{X: float64(full.Min.X)}
		p1 = gfx.Pt{X: float64(full.Max.X)}
	}
	g := gfx.NewLinear(p0, p1, gfx.Identity, stops)
	path := &gfx.Path{}
	path.Rect(float64(bar.Min.X), float64(bar.Min.Y), float64(bar.Dx()), float64(bar.Dy()))
	dst.Fill(path, gfx.Identity, gfx.NonZero, g, clip.Intersect(bar))
}
