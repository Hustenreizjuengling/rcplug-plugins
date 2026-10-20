package ui

import (
	"image"
	"math"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
)

// Drawing helpers used across the controls and the pages.

// Panel fills a rounded card.
func Panel(dst *gfx.Image, r image.Rectangle, radius int, fill gfx.Color, clip image.Rectangle) {
	if r.Empty() || clip.Intersect(r).Empty() {
		return
	}
	p := &gfx.Path{}
	p.RoundRect(float64(r.Min.X), float64(r.Min.Y), float64(r.Dx()), float64(r.Dy()), float64(radius), float64(radius))
	dst.Fill(p, gfx.Identity, gfx.NonZero, gfx.Solid(fill), clip)
}

// Gradient fills a rounded card with a vertical gradient from top to bottom.
func Gradient(dst *gfx.Image, r image.Rectangle, radius int, top, bottom gfx.Color, clip image.Rectangle) {
	if r.Empty() || clip.Intersect(r).Empty() {
		return
	}
	p := &gfx.Path{}
	p.RoundRect(float64(r.Min.X), float64(r.Min.Y), float64(r.Dx()), float64(r.Dy()), float64(radius), float64(radius))
	g := gfx.NewLinear(gfx.Pt{X: 0, Y: float64(r.Min.Y)}, gfx.Pt{X: 0, Y: float64(r.Max.Y)}, gfx.Identity,
		[]gfx.Stop{{Offset: 0, Color: top}, {Offset: 1, Color: bottom}})
	dst.Fill(p, gfx.Identity, gfx.NonZero, g, clip)
}

// Card draws a tile in the theme's grey gradient.
func Card(dst *gfx.Image, r image.Rectangle, t *Theme, clip image.Rectangle) {
	Gradient(dst, r, t.Radius, t.CardTop, t.CardBottom, clip)
}

// LightCard draws the light card of a chosen option.
func LightCard(dst *gfx.Image, r image.Rectangle, t *Theme, clip image.Rectangle) {
	Gradient(dst, r, t.Radius, t.LightTop, t.LightBottom, clip)
}

// PanelOutline strokes a rounded card's border.
func PanelOutline(dst *gfx.Image, r image.Rectangle, radius int, width float64, col gfx.Color, clip image.Rectangle) {
	if r.Empty() || clip.Intersect(r.Inset(-int(width))).Empty() {
		return
	}
	p := &gfx.Path{}
	h := width / 2
	p.RoundRect(float64(r.Min.X)+h, float64(r.Min.Y)+h, float64(r.Dx())-width, float64(r.Dy())-width, float64(radius), float64(radius))
	dst.StrokePath(p, gfx.Identity, gfx.Stroke{Width: width, Join: gfx.RoundJoin}, gfx.Solid(col), clip)
}

// Circle fills a disc.
func Circle(dst *gfx.Image, cx, cy, radius float64, col gfx.Color, clip image.Rectangle) {
	p := &gfx.Path{}
	p.Circle(cx, cy, radius)
	dst.Fill(p, gfx.Identity, gfx.NonZero, gfx.Solid(col), clip)
}

// Ring strokes a circle.
func Ring(dst *gfx.Image, cx, cy, radius, width float64, col gfx.Color, clip image.Rectangle) {
	p := &gfx.Path{}
	p.Circle(cx, cy, radius)
	dst.StrokePath(p, gfx.Identity, gfx.Stroke{Width: width}, gfx.Solid(col), clip)
}

// Arc strokes part of a circle from angle a0 to a1 (radians, clockwise from
// 3 o'clock, as the screen's y axis points down).
func Arc(dst *gfx.Image, cx, cy, radius, width, a0, a1 float64, col gfx.Color, clip image.Rectangle) {
	if a1 <= a0 {
		return
	}
	p := &gfx.Path{}
	p.Arc(cx, cy, radius, a0, a1)
	dst.StrokePath(p, gfx.Identity, gfx.Stroke{Width: width, Cap: gfx.RoundCap}, gfx.Solid(col), clip)
}

