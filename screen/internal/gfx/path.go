package gfx

import "math"

// Pt is a point in pixels (or in a shape's own units before a transform).
type Pt struct{ X, Y float64 }

// Affine maps points: x' = A*x + C*y + E, y' = B*x + D*y + F.
type Affine struct{ A, B, C, D, E, F float64 }

// Identity leaves points where they are.
var Identity = Affine{A: 1, D: 1}

func Translate(x, y float64) Affine { return Affine{A: 1, D: 1, E: x, F: y} }
func Scale(sx, sy float64) Affine   { return Affine{A: sx, D: sy} }
func Rotate(rad float64) Affine {
	s, c := math.Sincos(rad)
	return Affine{A: c, B: s, C: -s, D: c}
}

// Then is m followed by n.
func (m Affine) Then(n Affine) Affine {
	return Affine{
		A: n.A*m.A + n.C*m.B, B: n.B*m.A + n.D*m.B,
		C: n.A*m.C + n.C*m.D, D: n.B*m.C + n.D*m.D,
		E: n.A*m.E + n.C*m.F + n.E, F: n.B*m.E + n.D*m.F + n.F,
	}
}

// Apply maps p.
func (m Affine) Apply(p Pt) Pt { return Pt{m.A*p.X + m.C*p.Y + m.E, m.B*p.X + m.D*p.Y + m.F} }

// Invert is the inverse map (Identity if m cannot be inverted).
func (m Affine) Invert() Affine {
	det := m.A*m.D - m.B*m.C
	if det == 0 {
		return Identity
	}
	return Affine{
		A: m.D / det, B: -m.B / det, C: -m.C / det, D: m.A / det,
		E: (m.C*m.F - m.D*m.E) / det, F: (m.B*m.E - m.A*m.F) / det,
	}
}

// ScaleFactor is how much m enlarges lengths (on average), for the
// flattening tolerance and stroke widths.
func (m Affine) ScaleFactor() float64 { return math.Sqrt(math.Abs(m.A*m.D - m.B*m.C)) }

type verb uint8

const (
	moveTo verb = iota
	lineTo
	quadTo
	cubicTo
	closePath
)

// Path is an outline of straight and curved pieces, in one or more closed
// or open sub-paths.
type Path struct {
	verbs []verb
	pts   []Pt
	start Pt // of the current sub-path
	cur   Pt
	open  bool // a sub-path has been started
}

func (p *Path) MoveTo(x, y float64) {
	p.verbs, p.pts = append(p.verbs, moveTo), append(p.pts, Pt{x, y})
	p.start, p.cur, p.open = Pt{x, y}, Pt{x, y}, true
}

func (p *Path) ensure() {
	if !p.open {
		p.MoveTo(p.cur.X, p.cur.Y)
	}
}

func (p *Path) LineTo(x, y float64) {
	p.ensure()
	p.verbs, p.pts = append(p.verbs, lineTo), append(p.pts, Pt{x, y})
	p.cur = Pt{x, y}
}

func (p *Path) QuadTo(cx, cy, x, y float64) {
	p.ensure()
	p.verbs, p.pts = append(p.verbs, quadTo), append(p.pts, Pt{cx, cy}, Pt{x, y})
	p.cur = Pt{x, y}
}

func (p *Path) CubicTo(c1x, c1y, c2x, c2y, x, y float64) {
	p.ensure()
	p.verbs, p.pts = append(p.verbs, cubicTo), append(p.pts, Pt{c1x, c1y}, Pt{c2x, c2y}, Pt{x, y})
	p.cur = Pt{x, y}
}

// Close ends the sub-path with a line back to its start.
func (p *Path) Close() {
	if !p.open {
		return
	}
	p.verbs = append(p.verbs, closePath)
	p.cur, p.open = p.start, false
}

// Current is where the path ends now.
func (p *Path) Current() Pt { return p.cur }

// Empty reports whether the path has nothing to draw.
func (p *Path) Empty() bool { return len(p.verbs) == 0 }

// Append adds q's sub-paths.
func (p *Path) Append(q *Path) {
	p.verbs, p.pts = append(p.verbs, q.verbs...), append(p.pts, q.pts...)
	p.start, p.cur, p.open = q.start, q.cur, q.open
}

// Rect adds a closed rectangle.
func (p *Path) Rect(x, y, w, h float64) {
	p.MoveTo(x, y)
	p.LineTo(x+w, y)
	p.LineTo(x+w, y+h)
	p.LineTo(x, y+h)
	p.Close()
}

// kappa places cubic control points for a quarter circle.
const kappa = 0.5522847498307936

// RoundRect adds a rectangle with corners of radius rx, ry.
func (p *Path) RoundRect(x, y, w, h, rx, ry float64) {
	rx, ry = math.Min(math.Max(rx, 0), w/2), math.Min(math.Max(ry, 0), h/2)
	if rx == 0 || ry == 0 {
		p.Rect(x, y, w, h)
		return
	}
	kx, ky := rx*kappa, ry*kappa
	p.MoveTo(x+rx, y)
	p.LineTo(x+w-rx, y)
	p.CubicTo(x+w-rx+kx, y, x+w, y+ry-ky, x+w, y+ry)
	p.LineTo(x+w, y+h-ry)
	p.CubicTo(x+w, y+h-ry+ky, x+w-rx+kx, y+h, x+w-rx, y+h)
	p.LineTo(x+rx, y+h)
	p.CubicTo(x+rx-kx, y+h, x, y+h-ry+ky, x, y+h-ry)
	p.LineTo(x, y+ry)
	p.CubicTo(x, y+ry-ky, x+rx-kx, y, x+rx, y)
	p.Close()
}

