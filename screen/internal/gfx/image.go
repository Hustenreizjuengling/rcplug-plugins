// Package gfx draws the screen's pictures in software: images of
// premultiplied ARGB pixels, anti-aliased paths with both fill rules,
// strokes, gradients and image drawing. It is small on purpose and uses
// only the standard library; everything the screen shows goes through it.
package gfx

import (
	"image"
	"image/color"
)

// Color is a straight (not premultiplied) colour, 0xAARRGGBB.
type Color uint32

// RGB is an opaque colour.
func RGB(r, g, b uint8) Color { return Color(0xff000000 | uint32(r)<<16 | uint32(g)<<8 | uint32(b)) }

// RGBA is a colour with alpha (straight).
func RGBA(r, g, b, a uint8) Color {
	return Color(uint32(a)<<24 | uint32(r)<<16 | uint32(g)<<8 | uint32(b))
}

// Hex is a colour from 0xRRGGBB.
func Hex(rgb uint32) Color { return Color(0xff000000 | rgb&0xffffff) }

func (c Color) A() uint8 { return uint8(c >> 24) }
func (c Color) R() uint8 { return uint8(c >> 16) }
func (c Color) G() uint8 { return uint8(c >> 8) }
func (c Color) B() uint8 { return uint8(c) }

// WithAlpha multiplies the colour's alpha by a (0…1).
func (c Color) WithAlpha(a float64) Color {
	na := int(float64(c.A())*a + 0.5)
	na = max(0, min(255, na))
	return Color(uint32(c)&0xffffff | uint32(na)<<24)
}

// Lerp mixes the colour with d: t 0 is the colour, 1 is d (straight alpha).
func (c Color) Lerp(d Color, t float64) Color {
	if t <= 0 {
		return c
	}
	if t >= 1 {
		return d
	}
	mix := func(a, b uint8) uint8 { return uint8(float64(a) + (float64(b)-float64(a))*t + 0.5) }
	return RGBA(mix(c.R(), d.R()), mix(c.G(), d.G()), mix(c.B(), d.B()), mix(c.A(), d.A()))
}

// Premul is the colour as a premultiplied pixel.
func (c Color) Premul() uint32 {
	a := uint32(c >> 24)
	if a == 255 {
		return uint32(c)
	}
	r := (uint32(c>>16&0xff)*a + 127) / 255
	g := (uint32(c>>8&0xff)*a + 127) / 255
	b := (uint32(c&0xff)*a + 127) / 255
	return a<<24 | r<<16 | g<<8 | b
}

// Mix is the colour between a (t = 0) and b (t = 1).
func Mix(a, b Color, t float64) Color {
	l := func(x, y uint8) uint32 { return uint32(float64(x) + (float64(y)-float64(x))*t + 0.5) }
	return Color(l(a.A(), b.A())<<24 | l(a.R(), b.R())<<16 | l(a.G(), b.G())<<8 | l(a.B(), b.B()))
}

// Image is a picture of premultiplied ARGB pixels (0xAARRGGBB).
type Image struct {
	W, H   int
	Stride int // pixels per row
	Pix    []uint32
}

// NewImage is a transparent image.
func NewImage(w, h int) *Image {
	return &Image{W: w, H: h, Stride: w, Pix: make([]uint32, w*h)}
}

// Bounds is the image's rectangle.
func (m *Image) Bounds() image.Rectangle { return image.Rect(0, 0, m.W, m.H) }

// At is the premultiplied pixel at x, y (0 outside).
func (m *Image) At(x, y int) uint32 {
	if x < 0 || y < 0 || x >= m.W || y >= m.H {
		return 0
	}
	return m.Pix[y*m.Stride+x]
}

// Sub is the part r of the image, sharing its pixels; its origin is r.Min.
func (m *Image) Sub(r image.Rectangle) *Image {
	r = r.Intersect(m.Bounds())
	if r.Empty() {
		return &Image{}
	}
	return &Image{W: r.Dx(), H: r.Dy(), Stride: m.Stride, Pix: m.Pix[r.Min.Y*m.Stride+r.Min.X:]}
}

// Clear sets every pixel to c.
func (m *Image) Clear(c Color) { m.Set(m.Bounds(), c) }

// Set replaces the pixels in r with c (no blending).
func (m *Image) Set(r image.Rectangle, c Color) {
	r = r.Intersect(m.Bounds())
	if r.Empty() {
		return
	}
	p := c.Premul()
	// The first row pixel by pixel, the rest copied from it.
	first := m.Pix[r.Min.Y*m.Stride+r.Min.X : r.Min.Y*m.Stride+r.Max.X]
	for i := range first {
		first[i] = p
	}
	for y := r.Min.Y + 1; y < r.Max.Y; y++ {
		copy(m.Pix[y*m.Stride+r.Min.X:y*m.Stride+r.Max.X], first)
	}
}

// FillRect blends c over the pixels in r.
func (m *Image) FillRect(r image.Rectangle, c Color) {
	if c.A() == 255 {
		m.Set(r, c)
		return
	}
	if c.A() == 0 {
		return
	}
	r = r.Intersect(m.Bounds())
	p := c.Premul()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		row := m.Pix[y*m.Stride+r.Min.X : y*m.Stride+r.Max.X]
		for i := range row {
			row[i] = over(p, row[i])
		}
	}
}

