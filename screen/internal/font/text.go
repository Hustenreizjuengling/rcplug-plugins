package font

import (
	"image"
	"math"
	"strings"
	"sync"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
)

// Font is a face at one size.
type Font struct {
	Face   *Face
	Height float64 // pixels: ascent + descent (JUCE's font height)
	scale  float64 // pixels per font unit

	mu    sync.Mutex
	cache map[glyphKey]*glyphMask
}

type glyphKey struct {
	g   uint16
	sub uint8 // quarter pixels
}

// glyphMask is a rendered glyph: white with the coverage as alpha, placed
// at (dx, dy) from the pen on the baseline.
type glyphMask struct {
	img    *gfx.Image
	dx, dy int
}

// Sized is face at height pixels.
func (f *Face) Sized(height float64) *Font {
	return &Font{Face: f, Height: height, scale: height / f.heightUnits(), cache: map[glyphKey]*glyphMask{}}
}

// Ascent and Descent are the font's extent above and below the baseline.
func (f *Font) Ascent() float64  { return float64(f.Face.Ascender) * f.scale }
func (f *Font) Descent() float64 { return float64(f.Face.Descender) * f.scale }

// Scale is pixels per font unit.
func (f *Font) Scale() float64 { return f.scale }

// Width is how wide s is drawn.
func (f *Font) Width(s string) float64 {
	var w float64
	var prev uint16
	for i, r := range s {
		g := f.Face.Index(r)
		if i > 0 {
			w += float64(f.Face.Kern(prev, g)) * f.scale
		}
		w += float64(f.Face.Advance(g)) * f.scale
		prev = g
	}
	return w
}

// Draw draws s with its baseline starting at (x, y).
func (f *Font) Draw(dst *gfx.Image, s string, x, y float64, c gfx.Color, clip image.Rectangle) {
	if c.A() == 0 {
		return
	}
	target := dst
	if clip != dst.Bounds() {
		clip = clip.Intersect(dst.Bounds())
		if clip.Empty() {
			return
		}
		target = dst.Sub(clip)
		x -= float64(clip.Min.X)
		y -= float64(clip.Min.Y)
	}
	by := int(math.Floor(y + 0.5))
	var prev uint16
	first := true
	for _, r := range s {
		g := f.Face.Index(r)
		if !first {
			x += float64(f.Face.Kern(prev, g)) * f.scale
		}
		first = false
		xi := math.Floor(x)
		sub := uint8(math.Floor((x-xi)*4)) & 3
		if m := f.glyph(g, sub); m != nil {
			target.Tint(m.img, int(xi)+m.dx, by+m.dy, c)
		}
		x += float64(f.Face.Advance(g)) * f.scale
		prev = g
	}
}

func (f *Font) glyph(g uint16, sub uint8) *glyphMask {
	k := glyphKey{g, sub}
	f.mu.Lock()
	m, ok := f.cache[k]
	f.mu.Unlock()
	if ok {
		return m
	}
	p := f.Face.Outline(g)
	if !p.Empty() {
		tr := gfx.Scale(f.scale, f.scale).Then(gfx.Translate(float64(sub)/4, 0))
		lo, hi := p.Transform(tr).Bounds()
		x0, y0 := int(math.Floor(lo.X))-1, int(math.Floor(lo.Y))-1
		x1, y1 := int(math.Ceil(hi.X))+1, int(math.Ceil(hi.Y))+1
		img := gfx.NewImage(x1-x0, y1-y0)
		img.Fill(p, tr.Then(gfx.Translate(float64(-x0), float64(-y0))), gfx.NonZero, gfx.Solid(gfx.Hex(0xffffff)), img.Bounds())
		m = &glyphMask{img: img, dx: x0, dy: y0}
	}
	f.mu.Lock()
	f.cache[k] = m
	f.mu.Unlock()
	return m
}

// Ellipsis is what ends a text cut short.
const Ellipsis = "..."

// Fit is s, cut short with an ellipsis if it is wider than w.
func (f *Font) Fit(s string, w float64) string {
	if f.Width(s) <= w {
		return s
	}
	rs := []rune(s)
	for n := len(rs) - 1; n > 0; n-- {
		t := strings.TrimRight(string(rs[:n]), " ") + Ellipsis
		if f.Width(t) <= w {
			return t
		}
	}
	return Ellipsis
}

// Wrap breaks s into lines no wider than w, at spaces where it can (a
// newline always breaks).
func (f *Font) Wrap(s string, w float64) []string {
	var out []string
	for _, para := range strings.Split(s, "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		line := words[0]
		for _, wd := range words[1:] {
			if f.Width(line+" "+wd) <= w {
				line += " " + wd
				continue
			}
			out = append(out, line)
			line = wd
		}
		out = append(out, line)
	}
	return out
}

// Align places text horizontally.
type Align uint8

const (
	Left Align = iota
	Center
	Right
)

// DrawIn draws s in the box r: one line, vertically centred, aligned as
// asked, cut short with an ellipsis if it does not fit.
func (f *Font) DrawIn(dst *gfx.Image, s string, r image.Rectangle, a Align, c gfx.Color) {
	s = f.Fit(s, float64(r.Dx()))
	w := f.Width(s)
	x := float64(r.Min.X)
	switch a {
	case Center:
		x += (float64(r.Dx()) - w) / 2
	case Right:
		x += float64(r.Dx()) - w
	}
	// Centre the font's height (ascent + descent), as JUCE does.
	y := float64(r.Min.Y) + (float64(r.Dy())-f.Height)/2 + f.Ascent()
	f.Draw(dst, s, x, y, c, r)
}
