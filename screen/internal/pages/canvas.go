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
// picture a few times a second (or streams it); shapes that carry an id
// glide from where the previous description had them to where the new one
// has them over the time between the two, so a radar's blip or a meter's
// bar moves as smoothly as the screen draws, whatever the plugin's rate.
//
// Drawing is kept cheap, as the device's processor is small: the still
// part of the picture (the panel, the label and the shapes at the start of
// the list that the last description left as they were) is drawn once into
// bg; a frame copies bg and draws the rest over it, and only where a shape
// moved or changed, so a gliding blip costs its own few pixels, not the
// whole picture.
type canvasView struct {
	ui.Base
	label string
	get   func() (PluginControl, *PluginPage, bool) // the latest canvas and the description it is from
	now   func() time.Time

	seen   *PluginPage // the description last taken
	units  gfx.Pt      // the canvas's own size
	prev   []PluginShape
	cur    []PluginShape
	at     time.Time     // when cur arrived
	period time.Duration // how long the glide from prev to cur takes
	drawn  time.Time
	moving bool // some shape has an id: there is something to glide
	static int  // how many shapes at the start of cur the last description did not change

	th     *ui.Theme     // the theme last drawn with (text sizes for the dirty rectangles)
	bg     *gfx.Image    // the still part, for bgR
	bgR    image.Rectangle
	bgFrom []PluginShape // the still shapes bg holds
	next   []PluginShape // the picture the coming frame draws
	last   []PluginShape // the moving part of the picture on the screen
	full   bool          // the coming frame draws everything
}

// canvasFrame is the least time between two redraws of a gliding canvas.
const canvasFrame = 33 * time.Millisecond