// Draw blends src over m with its top left at (x, y), with extra opacity
// alpha (0…255).
func (m *Image) Draw(src *Image, x, y int, alpha uint8) {
	m.DrawIn(src, x, y, alpha, m.Bounds())
}

// DrawIn is Draw limited to the clip rectangle.
func (m *Image) DrawIn(src *Image, x, y int, alpha uint8, clip image.Rectangle) {
	dr := image.Rect(x, y, x+src.W, y+src.H).Intersect(m.Bounds()).Intersect(clip)
	if dr.Empty() || alpha == 0 {
		return
	}
	for yy := dr.Min.Y; yy < dr.Max.Y; yy++ {
		srow := src.Pix[(yy-y)*src.Stride+(dr.Min.X-x):]
		drow := m.Pix[yy*m.Stride+dr.Min.X : yy*m.Stride+dr.Max.X]
		if alpha == 255 {
			for i := range drow {
				drow[i] = over(srow[i], drow[i])
			}
			continue
		}
		for i := range drow {
			drow[i] = over(scale(srow[i], uint32(alpha)), drow[i])
		}
	}
}

// Copy replaces the pixels at (x, y) with src (no blending).
func (m *Image) Copy(src *Image, x, y int) {
	dr := image.Rect(x, y, x+src.W, y+src.H).Intersect(m.Bounds())
	for yy := dr.Min.Y; yy < dr.Max.Y; yy++ {
		copy(m.Pix[yy*m.Stride+dr.Min.X:yy*m.Stride+dr.Max.X], src.Pix[(yy-y)*src.Stride+(dr.Min.X-x):])
	}
}

// Tint draws the mask (an image whose alpha is used) in colour c at (x, y):
// for icons drawn in one colour.
func (m *Image) Tint(mask *Image, x, y int, c Color) {
	dr := image.Rect(x, y, x+mask.W, y+mask.H).Intersect(m.Bounds())
	p := c.Premul()
	for yy := dr.Min.Y; yy < dr.Max.Y; yy++ {
		srow := mask.Pix[(yy-y)*mask.Stride+(dr.Min.X-x):]
		drow := m.Pix[yy*m.Stride+dr.Min.X : yy*m.Stride+dr.Max.X]
		for i := range drow {
			if a := srow[i] >> 24; a != 0 {
				drow[i] = over(scale(p, a), drow[i])
			}
		}
	}
}

// over is src over dst, both premultiplied.
func over(src, dst uint32) uint32 {
	sa := src >> 24
	if sa == 255 {
		return src
	}
	if sa == 0 {
		return dst
	}
	ia := 255 - sa
	rb := (dst & 0x00ff00ff) * ia
	ag := (dst >> 8 & 0x00ff00ff) * ia
	rb = (rb + 0x00800080 + (rb >> 8 & 0x00ff00ff)) >> 8 & 0x00ff00ff
	ag = (ag + 0x00800080 + (ag >> 8 & 0x00ff00ff)) & 0xff00ff00
	return src + (rb | ag)
}

// scale multiplies a premultiplied pixel by a/255.
func scale(p, a uint32) uint32 {
	if a == 255 {
		return p
	}
	rb := (p & 0x00ff00ff) * a
	ag := (p >> 8 & 0x00ff00ff) * a
	rb = (rb + 0x00800080 + (rb >> 8 & 0x00ff00ff)) >> 8 & 0x00ff00ff
	ag = (ag + 0x00800080 + (ag >> 8 & 0x00ff00ff)) & 0xff00ff00
	return rb | ag
}

// ToNRGBA converts the image for encoding (PNG) and tests.
func (m *Image) ToNRGBA() *image.NRGBA {
	out := image.NewNRGBA(m.Bounds())
	for y := 0; y < m.H; y++ {
		for x := 0; x < m.W; x++ {
			p := m.Pix[y*m.Stride+x]
			a := p >> 24
			o := out.PixOffset(x, y)
			if a == 0 {
				continue
			}
			r, g, b := p>>16&0xff, p>>8&0xff, p&0xff
			if a < 255 {
				r, g, b = (r*255+a/2)/a, (g*255+a/2)/a, (b*255+a/2)/a
			}
			out.Pix[o], out.Pix[o+1], out.Pix[o+2], out.Pix[o+3] = uint8(min(r, 255)), uint8(min(g, 255)), uint8(min(b, 255)), uint8(a)
		}
	}
	return out
}

// FromImage converts any image (a decoded PNG) into premultiplied pixels.
func FromImage(src image.Image) *Image {
	b := src.Bounds()
	m := NewImage(b.Dx(), b.Dy())
	for y := 0; y < m.H; y++ {
		for x := 0; x < m.W; x++ {
			c := color.NRGBAModel.Convert(src.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA)
			m.Pix[y*m.Stride+x] = RGBA(c.R, c.G, c.B, c.A).Premul()
		}
	}
	return m
}
