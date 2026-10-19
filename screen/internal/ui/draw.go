package ui

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
)

// Drawing helpers used across the controls.

// Panel fills a rounded card.
func Panel(dst *gfx.Image, r image.Rectangle, radius int, fill gfx.Color, clip image.Rectangle) {
	p := &gfx.Path{}
	p.RoundRect(float64(r.Min.X), float64(r.Min.Y), float64(r.Dx()), float64(r.Dy()), float64(radius), float64(radius))
	dst.Fill(p, gfx.Identity, gfx.NonZero, gfx.Solid(fill), clip)
}

// PanelOutline strokes a rounded card's border.
func PanelOutline(dst *gfx.Image, r image.Rectangle, radius int, width float64, col gfx.Color, clip image.Rectangle) {
	p := &gfx.Path{}
	h := width / 2
	p.RoundRect(float64(r.Min.X)+h, float64(r.Min.Y)+h, float64(r.Dx())-width, float64(r.Dy())-width, float64(radius), float64(radius))
	dst.StrokePath(p, gfx.Identity, gfx.Stroke{Width: width, Join: gfx.RoundJoin}, gfx.Solid(col), clip)
}

// Text draws one line in the box r, aligned and vertically centred, cut
// short with an ellipsis if it does not fit.
func Text(dst *gfx.Image, f *Font, s string, r image.Rectangle, a font.Align, col gfx.Color, clip image.Rectangle) {
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

// Line draws a horizontal divider across the width of r at its top.
func Line(dst *gfx.Image, r image.Rectangle, col gfx.Color, clip image.Rectangle) {
	dst.FillRect(image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+1).Intersect(clip), col)
}

// Fill fills a plain rectangle.
func Fill(dst *gfx.Image, r image.Rectangle, col gfx.Color, clip image.Rectangle) {
	dst.FillRect(r.Intersect(clip), col)
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