func newCanvas(label string, get func() (PluginControl, *PluginPage, bool)) *canvasView {
	c := &canvasView{label: label, get: get, now: time.Now, units: gfx.Pt{X: 100, Y: 100}, full: true}
	c.take(c.now())
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
		c.static = stillPrefix(c.cur, ctl.Shapes)
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

// stillPrefix counts the shapes at the start of b that a has the same, and
// that carry no id (those move): the part of the picture that stays.
func stillPrefix(a, b []PluginShape) int {
	n := 0
	for n < len(a) && n < len(b) && b[n].ID == "" && shapeEqual(a[n], b[n]) {
		n++
	}
	return n
}

// shapeEqual reports whether two shapes are the same in every respect.
func shapeEqual(a, b PluginShape) bool {
	if a.ID != b.ID || a.Shape != b.Shape || a.X != b.X || a.Y != b.Y || a.R != b.R ||
		a.W != b.W || a.H != b.H || a.Radius != b.Radius || a.A0 != b.A0 || a.A1 != b.A1 ||
		a.Width != b.Width || a.Text != b.Text || a.Size != b.Size || a.Align != b.Align ||
		a.Color != b.Color || a.Alpha != b.Alpha || len(a.Points) != len(b.Points) {
		return false
	}
	for i := range a.Points {
		if a.Points[i] != b.Points[i] {
			return false
		}
	}
	return true
}

func sameShapes(a, b []PluginShape) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !shapeEqual(a[i], b[i]) {
			return false
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

// Dirty settles the coming frame's picture and says where it differs from
// the one on the screen: everything after a relayout or when the still
// part changed, else the places the moving shapes left and the places they
// are now.
func (c *canvasView) Dirty() []image.Rectangle {
	now := c.now()
	fresh := c.take(now)
	if c.Base.Dirty() != nil {
		c.full = true
	}
	gliding := c.moving && now.Sub(c.at) < c.period+canvasFrame
	if !c.full && !fresh && !gliding {
		return nil
	}
	if !c.full && now.Sub(c.drawn) < canvasFrame {
		return nil
	}
	c.next = c.shapesAt(now)
	r := c.Bounds()
	if c.full || c.th == nil || c.bg == nil || c.bgR != r {
		c.full = true
		return []image.Rectangle{r}
	}
	still, moving := c.next[:c.static], c.next[c.static:]
	if !sameShapes(still, c.bgFrom) {
		c.full = true
		return []image.Rectangle{r}
	}
	g := c.geom(r, image.Point{})
	if len(moving) != len(c.last) {
		return []image.Rectangle{g.inner}
	}
	var out []image.Rectangle
	for i := range moving {
		if shapeEqual(moving[i], c.last[i]) {
			continue
		}
		for _, b := range []image.Rectangle{c.bbox(g, c.last[i]), c.bbox(g, moving[i])} {
			if !b.Empty() {
				out = append(out, b)
			}
		}
	}
	return coalesce(out)
}

// coalesce joins rectangles that overlap or touch, so a shape's old and
// new place are drawn once; the lists are short.
func coalesce(rs []image.Rectangle) []image.Rectangle {
	for {
		joined := false
		for i := 0; i < len(rs) && !joined; i++ {
			for j := i + 1; j < len(rs); j++ {
				if rs[i].Inset(-1).Overlaps(rs[j]) {
					rs[i] = rs[i].Union(rs[j])
					rs = append(rs[:j], rs[j+1:]...)
					joined = true
					break
				}
			}
		}
		if !joined {
			return rs
		}
	}
}

// canvasGeom maps the plugin's units onto the pixels: a uniform scale,
// centred in the inset (shifted by off when drawing into a buffer of the
// canvas's own).
type canvasGeom struct {
	s, ox, oy float64
	inner     image.Rectangle
}

func (c *canvasView) geom(r image.Rectangle, off image.Point) canvasGeom {
	inner := ui.Inset(r, 12).Sub(off)
	s := math.Min(float64(inner.Dx())/c.units.X, float64(inner.Dy())/c.units.Y)
	return canvasGeom{
		s:     s,
		ox:    float64(inner.Min.X) + (float64(inner.Dx())-c.units.X*s)/2,
		oy:    float64(inner.Min.Y) + (float64(inner.Dy())-c.units.Y*s)/2,
		inner: inner,
	}
}

func (g canvasGeom) px(x, y float64) (float64, float64) { return g.ox + x*g.s, g.oy + y*g.s }

// affine is the mapping for the rasterizer.
func (g canvasGeom) affine() gfx.Affine { return gfx.Scale(g.s, g.s).Then(gfx.Translate(g.ox, g.oy)) }

// stroke is a stroke of w units; a hairline is 1.5 px whatever the scale
// (the mapping scales the width).
func (g canvasGeom) stroke(w float64) gfx.Stroke {
	if w*g.s < 1.5 {
		w = 1.5 / g.s
	}
	return gfx.Stroke{Width: w, Cap: gfx.RoundCap, Join: gfx.RoundJoin}
}

// textBox is where a text shape goes and how it is aligned.
func (c *canvasView) textBox(g canvasGeom, sh PluginShape) (image.Rectangle, font.Align, *ui.Font) {
	f := canvasFont(c.th, sh.Size)
	x, y := g.px(sh.X, sh.Y)
	lh := ui.LineHeight(f)
	w := ui.TextWidth(f, sh.Text) + 4
	switch strings.ToLower(sh.Align) {
	case "left":
		return image.Rect(int(x), int(y)-lh/2, int(x)+w, int(y)+lh/2), font.Left, f
	case "right":
		return image.Rect(int(x)-w, int(y)-lh/2, int(x), int(y)+lh/2), font.Right, f
	}
	return image.Rect(int(x)-w/2, int(y)-lh/2, int(x)+w/2, int(y)+lh/2), font.Center, f
}

// bbox is the pixels a shape can touch (within the inset, with room for
// the anti-aliasing).
func (c *canvasView) bbox(g canvasGeom, sh PluginShape) image.Rectangle {
	var x0, y0, x1, y1 float64 // in units
	switch sh.Shape {
	case "circle", "glow":
		x0, y0, x1, y1 = sh.X-sh.R, sh.Y-sh.R, sh.X+sh.R, sh.Y+sh.R
	case "ring", "arc":
		w := g.stroke(sh.Width).Width / 2
		x0, y0, x1, y1 = sh.X-sh.R-w, sh.Y-sh.R-w, sh.X+sh.R+w, sh.Y+sh.R+w
	case "line", "polygon":
		if len(sh.Points) == 0 {
			return image.Rectangle{}
		}
		x0, y0, x1, y1 = math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
		for _, p := range sh.Points {
			x0, y0 = math.Min(x0, p[0]), math.Min(y0, p[1])
			x1, y1 = math.Max(x1, p[0]), math.Max(y1, p[1])
		}
		if sh.Shape == "line" {
			w := g.stroke(sh.Width).Width / 2
			x0, y0, x1, y1 = x0-w, y0-w, x1+w, y1+w
		}
	case "rect":
		x0, y0, x1, y1 = math.Min(sh.X, sh.X+sh.W), math.Min(sh.Y, sh.Y+sh.H), math.Max(sh.X, sh.X+sh.W), math.Max(sh.Y, sh.Y+sh.H)
	case "text":
		if sh.Text == "" {
			return image.Rectangle{}
		}
		box, _, _ := c.textBox(g, sh)
		return box.Inset(-2).Intersect(g.inner)
	default:
		return image.Rectangle{}
	}
	px0, py0 := g.px(x0, y0)
	px1, py1 := g.px(x1, y1)
	return image.Rect(int(math.Floor(px0))-2, int(math.Floor(py0))-2, int(math.Ceil(px1))+3, int(math.Ceil(py1))+3).Intersect(g.inner)
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
	c.drawn = c.now()
	c.th = th
	if c.next == nil { // drawn without Dirty (previews, tests)
		c.next = c.shapesAt(c.drawn)
	}
	if c.static > len(c.next) {
		c.static = len(c.next)
	}
	still, moving := c.next[:c.static], c.next[c.static:]
	if c.bg == nil || c.bgR != r || c.full || !sameShapes(still, c.bgFrom) {
		c.renderBg(th, r, still)
	}
	c.full = false
	clip = clip.Intersect(r)
	if clip.Empty() {
		return
	}
	dst.CopyIn(c.bg, r.Min.X, r.Min.Y, clip)
	inner := ui.Inset(r, 12)
	c.paint(dst, th, clip.Intersect(inner), c.geom(r, image.Point{}), moving)
	c.last = append(c.last[:0], moving...)
}

// renderBg draws the still part of the picture into a buffer of the
// canvas's size: the page's background (under the panel's corners), the
// panel, the still shapes and the label.
func (c *canvasView) renderBg(th *ui.Theme, r image.Rectangle, still []PluginShape) {
	if c.bg == nil || c.bgR.Dx() != r.Dx() || c.bgR.Dy() != r.Dy() {
		c.bg = gfx.NewImage(r.Dx(), r.Dy())
	}
	c.bgR = r
	c.bgFrom = append(c.bgFrom[:0], still...)
	off := r.Min
	local := r.Sub(off)
	c.bg.Set(local, th.Bg)
	ui.Panel(c.bg, local, th.Radius, th.Black, local)
	inner := ui.Inset(local, 12)
	c.paint(c.bg, th, inner, c.geom(r, off), still)
	if c.label != "" {
		ui.Text(c.bg, th.Fonts.Small, c.label, image.Rect(inner.Min.X+8, inner.Min.Y, inner.Max.X-8, inner.Min.Y+32), font.Left, th.TextDim, inner)
	}
}

// paint draws shapes within clip (shapes that cannot touch it are skipped).
func (c *canvasView) paint(dst *gfx.Image, th *ui.Theme, clip image.Rectangle, g canvasGeom, shapes []PluginShape) {
	if clip.Empty() {
		return
	}
	m := g.affine()
	for _, sh := range shapes {
		if !c.bbox(g, sh).Overlaps(clip) {
			continue
		}
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
			gr := gfx.NewRadial(gfx.Pt{X: sh.X, Y: sh.Y}, gfx.Pt{X: sh.X, Y: sh.Y}, sh.R, m,
				[]gfx.Stop{{Offset: 0, Color: col}, {Offset: 0.45, Color: col.WithAlpha(0.45)}, {Offset: 1, Color: col.WithAlpha(0)}})
			dst.Fill(p, m, gfx.NonZero, gr, clip)
		case "ring":
			p.Circle(sh.X, sh.Y, sh.R)
			dst.StrokePath(p, m, g.stroke(sh.Width), gfx.Solid(col), clip)
		case "arc":
			p.Arc(sh.X, sh.Y, sh.R, sh.A0*math.Pi/180, sh.A1*math.Pi/180)
			dst.StrokePath(p, m, g.stroke(sh.Width), gfx.Solid(col), clip)
		case "line":
			if len(sh.Points) < 2 {
				continue
			}
			p.MoveTo(sh.Points[0][0], sh.Points[0][1])
			for _, q := range sh.Points[1:] {
				p.LineTo(q[0], q[1])
			}
			dst.StrokePath(p, m, g.stroke(sh.Width), gfx.Solid(col), clip)
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
			box, a, f := c.textBox(g, sh)
			ui.Text(dst, f, sh.Text, box, a, col, clip)
		}
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
