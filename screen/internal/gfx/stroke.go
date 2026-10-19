package gfx

import (
	"image"
	"math"
)

// Cap is how open lines end.
type Cap uint8

const (
	ButtCap Cap = iota
	RoundCap
	SquareCap
)

// Join is how lines meet at corners.
type Join uint8

const (
	MiterJoin Join = iota
	RoundJoin
	BevelJoin
)

// Stroke describes a line drawn along a path.
type Stroke struct {
	Width float64
	Cap   Cap
	Join  Join
	Miter float64 // miter limit (ratio), 4 if 0
}

// StrokePath draws a line along path (mapped by m).
func (dst *Image) StrokePath(path *Path, m Affine, st Stroke, paint Paint, clip image.Rectangle) {
	dst.StrokePathAlpha(path, m, st, paint, clip, 255)
}

// StrokePathAlpha is StrokePath with an extra opacity (0…255).
func (dst *Image) StrokePathAlpha(path *Path, m Affine, st Stroke, paint Paint, clip image.Rectangle, alpha uint8) {
	hw := st.Width * m.ScaleFactor() / 2
	if hw <= 0 {
		return
	}
	lines, closed := path.Transform(m).flatten(tolerance)
	var polys [][]Pt
	for i, pl := range lines {
		polys = strokePolyline(polys, dedupe(pl), closed[i], hw, st)
	}
	dst.fillPolys(polys, NonZero, paint, clip, alpha)
}

func dedupe(pl []Pt) []Pt {
	out := pl[:0:0]
	for _, p := range pl {
		if len(out) == 0 || math.Hypot(p.X-out[len(out)-1].X, p.Y-out[len(out)-1].Y) > 1e-6 {
			out = append(out, p)
		}
	}
	return out
}

// strokePolyline adds the outline of one stroked polyline as whole
// polygons: an open line becomes one polygon (left side, end cap, right
// side back, start cap), a closed one two loops turning opposite ways.
// Inner corners pass through the corner point, so the non-zero rule keeps
// them filled; edges never overlap themselves, so anti-aliasing stays
// exact.
func strokePolyline(polys [][]Pt, pl []Pt, closed bool, hw float64, st Stroke) [][]Pt {
	if closed && len(pl) > 2 && pl[0] == pl[len(pl)-1] {
		pl = pl[:len(pl)-1]
	}
	if len(pl) < 2 {
		if len(pl) == 1 && st.Cap == RoundCap {
			polys = append(polys, circlePoly(pl[0], hw))
		} else if len(pl) == 1 && st.Cap == SquareCap {
			p := pl[0]
			polys = append(polys, []Pt{{p.X - hw, p.Y - hw}, {p.X + hw, p.Y - hw}, {p.X + hw, p.Y + hw}, {p.X - hw, p.Y + hw}})
		}
		return polys
	}
	if closed && len(pl) > 2 {
		return append(polys, side(pl, true, hw, st), side(reversed(pl), true, hw, st))
	}
	left := side(pl, false, hw, st)
	right := side(reversed(pl), false, hw, st)
	var out []Pt
	out = append(out, left...)
	out = appendCap(out, pl[len(pl)-2], pl[len(pl)-1], hw, st.Cap)
	out = append(out, right...)
	out = appendCap(out, pl[1], pl[0], hw, st.Cap)
	return append(polys, out)
}

func reversed(pl []Pt) []Pt {
	r := make([]Pt, len(pl))
	for i, p := range pl {
		r[len(pl)-1-i] = p
	}
	return r
}

// side is the offset line on the left of the direction of travel, with
// joins at outer corners.
func side(pl []Pt, closed bool, hw float64, st Stroke) []Pt {
	n := len(pl)
	segs := n - 1
	if closed {
		segs = n
	}
	var out []Pt
	for i := 0; i < segs; i++ {
		a, b := pl[i], pl[(i+1)%n]
		nx, ny := normal(a, b, hw)
		if i == 0 || closed && i == 0 {
			out = append(out, Pt{a.X + nx, a.Y + ny})
		}
		out = append(out, Pt{b.X + nx, b.Y + ny})
		if i+1 < segs || closed {
			c := pl[(i+2)%n]
			out = appendJoin(out, a, b, c, hw, st)
		}
	}
	if closed && len(out) > 0 {
		// The loop starts after the last join; the first point was the start
		// of segment 0, which the last join already reached.
		out = out[1:]
	}
	return out
}