// Focus draws the encoder's focus around a control: a green ring, thicker
// and white while the control is being adjusted.
func Focus(dst *gfx.Image, r image.Rectangle, radius int, adjusting bool, t *Theme, clip image.Rectangle) {
	if adjusting {
		PanelOutline(dst, r.Inset(-4), radius+4, 6, t.Text, clip)
		PanelOutline(dst, r.Inset(-4), radius+4, 2, t.Accent, clip)
		return
	}
	PanelOutline(dst, r.Inset(-3), radius+3, 4, t.Accent, clip)
}

// Text draws one line in the box r, aligned and vertically centred, cut
// short with an ellipsis if it does not fit.
func Text(dst *gfx.Image, f *Font, s string, r image.Rectangle, a font.Align, col gfx.Color, clip image.Rectangle) {
	if s == "" {
		return
	}
	target := dst
	off := image.Point{}
	cr := clip.Intersect(r)
	if cr != dst.Bounds() {
		if cr.Empty() {
			return
		}
		target = dst.Sub(cr)
		off = cr.Min
	}
	f.DrawIn(target, s, r.Sub(off), a, col)
}

// TextWidth is the width of a string in a font (for layouts that follow the
// text).
func TextWidth(f *Font, s string) int { return int(math.Ceil(f.Width(s))) }

// LineHeight is a font's line height.
func LineHeight(f *Font) int { return int(math.Ceil(f.Ascent() + f.Descent())) }

// Lines draws wrapped text from the top of r, each line aligned; it returns
// the height used.
func Lines(dst *gfx.Image, f *Font, s string, r image.Rectangle, a font.Align, col gfx.Color, clip image.Rectangle) int {
	lh := LineHeight(f) + 6
	y := r.Min.Y
	for _, line := range f.Wrap(s, float64(r.Dx())) {
		if y+lh > r.Max.Y+lh/2 {
			break
		}
		Text(dst, f, line, image.Rect(r.Min.X, y, r.Max.X, y+lh), a, col, clip)
		y += lh
	}
	return y - r.Min.Y
}

// Line draws a horizontal divider across the width of r at its top.
func Line(dst *gfx.Image, r image.Rectangle, col gfx.Color, clip image.Rectangle) {
	dst.FillRect(image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+1).Intersect(clip), col)
}

// Fill fills a plain rectangle.
func Fill(dst *gfx.Image, r image.Rectangle, col gfx.Color, clip image.Rectangle) {
	dst.FillRect(r.Intersect(clip), col)
}

// PageBackground paints the gradient behind a menu page.
func PageBackground(dst *gfx.Image, r image.Rectangle, t *Theme, clip image.Rectangle) {
	area := clip.Intersect(r)
	if area.Empty() {
		return
	}
	p := &gfx.Path{}
	p.Rect(float64(area.Min.X), float64(area.Min.Y), float64(area.Dx()), float64(area.Dy()))
	g := gfx.NewLinear(gfx.Pt{X: 0, Y: float64(r.Min.Y)}, gfx.Pt{X: 0, Y: float64(r.Max.Y)}, gfx.Identity,
		[]gfx.Stop{{Offset: 0, Color: t.PageTop}, {Offset: 1, Color: t.PageBottom}})
	dst.Fill(p, gfx.Identity, gfx.NonZero, g, area)
}

// centre is the centre of a rectangle.
func centre(r image.Rectangle) gfx.Pt {
	return gfx.Pt{X: float64(r.Min.X+r.Max.X) / 2, Y: float64(r.Min.Y+r.Max.Y) / 2}
}

// clamp01 keeps v in 0…1.
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// Square is the largest centred square inside r.
func Square(r image.Rectangle) image.Rectangle {
	s := min(r.Dx(), r.Dy())
	x := r.Min.X + (r.Dx()-s)/2
	y := r.Min.Y + (r.Dy()-s)/2
	return image.Rect(x, y, x+s, y+s)
}
