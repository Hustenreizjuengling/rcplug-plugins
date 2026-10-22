package pages

import (
	"image"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// canvasView draws a plugin's own picture: the shapes of a canvas control,
// in the plugin's units, scaled to fit a dark inset. A plugin describes its
// picture a few times a second; shapes that carry an id glide from where
// the previous description had them to where the new one has them over
// the time between the two, so a radar's blip or a meter's bar moves as
// smoothly as the screen draws, whatever the plugin's rate.
type canvasView struct {
	ui.Base
	label string
	get   func() (PluginControl, *PluginPage, bool) // the latest canvas and the description it is from

	seen   *PluginPage // the description last taken
	units  gfx.Pt      // the canvas's own size
	prev   []PluginShape
	cur    []PluginShape
	at     time.Time     // when cur arrived
	period time.Duration // how long the glide from prev to cur takes
	drawn  time.Time
	moving bool // some shape has an id: there is something to glide
}

// canvasFrame is the least time between two redraws of a gliding canvas.
const canvasFrame = 33 * time.Millisecond

func newCanvas(label string, get func() (PluginControl, *PluginPage, bool)) *canvasView {
	c := &canvasView{label: label, get: get, units: gfx.Pt{X: 100, Y: 100}}
	c.take(time.Now())
	return c
}

// take reads the latest description; a new one starts a glide from where
// the shapes are right now.
func (c *canvasView) take(now time.Time) bool {
	ctl, pg, ok := c.get()
	if !ok || pg == c.seen {
		return false
	}
	if c.seen != nil {
		c.prev = c.shapesAt(now)
		p := now.Sub(c.at)
		c.period = min(max(p, 50*time.Millisecond), 600*time.Millisecond)
	}
	c.seen = pg
	c.cur = ctl.Shapes
	c.at = now
	if ctl.W > 0 && ctl.H > 0 {
		c.units = gfx.Pt{X: ctl.W, Y: ctl.H}
	}
	c.moving = false
	for _, s := range c.cur {
		if s.ID != "" {
			c.moving = true
			break
		}
	}
	return true
}

// shapesAt is the picture at an instant: cur, with the shapes that have an
// id part way from prev.
func (c *canvasView) shapesAt(now time.Time) []PluginShape {
	if !c.moving || c.period <= 0 || len(c.prev) == 0 {
		return c.cur
	}
	t := float64(now.Sub(c.at)) / float64(c.period)
	if t >= 1 {
		return c.cur
	}
	byID := map[string]*PluginShape{}
	for i := range c.prev {
		if id := c.prev[i].ID; id != "" {
			byID[id] = &c.prev[i]
		}
	}
	out := make([]PluginShape, len(c.cur))
	for i, s := range c.cur {
		if p, ok := byID[s.ID]; ok && s.ID != "" && p.Shape == s.Shape {
			out[i] = lerpShape(*p, s, t)
		} else {
			out[i] = s
		}
	}
	return out
}

// lerpShape is the shape t of the way from a to b (its numbers; the words
// and colours are b's).
func lerpShape(a, b PluginShape, t float64) PluginShape {
	l := func(x, y float64) float64 { return x + (y-x)*t }
	o := b
	o.X, o.Y, o.R = l(a.X, b.X), l(a.Y, b.Y), l(a.R, b.R)
	o.W, o.H, o.Width = l(a.W, b.W), l(a.H, b.H), l(a.Width, b.Width)
	o.A0, o.A1 = l(a.A0, b.A0), l(a.A1, b.A1)
	if a.Alpha > 0 || b.Alpha > 0 {
		aa, ba := a.Alpha, b.Alpha
		if aa == 0 {
			aa = 1
		}
		if ba == 0 {
			ba = 1
		}
		o.Alpha = l(aa, ba)
	}
	if len(a.Points) == len(b.Points) && len(b.Points) > 0 {
		pts := make([][2]float64, len(b.Points))
		for i := range b.Points {
			pts[i] = [2]float64{l(a.Points[i][0], b.Points[i][0]), l(a.Points[i][1], b.Points[i][1])}
		}
		o.Points = pts
	}
	return o
}

func (c *canvasView) Dirty() []image.Rectangle {
	now := time.Now()
	fresh := c.take(now)
	gliding := c.moving && now.Sub(c.at) < c.period+canvasFrame
	if (fresh || gliding) && now.Sub(c.drawn) >= canvasFrame {
		c.MarkDirty()
	}
	return c.Base.Dirty()
}

// canvasColor is a shape's colour: a name from the theme or #rrggbb.
func canvasColor(th *ui.Theme, name string, fallback gfx.Color) gfx.Color {
	switch strings.ToLower(name) {
	case "":
		return fallback
	case "accent":
		return th.Accent
	case "text", "white":
		return th.Text
	case "dim":
		return th.TextDim
	case "muted":
		return th.TextMuted
	case "line":
		return th.Line
	case "panel":
		return th.Panel
	case "black":
		return th.Black
	case "record", "red":
		return th.Record
	case "warn", "amber":
		return th.Warn
	case "blue":
		return th.Blue
	}
	if len(name) == 7 && name[0] == '#' {
		if v, err := strconv.ParseUint(name[1:], 16, 32); err == nil {
			return gfx.Hex(uint32(v))
		}
	}
	return fallback
}

// canvasFont is a text shape's font by its size name.
func canvasFont(th *ui.Theme, size string) *ui.Font {
	switch size {
	case "tiny":
		return th.Fonts.Tiny
	case "body":
		return th.Fonts.Body
	case "label":
		return th.Fonts.Label
	case "title":
		return th.Fonts.Title
	case "big":
		return th.Fonts.Big
	case "huge":
		return th.Fonts.Huge
	}
	return th.Fonts.Small
}

func (c *canvasView) Draw(dst *gfx.Image, ctx *ui.Context, clip image.Rectangle) {
	th := ctx.Theme
	r := c.Bounds()
	c.drawn = time.Now()
	ui.Panel(dst, r, th.Radius, th.Black, clip)
	inner := ui.Inset(r, 12)
	clip = clip.Intersect(inner)
	if clip.Empty() {
		return
	}
	// The plugin's units onto the inset, kept in proportion and centred.
	s := math.Min(float64(inner.Dx())/c.units.X, float64(inner.Dy())/c.units.Y)
	ox := float64(inner.Min.X) + (float64(inner.Dx())-c.units.X*s)/2
	oy := float64(inner.Min.Y) + (float64(inner.Dy())-c.units.Y*s)/2
	m := gfx.Scale(s, s).Then(gfx.Translate(ox, oy))
	px := func(x, y float64) (float64, float64) { return ox + x*s, oy + y*s }
	// Widths are in the plugin's units (the mapping scales them); a
	// hairline is 1.5 px whatever the scale.
	stroke := func(w float64) gfx.Stroke {
		if w*s < 1.5 {
			w = 1.5 / s
		}
		return gfx.Stroke{Width: w, Cap: gfx.RoundCap, Join: gfx.RoundJoin}
	}
	for _, sh := range c.shapesAt(c.drawn) {
		isText := sh.Shape == "text"
		fallback := th.Accent
		if isText {
			fallback = th.Text
		}
		col := canvasColor(th, sh.Color, fallback)
		if sh.Alpha > 0 && sh.Alpha < 1 {
			col = col.WithAlpha(sh.Alpha)
		}
		p := &gfx.Path{}
		switch sh.Shape {
		case "circle":
			p.Circle(sh.X, sh.Y, sh.R)
			dst.Fill(p, m, gfx.NonZero, gfx.Solid(col), clip)
		case "glow":
			if sh.R <= 0 {
				continue
			}
			p.Circle(sh.X, sh.Y, sh.R)
			g := gfx.NewRadial(gfx.Pt{X: sh.X, Y: sh.Y}, gfx.Pt{X: sh.X, Y: sh.Y}, sh.R, m,
				[]gfx.Stop{{Offset: 0, Color: col}, {Offset: 0.45, Color: col.WithAlpha(0.45)}, {Offset: 1, Color: col.WithAlpha(0)}})
			dst.Fill(p, m, gfx.NonZero, g, clip)
		case "ring":
			p.Circle(sh.X, sh.Y, sh.R)
			dst.StrokePath(p, m, stroke(sh.Width), gfx.Solid(col), clip)
		case "arc":
			p.Arc(sh.X, sh.Y, sh.R, sh.A0*math.Pi/180, sh.A1*math.Pi/180)
			dst.StrokePath(p, m, stroke(sh.Width), gfx.Solid(col), clip)
		case "line":
			if len(sh.Points) < 2 {
				continue
			}
			p.MoveTo(sh.Points[0][0], sh.Points[0][1])
			for _, q := range sh.Points[1:] {
				p.LineTo(q[0], q[1])
			}
			dst.StrokePath(p, m, stroke(sh.Width), gfx.Solid(col), clip)
		case "polygon":
			if len(sh.Points) < 3 {
				continue
			}
			p.MoveTo(sh.Points[0][0], sh.Points[0][1])
			for _, q := range sh.Points[1:] {
				p.LineTo(q[0], q[1])
			}
			p.Close()
			dst.Fill(p, m, gfx.NonZero, gfx.Solid(col), clip)
		case "rect":
			if sh.Radius > 0 {
				p.RoundRect(sh.X, sh.Y, sh.W, sh.H, sh.Radius, sh.Radius)
			} else {
				p.Rect(sh.X, sh.Y, sh.W, sh.H)
			}
			dst.Fill(p, m, gfx.NonZero, gfx.Solid(col), clip)
		case "text":
			if sh.Text == "" {
				continue
			}
			f := canvasFont(th, sh.Size)
			x, y := px(sh.X, sh.Y)
			lh := ui.LineHeight(f)
			w := ui.TextWidth(f, sh.Text) + 4
			var box image.Rectangle
			a := font.Center
			switch strings.ToLower(sh.Align) {
			case "left":
				a = font.Left
				box = image.Rect(int(x), int(y)-lh/2, int(x)+w, int(y)+lh/2)
			case "right":
				a = font.Right
				box = image.Rect(int(x)-w, int(y)-lh/2, int(x), int(y)+lh/2)
			default:
				box = image.Rect(int(x)-w/2, int(y)-lh/2, int(x)+w/2, int(y)+lh/2)
			}
			ui.Text(dst, f, sh.Text, box, a, col, clip)
		}
	}
	if c.label != "" {
		ui.Text(dst, th.Fonts.Small, c.label, image.Rect(inner.Min.X+8, inner.Min.Y, inner.Max.X-8, inner.Min.Y+32), font.Left, th.TextDim, clip)
	}
}

func (c *canvasView) Touch(ui.TouchPhase, image.Point) bool { return false }

// canvasOf finds a page's first canvas control.
func canvasOf(p *PluginPage) (PluginControl, bool) {
	if p == nil {
		return PluginControl{}, false
	}
	for _, ctl := range p.Controls {
		if ctl.Type == "canvas" {
			return ctl, true
		}
	}
	return PluginControl{}, false
}

// canvasPlace splits a page between its canvas and the controls: a canvas
// that is about square or taller stands on the left, a wide one lies on
// top, each as large as its proportion allows.
func canvasPlace(r image.Rectangle, ctl PluginControl, pad int) (canvas, rest image.Rectangle) {
	w, h := ctl.W, ctl.H
	if w <= 0 || h <= 0 {
		w, h = 100, 100
	}
	aspect := w / h
	if aspect <= 1.35 {
		cw := math.Min(float64(r.Dy())*aspect, float64(r.Dx())*0.56)
		ch := cw / aspect
		canvas = image.Rect(r.Min.X, r.Min.Y, r.Min.X+int(cw), r.Min.Y+int(ch))
		rest = image.Rect(canvas.Max.X+pad, r.Min.Y, r.Max.X, r.Max.Y)
		return
	}
	ch := math.Min(float64(r.Dy())*0.6, float64(r.Dx())/aspect)
	cw := ch * aspect
	x0 := r.Min.X + (r.Dx()-int(cw))/2
	canvas = image.Rect(x0, r.Min.Y, x0+int(cw), r.Min.Y+int(ch))
	rest = image.Rect(r.Min.X, canvas.Max.Y+pad, r.Max.X, r.Max.Y)
	return
}