// appendJoin adds the corner at v between the segments prev→v and v→next
// on the left side.
func appendJoin(out []Pt, prev, v, next Pt, hw float64, st Stroke) []Pt {
	n2x, n2y := normal(v, next, hw)
	cross := (v.X-prev.X)*(next.Y-v.Y) - (v.Y-prev.Y)*(next.X-v.X)
	o2 := Pt{v.X + n2x, v.Y + n2y}
	if math.Abs(cross) < 1e-9 {
		return append(out, o2)
	}
	// With y pointing down and the normal (-dy, dx), the left side is the
	// outer side of a corner that turns with cross < 0.
	if cross > 0 {
		return append(out, v, o2) // inner corner: through the corner point
	}
	n1x, n1y := normal(prev, v, hw)
	o1 := Pt{v.X + n1x, v.Y + n1y}
	switch st.Join {
	case RoundJoin:
		a0 := math.Atan2(n1y, n1x)
		a1 := math.Atan2(n2y, n2x)
		for a1 > a0 {
			a1 -= 2 * math.Pi
		}
		return append(appendArc(out, v, hw, a0, a1), o2)
	case MiterJoin:
		limit := st.Miter
		if limit == 0 {
			limit = 4
		}
		d1x, d1y := v.X-prev.X, v.Y-prev.Y
		d2x, d2y := next.X-v.X, next.Y-v.Y
		if den := d1x*d2y - d1y*d2x; den != 0 {
			t := ((o2.X-o1.X)*d2y - (o2.Y-o1.Y)*d2x) / den
			tip := Pt{o1.X + t*d1x, o1.Y + t*d1y}
			if math.Hypot(tip.X-v.X, tip.Y-v.Y) <= limit*hw {
				return append(out, tip, o2)
			}
		}
	}
	return append(out, o2)
}

// appendArc adds points on the circle (c, r) from angle a0 to a1 (ends
// excluded).
func appendArc(out []Pt, c Pt, r, a0, a1 float64) []Pt {
	step := math.Acos(math.Max(-1, 1-tolerance/math.Max(r, tolerance))) * 2
	n := int(math.Ceil(math.Abs(a1-a0) / step))
	for i := 1; i < n; i++ {
		s, co := math.Sincos(a0 + (a1-a0)*float64(i)/float64(n))
		out = append(out, Pt{c.X + r*co, c.Y + r*s})
	}
	return out
}

func normal(a, b Pt, hw float64) (float64, float64) {
	dx, dy := b.X-a.X, b.Y-a.Y
	l := math.Hypot(dx, dy)
	if l == 0 {
		return 0, 0
	}
	return -dy / l * hw, dx / l * hw
}

// appendCap goes round the end of the line at end (coming from from), from
// the left side to the right side.
func appendCap(out []Pt, from, end Pt, hw float64, c Cap) []Pt {
	nx, ny := normal(from, end, hw)
	dx, dy := end.X-from.X, end.Y-from.Y
	l := math.Hypot(dx, dy)
	if l == 0 {
		return out
	}
	ux, uy := dx/l*hw, dy/l*hw
	switch c {
	case RoundCap:
		a0 := math.Atan2(ny, nx)
		out = appendArc(out, end, hw, a0, a0-math.Pi)
	case SquareCap:
		out = append(out, Pt{end.X + nx + ux, end.Y + ny + uy}, Pt{end.X - nx + ux, end.Y - ny + uy})
	}
	return out
}

// circlePoly is a circle as a polygon fine enough for its size.
func circlePoly(c Pt, r float64) []Pt {
	n := int(math.Ceil(math.Pi / math.Acos(math.Max(-1, 1-tolerance/math.Max(r, tolerance)))))
	n = max(8, min(n, 256))
	pts := make([]Pt, n)
	for i := range pts {
		s, co := math.Sincos(2 * math.Pi * float64(i) / float64(n))
		pts[i] = Pt{c.X + r*co, c.Y + r*s}
	}
	return pts
}

// oriented makes the polygon turn the positive way.
func oriented(pts []Pt) []Pt {
	var a float64
	for i := range pts {
		j := (i + 1) % len(pts)
		a += pts[i].X*pts[j].Y - pts[j].X*pts[i].Y
	}
	if a < 0 {
		for i, j := 0, len(pts)-1; i < j; i, j = i+1, j-1 {
			pts[i], pts[j] = pts[j], pts[i]
		}
	}
	return pts
}
