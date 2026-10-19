// Package panel puts pages on the device's display and reads its
// touchscreen and backlight. The display is a portrait panel (720 × 1280)
// that shows a landscape page turned by 90°, as RØDE's screen program
// draws it.
package panel

import (
	"image"
	"unsafe"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
)

// Turn copies the part r of the page (landscape, opaque premultiplied
// ARGB) into a portrait XRGB8888 buffer of pitch bytes per row. turn is how
// the page lies on the panel: 90 = its top on the panel's right (the Duo),
// 270 = on its left. The copy goes in tiles to keep the caches warm.
func Turn(page *gfx.Image, r image.Rectangle, dst []byte, pitch, turn int) {
	r = r.Intersect(page.Bounds())
	if r.Empty() || len(dst) < 4 {
		return
	}
	px := unsafe.Slice((*uint32)(unsafe.Pointer(&dst[0])), len(dst)/4)
	stride := pitch / 4
	pw, ph := page.H, page.W // the panel
	const tile = 32
	for ty := r.Min.Y; ty < r.Max.Y; ty += tile {
		ty1 := min(ty+tile, r.Max.Y)
		for tx := r.Min.X; tx < r.Max.X; tx += tile {
			tx1 := min(tx+tile, r.Max.X)
			for y := ty; y < ty1; y++ {
				src := page.Pix[y*page.Stride:]
				if turn == 270 {
					// Page (x, y) → panel (y, ph-1-x).
					for x := tx; x < tx1; x++ {
						if o := (ph-1-x)*stride + y; o < len(px) {
							px[o] = src[x] | 0xff000000
						}
					}
					continue
				}
				// Page (x, y) → panel (pw-1-y, x).
				col := pw - 1 - y
				for x := tx; x < tx1; x++ {
					if o := x*stride + col; o < len(px) {
						px[o] = src[x] | 0xff000000
					}
				}
			}
		}
	}
}

// PageOf maps a point of the portrait panel (pw × ph pixels) onto the page.
func PageOf(p image.Point, pw, ph, turn int) image.Point {
	if turn == 270 {
		return image.Pt(ph-1-p.Y, p.X)
	}
	return image.Pt(p.Y, pw-1-p.X)
}

// Output shows pages.
type Output interface {
	// Size is the page's size (landscape).
	Size() (w, h int)
	// Present shows the page; dirty lists the parts that changed since the
	// last call (nil: all of it).
	Present(page *gfx.Image, dirty []image.Rectangle) error
	Close()
}

// TouchEvent is the finger going down, moving or going up at a point of the
// portrait panel (as the touchscreen reports it).
type TouchEvent struct {
	Down bool
	At   image.Point
}
