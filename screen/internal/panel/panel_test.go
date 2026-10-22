package panel

import (
	"bytes"
	"image"
	"math/rand"
	"testing"
	"unsafe"

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

// turnSlow is Turn as it was first written, one pixel at a time: the
// reference the tiled copy must match, whatever the rectangle.
func turnSlow(page *gfx.Image, r image.Rectangle, dst []byte, pitch, turn int) {
	r = r.Intersect(page.Bounds())
	px := unsafe.Slice((*uint32)(unsafe.Pointer(&dst[0])), len(dst)/4)
	stride := pitch / 4
	pw, ph := page.H, page.W
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			v := page.Pix[y*page.Stride+x] | 0xff000000
			o := x*stride + pw - 1 - y
			if turn == 270 {
				o = (ph-1-x)*stride + y
			}
			if o < len(px) {
				px[o] = v
			}
		}
	}
}

func TestTurnMatchesReference(t *testing.T) {
	page := gfx.NewImage(1280, 720)
	rnd := rand.New(rand.NewSource(1))
	for i := range page.Pix {
		page.Pix[i] = uint32(rnd.Int63()) & 0xffffff
	}
	rects := []image.Rectangle{
		page.Bounds(), image.Rect(0, 64, 592, 656), image.Rect(3, 5, 7, 9),
		image.Rect(100, 100, 101, 101), image.Rect(1200, 700, 1280, 720), image.Rect(31, 31, 65, 97),
		image.Rect(-10, -10, 40, 40), image.Rect(1270, 710, 1300, 730),
	}
	for _, turn := range []int{90, 270} {
		for _, pitch := range []int{720 * 4, 720*4 + 64} {
			for _, r := range rects {
				want := make([]byte, pitch*1280)
				got := make([]byte, pitch*1280)
				turnSlow(page, r, want, pitch, turn)
				Turn(page, r, got, pitch, turn)
				if !bytes.Equal(want, got) {
					t.Fatalf("turn %d pitch %d rect %v: the tiled copy differs from the reference", turn, pitch, r)
				}
			}
		}
	}
	// A buffer shorter than the panel takes what fits and does not panic.
	short := make([]byte, 720*4*100)
	Turn(page, page.Bounds(), short, 720*4, 90)
	Turn(page, page.Bounds(), short, 720*4, 270)
}