// Ellipse adds a closed ellipse.
func (p *Path) Ellipse(cx, cy, rx, ry float64) {
	kx, ky := rx*kappa, ry*kappa
	p.MoveTo(cx+rx, cy)
	p.CubicTo(cx+rx, cy+ky, cx+kx, cy+ry, cx, cy+ry)
	p.CubicTo(cx-kx, cy+ry, cx-rx, cy+ky, cx-rx, cy)
	p.CubicTo(cx-rx, cy-ky, cx-kx, cy-ry, cx, cy-ry)
	p.CubicTo(cx+kx, cy-ry, cx+rx, cy-ky, cx+rx, cy)
	p.Close()
}

// Circle adds a closed circle.
func (p *Path) Circle(cx, cy, r float64) { p.Ellipse(cx, cy, r, r) }

// Arc adds a circular arc around (cx, cy) from angle a0 to a1 (radians,
// clockwise on the screen as angles grow), joined to the current point by
// a line if the path is open.
func (p *Path) Arc(cx, cy, r, a0, a1 float64) {
	n := int(math.Ceil(math.Abs(a1-a0) / (math.Pi / 2)))
	if n == 0 {
		return
	}
	step := (a1 - a0) / float64(n)
	k := 4.0 / 3 * math.Tan(step/4)
	s0, c0 := math.Sincos(a0)
	if p.open {
		p.LineTo(cx+r*c0, cy+r*s0)
	} else {
		p.MoveTo(cx+r*c0, cy+r*s0)
	}
	for i := 0; i < n; i++ {
		b := a0 + float64(i+1)*step
		s1, c1 := math.Sincos(b)
		p.CubicTo(cx+r*(c0-k*s0), cy+r*(s0+k*c0), cx+r*(c1+k*s1), cy+r*(s1-k*c1), cx+r*c1, cy+r*s1)
		s0, c0 = s1, c1
	}
}

// Transform is the path with every point mapped by m.
func (p *Path) Transform(m Affine) *Path {
	q := &Path{verbs: append([]verb(nil), p.verbs...), pts: make([]Pt, len(p.pts)), open: p.open}
	for i, pt := range p.pts {
		q.pts[i] = m.Apply(pt)
	}
	q.start, q.cur = m.Apply(p.start), m.Apply(p.cur)
	return q
}

// flatten turns the path into polylines (one per sub-path) within tol
// pixels of the curves; closed tells which sub-paths were closed.
func (p *Path) flatten(tol float64) (polys [][]Pt, closed []bool) {
	var cur []Pt
	var last Pt
	flush := func(c bool) {
		if len(cur) > 1 {
			polys, closed = append(polys, cur), append(closed, c)
		}
		cur = nil
	}
	i := 0
	for _, v := range p.verbs {
		switch v {
		case moveTo:
			flush(false)
			last = p.pts[i]
			cur = append(cur, last)
			i++
		case lineTo:
			last = p.pts[i]
			cur = append(cur, last)
			i++
		case quadTo:
			c, e := p.pts[i], p.pts[i+1]
			cur = flattenQuad(cur, last, c, e, tol)
			last = e
			i += 2
		case cubicTo:
			c1, c2, e := p.pts[i], p.pts[i+1], p.pts[i+2]
			cur = flattenCubic(cur, last, c1, c2, e, tol)
			last = e
			i += 3
		case closePath:
			if len(cur) > 0 {
				first := cur[0]
				flush(true)
				last = first
				cur = append(cur, first)
			}
		}
	}
	flush(false)
	return polys, closed
}

func flattenQuad(out []Pt, p0, p1, p2 Pt, tol float64) []Pt {
	// Number of pieces from the second difference (as in many renderers).
	dx, dy := p0.X-2*p1.X+p2.X, p0.Y-2*p1.Y+p2.Y
	dd := math.Hypot(dx, dy)
	n := int(math.Ceil(math.Sqrt(dd / (4 * tol))))
	n = max(1, min(n, 100))
	for i := 1; i <= n; i++ {
		t := float64(i) / float64(n)
		u := 1 - t
		out = append(out, Pt{u*u*p0.X + 2*u*t*p1.X + t*t*p2.X, u*u*p0.Y + 2*u*t*p1.Y + t*t*p2.Y})
	}
	return out
}

func flattenCubic(out []Pt, p0, p1, p2, p3 Pt, tol float64) []Pt {
	d1 := math.Hypot(p0.X-2*p1.X+p2.X, p0.Y-2*p1.Y+p2.Y)
	d2 := math.Hypot(p1.X-2*p2.X+p3.X, p1.Y-2*p2.Y+p3.Y)
	dd := math.Max(d1, d2)
	n := int(math.Ceil(math.Sqrt(dd * 3 / (4 * tol))))
	n = max(1, min(n, 200))
	for i := 1; i <= n; i++ {
		t := float64(i) / float64(n)
		u := 1 - t
		a, b, c, d := u*u*u, 3*u*u*t, 3*u*t*t, t*t*t
		out = append(out, Pt{a*p0.X + b*p1.X + c*p2.X + d*p3.X, a*p0.Y + b*p1.Y + c*p2.Y + d*p3.Y})
	}
	return out
}

// Bounds is the box around the path's points (control points included).
func (p *Path) Bounds() (min, max Pt) {
	if len(p.pts) == 0 {
		return
	}
	min, max = p.pts[0], p.pts[0]
	for _, q := range p.pts[1:] {
		min.X, min.Y = math.Min(min.X, q.X), math.Min(min.Y, q.Y)
		max.X, max.Y = math.Max(max.X, q.X), math.Max(max.Y, q.Y)
	}
	return
}
