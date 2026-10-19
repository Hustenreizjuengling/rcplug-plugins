package panel

import (
	"image"
	"testing"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
)

// readPanel reads a pixel from a portrait XRGB buffer.
func readPanel(dst []byte, pitch, x, y int) uint32 {
	o := y*pitch + x*4
	return uint32(dst[o]) | uint32(dst[o+1])<<8 | uint32(dst[o+2])<<16
}

func TestTurnAndPageOf(t *testing.T) {
	// A landscape page 1280 × 720; the panel is 720 × 1280.
	page := gfx.NewImage(1280, 720)
	page.Set(page.Bounds(), gfx.Hex(0x111111))
	page.Pix[0] = gfx.Hex(0xff0000).Premul()                               // page top left
	page.Pix[(page.H-1)*page.Stride+page.W-1] = gfx.Hex(0x00ff00).Premul() // page bottom right
	for _, turn := range []int{90, 270} {
		pw, ph := 720, 1280
		pitch := pw*4 + 32
		dst := make([]byte, pitch*ph)
		Turn(page, page.Bounds(), dst, pitch, turn)
		// A point on the panel maps back to the page: the colour there must
		// match page.At of PageOf.
		for _, pp := range []image.Point{{0, 0}, {10, 20}, {pw - 1, ph - 1}, {700, 5}} {
			pg := PageOf(pp, pw, ph, turn)
			want := page.At(pg.X, pg.Y) & 0xffffff
			got := readPanel(dst, pitch, pp.X, pp.Y)
			if got != want {
				t.Fatalf("turn %d: panel %v → page %v: %06x, want %06x", turn, pp, pg, got, want)
			}
		}
	}
	// On the Duo (90°) the page's top left is the panel's top right.
	page2 := gfx.NewImage(1280, 720)
	page2.Pix[0] = gfx.Hex(0xff0000).Premul()
	dst := make([]byte, (720*4)*1280)
	Turn(page2, page2.Bounds(), dst, 720*4, 90)
	if readPanel(dst, 720*4, 719, 0) != 0xff0000 {
		t.Fatalf("page top left is not the panel's top right: %06x", readPanel(dst, 720*4, 719, 0))
	}
}

func TestTurnPartial(t *testing.T) {
	page := gfx.NewImage(1280, 720)
	page.Set(image.Rect(100, 100, 200, 200), gfx.Hex(0x00ff00))
	pitch := 720 * 4
	dst := make([]byte, pitch*1280)
	Turn(page, image.Rect(100, 100, 200, 200), dst, pitch, 90)
	// Inside the copied rectangle the panel is green; a point from outside
	// stays black.
	pg := image.Pt(150, 150)
	for y := 0; y < 1280; y++ {
		for x := 0; x < 720; x++ {
			p := PageOf(image.Pt(x, y), 720, 1280, 90)
			inside := p.In(image.Rect(100, 100, 200, 200))
			if got := readPanel(dst, pitch, x, y); inside && got != 0x00ff00 {
				t.Fatalf("inside %v not copied", p)
			} else if !inside && got != 0 {
				t.Fatalf("outside %v changed", p)
			}
		}
	}
	_ = pg
}
