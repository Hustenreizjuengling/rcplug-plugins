package gfx

import (
	"image"
	"math"
	"testing"
)

// alphaSum is the covered area in pixels (alpha summed).
func alphaSum(m *Image) float64 {
	var s float64
	for y := 0; y < m.H; y++ {
		for x := 0; x < m.W; x++ {
			s += float64(m.Pix[y*m.Stride+x]>>24) / 255
		}
	}
	return s
}

func TestFillAreas(t *testing.T) {
	cases := []struct {
		name string
		path func(p *Path)
		rule FillRule
		want float64
	}{
		{"rect on the grid", func(p *Path) { p.Rect(10, 10, 20, 30) }, NonZero, 600},
		{"rect off the grid", func(p *Path) { p.Rect(10.3, 10.7, 20.2, 30.1) }, NonZero, 20.2 * 30.1},
		{"circle", func(p *Path) { p.Circle(50, 50, 30) }, NonZero, math.Pi * 900},
		{"triangle", func(p *Path) { p.MoveTo(5, 5); p.LineTo(85, 15); p.LineTo(30, 90); p.Close() }, NonZero, 0.5 * math.Abs((85-5)*(90-5)-(30-5)*(15-5))},
		{"two overlapping rects, non-zero", func(p *Path) { p.Rect(10, 10, 40, 40); p.Rect(30, 30, 40, 40) }, NonZero, 1600*2 - 400},
		{"two overlapping rects, even-odd", func(p *Path) { p.Rect(10, 10, 40, 40); p.Rect(30, 30, 40, 40) }, EvenOdd, 1600*2 - 800},
		{"ring by even-odd", func(p *Path) { p.Circle(50, 50, 40); p.Circle(50, 50, 20) }, EvenOdd, math.Pi * (1600 - 400)},
		{"round rect", func(p *Path) { p.RoundRect(10, 10, 60, 40, 10, 10) }, NonZero, 60*40 - (4-math.Pi)*100},
		{"partly outside left and top", func(p *Path) { p.Rect(-20, -10, 50, 40) }, NonZero, 30 * 30},
		{"partly outside right and bottom", func(p *Path) { p.Rect(80, 90, 50, 40) }, NonZero, 20 * 10},
		{"slanted edge leaving the right side", func(p *Path) { p.MoveTo(50, 0); p.LineTo(150, 100); p.LineTo(50, 100); p.Close() }, NonZero, 50*100 - 0.5*50*50},
	}
	for _, c := range cases {
		m := NewImage(100, 100)
		p := &Path{}
		c.path(p)
		m.Fill(p, Identity, c.rule, Solid(Hex(0xffffff)), m.Bounds())
		got := alphaSum(m)
		if math.Abs(got-c.want) > math.Max(1.5, c.want*0.004) {
			t.Errorf("%s: area %.2f, want %.2f", c.name, got, c.want)
		}
	}
}

func TestFillIsClipped(t *testing.T) {
	m := NewImage(50, 50)
	p := &Path{}
	p.Rect(0, 0, 50, 50)
	m.Fill(p, Identity, NonZero, Solid(Hex(0xff0000)), image.Rect(10, 10, 20, 20))
	if a := alphaSum(m); a != 100 {
		t.Fatalf("area %v inside a 10x10 clip", a)
	}
	if m.At(9, 9) != 0 || m.At(10, 10) != 0xffff0000 {
		t.Fatalf("pixels %08x %08x", m.At(9, 9), m.At(10, 10))
	}
}

func TestTransform(t *testing.T) {
	m := NewImage(100, 100)
	p := &Path{}
	p.Rect(0, 0, 10, 10)
	m.Fill(p, Scale(2, 3).Then(Translate(20, 10)), NonZero, Solid(Hex(0xffffff)), m.Bounds())
	if a := alphaSum(m); math.Abs(a-600) > 0.5 {
		t.Fatalf("area %v, want 600", a)
	}
	if m.At(20, 10)>>24 != 255 || m.At(19, 10) != 0 || m.At(39, 39)>>24 != 255 || m.At(40, 40) != 0 {
		t.Fatal("scaled rect in the wrong place")
	}
	a := Rotate(0.3).Then(Translate(5, 7))
	q := a.Invert().Apply(a.Apply(Pt{3, 4}))
	if math.Abs(q.X-3) > 1e-9 || math.Abs(q.Y-4) > 1e-9 {
		t.Fatalf("invert: %v", q)
	}
}

func TestStroke(t *testing.T) {
	m := NewImage(200, 100)
	p := &Path{}
	p.MoveTo(20, 50)
	p.LineTo(180, 50)
	m.StrokePath(p, Identity, Stroke{Width: 6, Cap: ButtCap}, Solid(Hex(0xffffff)), m.Bounds())
	if a := alphaSum(m); math.Abs(a-160*6) > 2 {
		t.Fatalf("butt line area %v, want 960", a)
	}
	m = NewImage(200, 100)
	m.StrokePath(p, Identity, Stroke{Width: 6, Cap: RoundCap}, Solid(Hex(0xffffff)), m.Bounds())
	if a := alphaSum(m); math.Abs(a-(160*6+math.Pi*9)) > 3 {
		t.Fatalf("round line area %v", a)
	}
	// A closed square outline: four sides, corners filled by the joins.
	m = NewImage(100, 100)
	p = &Path{}
	p.Rect(20, 20, 60, 60)
	m.StrokePath(p, Identity, Stroke{Width: 4, Join: MiterJoin}, Solid(Hex(0xffffff)), m.Bounds())
	if a, want := alphaSum(m), 64.0*64-56*56; math.Abs(a-want) > 3 {
		t.Fatalf("square outline area %v, want %v", a, want)
	}
	// A circle outline.
	m = NewImage(100, 100)
	p = &Path{}
	p.Circle(50, 50, 30)
	m.StrokePath(p, Identity, Stroke{Width: 2, Join: RoundJoin}, Solid(Hex(0xffffff)), m.Bounds())
	if a, want := alphaSum(m), math.Pi*(31*31-29*29); math.Abs(a-want) > want*0.02 {
		t.Fatalf("circle outline area %v, want %v", a, want)
	}
}

