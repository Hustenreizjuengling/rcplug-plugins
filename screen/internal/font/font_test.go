package font

import (
	"image"
	"math"
	"testing"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
)

func TestFaceTables(t *testing.T) {
	f := Regular()
	if f.UnitsPerEm != 2048 || f.Ascender != 1854 || f.Descender != 434 {
		t.Fatalf("metrics %d %d %d", f.UnitsPerEm, f.Ascender, f.Descender)
	}
	for _, r := range "AaÄöüßØ€0129%" {
		if f.Index(r) == 0 {
			t.Errorf("no glyph for %q", r)
		}
	}
	if f.Index('\U0001F600') != 0 {
		t.Error("an emoji has a glyph?")
	}
	// Liberation Sans is metric-compatible with Arial: 'A' is 1366 units.
	if a := f.Advance(f.Index('A')); a != 1366 {
		t.Errorf("advance of A: %d", a)
	}
	if k := f.Kern(f.Index('A'), f.Index('V')); k >= 0 {
		t.Errorf("A-V kerning %d, want negative", k)
	}
	if p := f.Outline(f.Index('O')); p.Empty() {
		t.Error("O has no outline")
	}
	// Ä is a composite glyph (A + dieresis): it must draw more than A.
	sz := f.Sized(40)
	if ink(sz, "Ä") <= ink(sz, "A") {
		t.Error("composite Ä draws no more than A")
	}
}

// ink is how much a text covers when drawn.
func ink(f *Font, s string) float64 {
	m := gfx.NewImage(200, 80)
	f.Draw(m, s, 10, 60, gfx.Hex(0xffffff), m.Bounds())
	var a float64
	for _, p := range m.Pix {
		a += float64(p>>24) / 255
	}
	return a
}

func TestSizes(t *testing.T) {
	f := Regular().Sized(22.88) // 2288 font units tall: 0.01 px per unit
	if math.Abs(f.Scale()-0.01) > 1e-12 {
		t.Fatalf("scale %v", f.Scale())
	}
	if math.Abs(f.Ascent()-18.54) > 1e-9 || math.Abs(f.Descent()-4.34) > 1e-9 {
		t.Fatalf("ascent %v descent %v", f.Ascent(), f.Descent())
	}
	w := f.Width("AV")
	want := (1366.0 + 1366 + float64(f.Face.Kern(f.Face.Index('A'), f.Face.Index('V')))) * 0.01
	if math.Abs(w-want) > 1e-9 {
		t.Fatalf("width %v, want %v", w, want)
	}
}

func TestDrawPlacement(t *testing.T) {
	f := Bold().Sized(30)
	m := gfx.NewImage(100, 50)
	f.Draw(m, "H", 10, 40, gfx.Hex(0xff0000), m.Bounds())
	// H stands on the baseline: ink in its rows above 40, none below.
	var above, below float64
	minX := 100
	for y := 0; y < 50; y++ {
		for x := 0; x < 100; x++ {
			a := float64(m.At(x, y)>>24) / 255
			if y < 40 {
				above += a
			} else {
				below += a
			}
			if a > 0.5 && x < minX {
				minX = x
			}
		}
	}
	if above < 50 || below > 1 {
		t.Fatalf("ink above %.1f below %.1f", above, below)
	}
	if minX < 10 || minX > 14 {
		t.Fatalf("H starts at x=%d", minX)
	}
	if m.At(minX+1, 30)&0xffffff != 0xff0000 {
		t.Fatalf("colour %08x", m.At(minX+1, 30))
	}
	// Clipped: nothing outside the box.
	m = gfx.NewImage(100, 50)
	f.Draw(m, "HHHH", 10, 40, gfx.Hex(0xffffff), image.Rect(0, 0, 25, 50))
	for y := 0; y < 50; y++ {
		for x := 25; x < 100; x++ {
			if m.At(x, y) != 0 {
				t.Fatalf("drawn outside the clip at %d,%d", x, y)
			}
		}
	}
}

func TestFitAndWrap(t *testing.T) {
	f := Regular().Sized(20)
	if got := f.Fit("SHORT", 500); got != "SHORT" {
		t.Fatalf("fit %q", got)
	}
	got := f.Fit("A LONG LABEL THAT DOES NOT FIT", 120)
	if f.Width(got) > 120 || got[len(got)-3:] != Ellipsis {
		t.Fatalf("fit %q (%v px)", got, f.Width(got))
	}
	lines := f.Wrap("ALL CHANNELS WILL BE INCLUDED IN THE HEADPHONE OUTPUT.", 250)
	if len(lines) < 2 {
		t.Fatalf("wrap %q", lines)
	}
	for _, l := range lines {
		if f.Width(l) > 250 && len(l) > 0 && !containsSpace(l) {
			continue
		}
		if f.Width(l) > 250 {
			t.Fatalf("line %q too wide", l)
		}
	}
}

func containsSpace(s string) bool {
	for _, r := range s {
		if r == ' ' {
			return true
		}
	}
	return false
}
