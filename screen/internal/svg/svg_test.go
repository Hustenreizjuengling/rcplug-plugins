package svg

import (
	"math"
	"testing"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
)

func render(t *testing.T, src string, w, h int) *gfx.Image {
	t.Helper()
	d, err := Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	return d.Render(w, h)
}

func covered(m *gfx.Image) float64 {
	var s float64
	for _, p := range m.Pix {
		s += float64(p>>24) / 255
	}
	return s
}

func TestShapesAndScaling(t *testing.T) {
	// A 10×10 viewBox drawn at 100×100: a rect of 5×5 units covers 2500 px.
	m := render(t, `<svg viewBox="0 0 10 10"><rect x="0" y="0" width="5" height="5" fill="#ff0000"/></svg>`, 100, 100)
	if a := covered(m); math.Abs(a-2500) > 1 {
		t.Fatalf("area %v", a)
	}
	if m.At(10, 10) != 0xffff0000 || m.At(60, 60) != 0 {
		t.Fatalf("pixels %08x %08x", m.At(10, 10), m.At(60, 60))
	}
	// Aspect kept and centred: a square picture in a wide image.
	m = render(t, `<svg viewBox="0 0 10 10"><rect width="10" height="10" fill="white"/></svg>`, 200, 100)
	if m.At(49, 50) != 0 || m.At(50, 50) == 0 || m.At(149, 50) == 0 || m.At(150, 50) != 0 {
		t.Fatal("not centred")
	}
	m = render(t, `<svg width="20" height="20"><circle cx="10" cy="10" r="8" fill="white"/></svg>`, 20, 20)
	if a := covered(m); math.Abs(a-math.Pi*64) > 1.6 {
		t.Fatalf("circle area %v", a)
	}
}

func TestPathData(t *testing.T) {
	cases := map[string]float64{
		"M0 0 L10 0 L10 10 L0 10 Z":           100,
		"m0 0 l10 0 l0 10 l-10 0 z":           100,
		"M0,0H10V10H0Z":                       100,
		"M0 0h10v10h-10z":                     100,
		"M0 0 10 0 10 10 0 10z":               100, // implicit line-tos
		"M0-0L10-0L10 10L-0 10z":              100,
		"M5 0A5 5 0 0 1 5 10A5 5 0 0 1 5 0Z":  math.Pi * 25,
		"M5 0a5 5 0 1 1 0 10a5 5 0 1 1 0-10z": math.Pi * 25,
		"M0 0C0 0 10 0 10 0L10 10L0 10Z":      100,
		"M0 0Q5 0 10 0T10 10L0 10Z":           -1, // just parses
		"M.5.5h9v9h-9z":                       81,
		"M0 0h1e1v1e1h-1e1z":                  100,
	}
	for d, want := range cases {
		m := render(t, `<svg viewBox="0 0 10 10"><path fill="white" d="`+d+`"/></svg>`, 10, 10)
		if want < 0 {
			continue
		}
		if a := covered(m); math.Abs(a-want) > math.Max(0.6, want*0.015) {
			t.Errorf("%q: area %.2f, want %.2f", d, a, want)
		}
	}
	// Arc flags packed without separators.
	p := &gfx.Path{}
	if err := parsePathData(p, "M0 5a5 5 0 1010 0z"); err != nil || p.Empty() {
		t.Fatalf("packed flags: %v", err)
	}
}

func TestStyleSources(t *testing.T) {
	src := `<svg viewBox="0 0 4 1">
<style>.a{fill:#00ff00} .b{fill:#0000ff}</style>
<rect x="0" width="1" height="1" class="a"/>
<rect x="1" width="1" height="1" fill="red" style="fill:#ffffff"/>
<g fill="#ff00ff"><rect x="2" width="1" height="1"/></g>
<rect x="3" width="1" height="1" fill="#000000" fill-opacity="0.5"/>
</svg>`
	m := render(t, src, 4, 1)
	want := []uint32{0xff00ff00, 0xffffffff, 0xffff00ff, 0x80000000}
	for i, w := range want {
		if got := m.At(i, 0); got != w {
			t.Errorf("pixel %d: %08x, want %08x", i, got, w)
		}
	}
}

func TestTransformsAndGroups(t *testing.T) {
	m := render(t, `<svg viewBox="0 0 10 10"><g transform="translate(5 0)"><rect width="5" height="5" fill="white" transform="scale(1 2)"/></g></svg>`, 10, 10)
	if m.At(4, 1) != 0 || m.At(6, 1) == 0 || m.At(6, 9) == 0 {
		t.Fatal("translate/scale wrong")
	}
	// Group opacity applies once to the whole group.
	m = render(t, `<svg viewBox="0 0 1 1"><g opacity="0.5"><rect width="1" height="1" fill="#ffffff"/><rect width="1" height="1" fill="#ffffff"/></g></svg>`, 1, 1)
	if a := m.At(0, 0) >> 24; a < 126 || a > 129 {
		t.Fatalf("group opacity gave alpha %d", a)
	}
	if got := parseTransform("rotate(90 5 5)").Apply(gfx.Pt{X: 10, Y: 5}); math.Abs(got.X-5) > 1e-9 || math.Abs(got.Y-10) > 1e-9 {
		t.Fatalf("rotate about a point: %v", got)
	}
}

func TestGradientsAndClip(t *testing.T) {
	src := `<svg viewBox="0 0 100 10">
<defs><linearGradient id="g" x1="0" y1="0" x2="100" y2="0" gradientUnits="userSpaceOnUse">
<stop stop-color="#000000"/><stop offset="1" stop-color="#ffffff"/></linearGradient>
<linearGradient id="h" href="#g" gradientUnits="objectBoundingBox" x1="0" x2="1"/>
<clipPath id="c"><rect width="50" height="10"/></clipPath></defs>
<rect width="100" height="5" fill="url(#g)"/>
<rect y="5" width="100" height="5" fill="url(#h)" clip-path="url(#c)"/>
</svg>`
	m := render(t, src, 100, 10)
	if l, r := m.At(1, 2)&0xff, m.At(98, 2)&0xff; l > 8 || r < 247 {
		t.Fatalf("gradient ends %d %d", l, r)
	}
	if m.At(30, 7) == 0 || m.At(70, 7) != 0 {
		t.Fatal("clip path not applied")
	}
	if mid := m.At(49, 7) & 0xff; mid < 115 || mid > 135 {
		t.Fatalf("inherited stops: %d", mid)
	}
}

func TestColours(t *testing.T) {
	for v, want := range map[string]gfx.Color{
		"#fff": gfx.Hex(0xffffff), "#102030": gfx.Hex(0x102030), "white": gfx.Hex(0xffffff),
		"rgb(255, 0, 0)": gfx.Hex(0xff0000), "rgba(0,0,255,0.5)": gfx.RGBA(0, 0, 255, 128), "#ff000080": gfx.RGBA(255, 0, 0, 128),
	} {
		if got, ok := parseColor(v); !ok || got != want {
			t.Errorf("%s: %08x %v", v, uint32(got), ok)
		}
	}
	if _, ok := parseColor("nonsense"); ok {
		t.Error("nonsense parsed")
	}
}

func TestBrokenInput(t *testing.T) {
	if _, err := Parse([]byte(`<html/>`)); err == nil {
		t.Fatal("not an svg accepted")
	}
	// A stray end tag (as in one of RØDE's files) is skipped.
	m := render(t, `<svg viewBox="0 0 1 1"><rect width="1" height="1" fill="white"/></defs></svg>`, 1, 1)
	if m.At(0, 0) == 0 {
		t.Fatal("drawing after a stray end tag lost")
	}
}
