package gfx

import (
	"math"
	"sort"
)

// Stop is a colour at a place (0…1) of a gradient.
type Stop struct {
	Offset float64
	Color  Color
}

// lut is a gradient as 256 premultiplied colours.
type lut [256]uint32

func makeLUT(stops []Stop) *lut {
	var l lut
	if len(stops) == 0 {
		return &l
	}
	s := append([]Stop(nil), stops...)
	sort.SliceStable(s, func(i, j int) bool { return s[i].Offset < s[j].Offset })
	for i := range l {
		t := float64(i) / 255
		c := s[0].Color
		switch {
		case t <= s[0].Offset:
		case t >= s[len(s)-1].Offset:
			c = s[len(s)-1].Color
		default:
			for k := 1; k < len(s); k++ {
				if t <= s[k].Offset {
					span := s[k].Offset - s[k-1].Offset
					f := 0.0
					if span > 0 {
						f = (t - s[k-1].Offset) / span
					}
					c = Mix(s[k-1].Color, s[k].Color, f)
					break
				}
			}
		}
		l[i] = c.Premul()
	}
	return &l
}

func (l *lut) at(t float64) uint32 {
	switch {
	case t <= 0 || math.IsNaN(t):
		return l[0]
	case t >= 1:
		return l[255]
	}
	return l[int(t*255+0.5)]
}

// Linear is a linear gradient from P0 to P1 (pad spread), in the space
// that M maps onto the device.
type Linear struct {
	P0, P1 Pt
	M      Affine
	lut    *lut
	inv    Affine
}

// NewLinear makes a linear gradient; m maps its points to device pixels.
func NewLinear(p0, p1 Pt, m Affine, stops []Stop) *Linear {
	return &Linear{P0: p0, P1: p1, M: m, lut: makeLUT(stops), inv: m.Invert()}
}

func (g *Linear) shade(y, x int, out []uint32) {
	dx, dy := g.P1.X-g.P0.X, g.P1.Y-g.P0.Y
	ll := dx*dx + dy*dy
	for i := range out {
		p := g.inv.Apply(Pt{float64(x+i) + 0.5, float64(y) + 0.5})
		t := 0.0
		if ll > 0 {
			t = ((p.X-g.P0.X)*dx + (p.Y-g.P0.Y)*dy) / ll
		}
		out[i] = g.lut.at(t)
	}
}

// Radial is a radial gradient around C with radius R and focal point F.
type Radial struct {
	C, F Pt
	R    float64
	M    Affine
	lut  *lut
	inv  Affine
}

// NewRadial makes a radial gradient; m maps its points to device pixels.
func NewRadial(c, f Pt, r float64, m Affine, stops []Stop) *Radial {
	return &Radial{C: c, F: f, R: r, M: m, lut: makeLUT(stops), inv: m.Invert()}
}

func (g *Radial) shade(y, x int, out []uint32) {
	for i := range out {
		p := g.inv.Apply(Pt{float64(x+i) + 0.5, float64(y) + 0.5})
		out[i] = g.lut.at(g.t(p))
	}
}

// t solves for the circle through p (focal gradients as in SVG).
func (g *Radial) t(p Pt) float64 {
	if g.R <= 0 {
		return 1
	}
	if g.F == g.C {
		return math.Hypot(p.X-g.C.X, p.Y-g.C.Y) / g.R
	}
	// Ray from F through p meets the circle (C, R) at distance s; t = |p-F|/s.
	dx, dy := p.X-g.F.X, p.Y-g.F.Y
	d := math.Hypot(dx, dy)
	if d == 0 {
		return 0
	}
	ux, uy := dx/d, dy/d
	fx, fy := g.F.X-g.C.X, g.F.Y-g.C.Y
	b := fx*ux + fy*uy
	c := fx*fx + fy*fy - g.R*g.R
	disc := b*b - c
	if disc < 0 {
		return 1
	}
	s := -b + math.Sqrt(disc)
	if s <= 0 {
		return 1
	}
	return d / s
}

// Pattern paints an image placed by M (image pixels → device), sampled
// bilinearly; outside the image it is transparent.
type Pattern struct {
	Img *Image
	M   Affine
	inv Affine
}

// NewPattern places img with m.
func NewPattern(img *Image, m Affine) *Pattern { return &Pattern{Img: img, M: m, inv: m.Invert()} }

func (p *Pattern) shade(y, x int, out []uint32) {
	for i := range out {
		q := p.inv.Apply(Pt{float64(x+i) + 0.5, float64(y) + 0.5})
		out[i] = p.Img.bilinear(q.X-0.5, q.Y-0.5)
	}
}

// bilinear samples between pixel centres (premultiplied, so it blends
// correctly).
func (m *Image) bilinear(fx, fy float64) uint32 {
	x0, y0 := int(math.Floor(fx)), int(math.Floor(fy))
	tx, ty := fx-float64(x0), fy-float64(y0)
	a, b := m.At(x0, y0), m.At(x0+1, y0)
	c, d := m.At(x0, y0+1), m.At(x0+1, y0+1)
	if a == b && a == c && a == d {
		return a
	}
	w := [4]float64{(1 - tx) * (1 - ty), tx * (1 - ty), (1 - tx) * ty, tx * ty}
	var out uint32
	for sh := 0; sh < 32; sh += 8 {
		v := w[0]*float64(a>>sh&0xff) + w[1]*float64(b>>sh&0xff) + w[2]*float64(c>>sh&0xff) + w[3]*float64(d>>sh&0xff)
		out |= uint32(math.Min(255, v+0.5)) << sh
	}
	return out
}

// Scaled is img resampled to w × h (area-averaged when shrinking).
func (m *Image) Scaled(w, h int) *Image {
	if w == m.W && h == m.H {
		return m
	}
	out := NewImage(w, h)
	if w <= 0 || h <= 0 || m.W == 0 || m.H == 0 {
		return out
	}
	sx, sy := float64(m.W)/float64(w), float64(m.H)/float64(h)
	if sx <= 1 && sy <= 1 {
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				out.Pix[y*out.Stride+x] = m.bilinear((float64(x)+0.5)*sx-0.5, (float64(y)+0.5)*sy-0.5)
			}
		}
		return out
	}
	for y := 0; y < h; y++ {
		y0, y1 := float64(y)*sy, float64(y+1)*sy
		for x := 0; x < w; x++ {
			x0, x1 := float64(x)*sx, float64(x+1)*sx
			var acc [4]float64
			var tot float64
			for yy := int(y0); float64(yy) < y1 && yy < m.H; yy++ {
				wy := math.Min(y1, float64(yy+1)) - math.Max(y0, float64(yy))
				for xx := int(x0); float64(xx) < x1 && xx < m.W; xx++ {
					wx := math.Min(x1, float64(xx+1)) - math.Max(x0, float64(xx))
					p := m.Pix[yy*m.Stride+xx]
					k := wx * wy
					for c := 0; c < 4; c++ {
						acc[c] += k * float64(p>>(8*c)&0xff)
					}
					tot += k
				}
			}
			var p uint32
			for c := 0; c < 4; c++ {
				p |= uint32(math.Min(255, acc[c]/tot+0.5)) << (8 * c)
			}
			out.Pix[y*out.Stride+x] = p
		}
	}
	return out
}
