package gfx

import (
	"image"
	"math"
)

// FillRule decides what is inside where outlines overlap.
type FillRule uint8

const (
	NonZero FillRule = iota
	EvenOdd
)

// tolerance is how far (pixels) flattened curves may stray.
const tolerance = 0.05

// coverage is the anti-aliased area a path covers, as signed areas
// accumulated per pixel: running sums along each row give the winding
// number (fractional at the edges), which the fill rule folds into 0…1.
// Interiors are exact for both rules; only edges that overlap within one
// pixel are approximated (as most renderers do).
type coverage struct {
	r   image.Rectangle // pixels covered (device space)
	w   int             // r.Dx() + 1: one more cell per row for x = right edge
	acc []float32
}

func newCoverage(r image.Rectangle) *coverage {
	c := &coverage{r: r, w: r.Dx() + 1}
	c.acc = make([]float32, c.w*r.Dy())
	return c
}

// line adds the edge p→q (device space).
func (c *coverage) line(p, q Pt) {
	ox, oy := float64(c.r.Min.X), float64(c.r.Min.Y)
	p.X, p.Y, q.X, q.Y = p.X-ox, p.Y-oy, q.X-ox, q.Y-oy
	right := float64(c.r.Dx())
	// Parts left or right of the area become vertical edges at its border:
	// what matters there is only how much winding they bring.
	ts := []float64{0, 1}
	if p.X != q.X {
		for _, b := range []float64{0, right} {
			if t := (b - p.X) / (q.X - p.X); t > 0 && t < 1 {
				ts = append(ts, t)
			}
		}
		if len(ts) == 4 && ts[2] > ts[3] {
			ts[2], ts[3] = ts[3], ts[2]
		}
		if len(ts) > 2 {
			ts = append(ts[:1], append(ts[2:], 1)...)
		}
	}
	at := func(t float64) Pt { return Pt{p.X + t*(q.X-p.X), p.Y + t*(q.Y-p.Y)} }
	for i := 0; i+1 < len(ts); i++ {
		a, b := at(ts[i]), at(ts[i+1])
		switch mid := (a.X + b.X) / 2; {
		case mid < 0:
			a.X, b.X = 0, 0
		case mid > right:
			a.X, b.X = right, right
		default:
			a.X, b.X = math.Max(0, math.Min(a.X, right)), math.Max(0, math.Min(b.X, right))
		}
		c.edge(a, b)
	}
}

// edge accumulates a line that lies within 0 ≤ x ≤ width (local space).
func (c *coverage) edge(p, q Pt) {
	if p.Y == q.Y {
		return
	}
	dir := float32(1)
	if p.Y > q.Y {
		p, q, dir = q, p, -1
	}
	h := float64(c.r.Dy())
	if q.Y <= 0 || p.Y >= h {
		return
	}
	dxdy := (q.X - p.X) / (q.Y - p.Y)
	x := p.X
	if p.Y < 0 {
		x -= p.Y * dxdy
	}
	y0 := int(math.Max(0, math.Floor(p.Y)))
	y1 := int(math.Min(h, math.Ceil(q.Y)))
	maxX := float64(c.w - 1)
	for y := y0; y < y1; y++ {
		row := c.acc[y*c.w : (y+1)*c.w]
		dy := math.Min(float64(y+1), q.Y) - math.Max(float64(y), p.Y)
		xnext := x + dxdy*dy
		d := float32(dy) * dir
		x0, x1 := x, xnext
		if x0 > x1 {
			x0, x1 = x1, x0
		}
		x0, x1 = math.Max(0, math.Min(x0, maxX)), math.Max(0, math.Min(x1, maxX))
		x0f := math.Floor(x0)
		x0i := int(x0f)
		x1c := math.Ceil(x1)
		x1i := int(x1c)
		if x1i <= x0i+1 {
			xm := float32(0.5*(x0+x1) - x0f)
			row[x0i] += d - d*xm
			if x0i+1 < len(row) {
				row[x0i+1] += d * xm
			}
		} else {
			s := 1 / (x1 - x0)
			fx0 := x0 - x0f
			a0 := 0.5 * s * (1 - fx0) * (1 - fx0)
			fx1 := x1 - x1c + 1
			am := 0.5 * s * fx1 * fx1
			row[x0i] += d * float32(a0)
			if x1i == x0i+2 {
				row[x0i+1] += d * float32(1-a0-am)
			} else {
				a1 := s * (1.5 - fx0)
				row[x0i+1] += d * float32(a1-a0)
				ds := d * float32(s)
				for xi := x0i + 2; xi < x1i-1; xi++ {
					row[xi] += ds
				}
				a2 := a1 + float64(x1i-x0i-3)*s
				row[x1i-1] += d * float32(1-a2-am)
			}
			if x1i < len(row) {
				row[x1i] += d * float32(am)
			}
		}
		x = xnext
	}
}