func TestBlend(t *testing.T) {
	m := NewImage(1, 1)
	m.Clear(Hex(0x000000))
	m.FillRect(m.Bounds(), RGBA(255, 255, 255, 128))
	if p := m.At(0, 0); p != 0xff808080 {
		t.Fatalf("half white over black: %08x", p)
	}
	m.Clear(Hex(0x204060))
	m.FillRect(m.Bounds(), RGBA(0, 0, 0, 0))
	if p := m.At(0, 0); p != 0xff204060 {
		t.Fatalf("transparent changed the pixel: %08x", p)
	}
	if over(0xff112233, 0xff445566) != 0xff112233 {
		t.Fatal("opaque over")
	}
	if c := RGBA(255, 0, 0, 128).Premul(); c != 0x80800000 {
		t.Fatalf("premul %08x", c)
	}
	n := NewImage(2, 1)
	n.Pix[0] = RGBA(255, 128, 0, 128).Premul()
	n.Pix[1] = Hex(0x00ff00).Premul()
	got := n.ToNRGBA()
	if got.Pix[0] != 255 || got.Pix[1] != 128 || got.Pix[3] != 128 || got.Pix[5] != 255 {
		t.Fatalf("unpremultiplied %v", got.Pix)
	}
}

func TestGradients(t *testing.T) {
	m := NewImage(101, 1)
	p := &Path{}
	p.Rect(0, 0, 101, 1)
	g := NewLinear(Pt{0, 0}, Pt{101, 0}, Identity, []Stop{{0, Hex(0x000000)}, {1, Hex(0xffffff)}})
	m.Fill(p, Identity, NonZero, g, m.Bounds())
	if l, r := m.At(0, 0)&0xff, m.At(100, 0)&0xff; l > 3 || r < 252 {
		t.Fatalf("linear ends %d %d", l, r)
	}
	if mid := m.At(50, 0) & 0xff; mid < 120 || mid > 136 {
		t.Fatalf("linear middle %d", mid)
	}
	m = NewImage(101, 101)
	p = &Path{}
	p.Rect(0, 0, 101, 101)
	r := NewRadial(Pt{50.5, 50.5}, Pt{50.5, 50.5}, 50, Identity, []Stop{{0, Hex(0xffffff)}, {1, Hex(0x000000)}})
	m.Fill(p, Identity, NonZero, r, m.Bounds())
	if c, e := m.At(50, 50)&0xff, m.At(0, 50)&0xff; c < 250 || e > 5 {
		t.Fatalf("radial centre %d edge %d", c, e)
	}
}

func TestScaled(t *testing.T) {
	m := NewImage(4, 4)
	m.Clear(Hex(0xffffff))
	m.Set(image.Rect(0, 0, 2, 4), Hex(0x000000))
	s := m.Scaled(2, 2)
	if s.At(0, 0) != 0xff000000 || s.At(1, 1) != 0xffffffff {
		t.Fatalf("halved: %08x %08x", s.At(0, 0), s.At(1, 1))
	}
	one := m.Scaled(1, 1).At(0, 0) & 0xff
	if one < 126 || one > 129 {
		t.Fatalf("average %d", one)
	}
}

func TestStrokeCorners(t *testing.T) {
	// An L with a miter corner, drawn both ways round (the corner turns
	// left one way and right the other).
	for _, pts := range [][]Pt{{{20, 20}, {80, 20}, {80, 80}}, {{80, 80}, {80, 20}, {20, 20}}} {
		for _, j := range []Join{MiterJoin, BevelJoin, RoundJoin} {
			m := NewImage(100, 100)
			p := &Path{}
			p.MoveTo(pts[0].X, pts[0].Y)
			p.LineTo(pts[1].X, pts[1].Y)
			p.LineTo(pts[2].X, pts[2].Y)
			m.StrokePath(p, Identity, Stroke{Width: 4, Join: j}, Solid(Hex(0xffffff)), m.Bounds())
			// The arms (60×4 each, their 2×2 overlap once) and the corner
			// square up to the vertical arm's outer edge, plus the outer
			// corner: a 2×2 square for a miter, half of it for a bevel, a
			// quarter circle of radius 2 for round.
			want := 240.0 + 240 - 4
			switch j {
			case MiterJoin:
				want += 4
			case BevelJoin:
				want += 2
			case RoundJoin:
				want += math.Pi
			}
			if a := alphaSum(m); math.Abs(a-want) > 1.5 {
				t.Errorf("%v join %d: area %.2f, want %.2f", pts, j, a, want)
			}
			if m.At(80, 50)>>24 != 255 || m.At(50, 20)>>24 != 255 || m.At(70, 30) != 0 {
				t.Errorf("%v join %d: the L is not where it should be", pts, j)
			}
		}
	}
}