// poly adds a polygon (closed implicitly).
func (c *coverage) poly(pts []Pt) {
	for i := range pts {
		j := i + 1
		if j == len(pts) {
			j = 0
		}
		c.line(pts[i], pts[j])
	}
}

// spans calls fn for each row with the coverage (0…255) of its pixels.
func (c *coverage) spans(rule FillRule, fn func(y, x0 int, cov []uint8)) {
	buf := make([]uint8, c.r.Dx())
	for y := 0; y < c.r.Dy(); y++ {
		row := c.acc[y*c.w : y*c.w+c.r.Dx()]
		var s float32
		lo, hi := -1, -1
		for x, v := range row {
			s += v
			a := s
			if a < 0 {
				a = -a
			}
			if rule == EvenOdd {
				a = float32(math.Mod(float64(a), 2))
				if a > 1 {
					a = 2 - a
				}
			} else if a > 1 {
				a = 1
			}
			b := uint8(a*255 + 0.5)
			buf[x] = b
			if b != 0 {
				if lo < 0 {
					lo = x
				}
				hi = x
			}
		}
		if lo >= 0 {
			fn(c.r.Min.Y+y, c.r.Min.X+lo, buf[lo:hi+1])
		}
	}
}

// Paint gives the colour of pixels: a solid colour, a gradient or an image.
type Paint interface {
	// shade fills out with premultiplied colours for the pixels x…x+len(out)
	// of row y.
	shade(y, x int, out []uint32)
}

// Solid paints one colour.
type Solid Color

func (s Solid) shade(_, _ int, out []uint32) {
	p := Color(s).Premul()
	for i := range out {
		out[i] = p
	}
}

// Fill paints the inside of path (mapped by m) onto dst within clip.
func (dst *Image) Fill(path *Path, m Affine, rule FillRule, paint Paint, clip image.Rectangle) {
	polys, _ := path.Transform(m).flatten(tolerance)
	dst.fillPolys(polys, rule, paint, clip, 255)
}

// FillAlpha is Fill with an extra opacity (0…255).
func (dst *Image) FillAlpha(path *Path, m Affine, rule FillRule, paint Paint, clip image.Rectangle, alpha uint8) {
	polys, _ := path.Transform(m).flatten(tolerance)
	dst.fillPolys(polys, rule, paint, clip, alpha)
}

func (dst *Image) fillPolys(polys [][]Pt, rule FillRule, paint Paint, clip image.Rectangle, alpha uint8) {
	c := polysCoverage(polys, clip.Intersect(dst.Bounds()))
	if c == nil {
		return
	}
	dst.paintCoverage(c, rule, paint, alpha)
}

// polysCoverage accumulates polygons within clip (nil if nothing shows).
func polysCoverage(polys [][]Pt, clip image.Rectangle) *coverage {
	if len(polys) == 0 || clip.Empty() {
		return nil
	}
	minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, pl := range polys {
		for _, p := range pl {
			minX, minY = math.Min(minX, p.X), math.Min(minY, p.Y)
			maxX, maxY = math.Max(maxX, p.X), math.Max(maxY, p.Y)
		}
	}
	if math.IsInf(minX, 0) || math.IsNaN(minX+minY+maxX+maxY) {
		return nil
	}
	r := image.Rect(int(math.Floor(minX)), int(math.Floor(minY)), int(math.Ceil(maxX))+1, int(math.Ceil(maxY))+1).Intersect(clip)
	// Winding from the left still counts when the shape starts left of the
	// clip; the area keeps the clip's left edge and lets line() fold it in.
	if r.Empty() {
		return nil
	}
	c := newCoverage(r)
	for _, pl := range polys {
		c.poly(pl)
	}
	return c
}

func (dst *Image) paintCoverage(c *coverage, rule FillRule, paint Paint, alpha uint8) {
	var shades []uint32
	solid, isSolid := paint.(Solid)
	sp := Color(solid).Premul()
	c.spans(rule, func(y, x0 int, cov []uint8) {
		row := dst.Pix[y*dst.Stride+x0 : y*dst.Stride+x0+len(cov)]
		if !isSolid {
			if cap(shades) < len(cov) {
				shades = make([]uint32, len(cov))
			}
			shades = shades[:len(cov)]
			paint.shade(y, x0, shades)
		}
		for i, a := range cov {
			if a == 0 {
				continue
			}
			s := sp
			if !isSolid {
				s = shades[i]
			}
			k := uint32(a)
			if alpha != 255 {
				k = (k*uint32(alpha) + 127) / 255
			}
			row[i] = over(scale(s, k), row[i])
		}
	})
}
