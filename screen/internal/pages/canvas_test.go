package pages

import (
	"image"
	"testing"
	"time"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// TestCanvasShapes draws every kind of shape and checks that each leaves
// paint inside the canvas and none outside the inset.
func TestCanvasShapes(t *testing.T) {
	ctx := ui.NewContext(ui.Dark(ui.Liberation()), nil)
	kinds := []PluginShape{
		{Shape: "circle", X: 50, Y: 50, R: 10},
		{Shape: "glow", X: 50, Y: 50, R: 20, Alpha: 0.8},
		{Shape: "ring", X: 50, Y: 50, R: 30, Width: 1},
		{Shape: "arc", X: 50, Y: 60, R: 30, A0: 180, A1: 360},
		{Shape: "line", Points: [][2]float64{{10, 10}, {90, 90}}, Width: 2, Color: "#ff00ff"},
		{Shape: "polygon", Points: [][2]float64{{10, 90}, {50, 10}, {90, 90}}, Color: "warn"},
		{Shape: "rect", X: 20, Y: 20, W: 60, H: 20, Radius: 4, Color: "blue"},
		{Shape: "text", X: 50, Y: 50, Text: "Hello", Size: "body"},
		{Shape: "text", X: 200, Y: 200, Text: "outside", Size: "title"}, // clipped
	}
	for _, k := range kinds {
		k := k
		page := &PluginPage{Controls: []PluginControl{{ID: "c", Type: "canvas", W: 100, H: 100, Shapes: []PluginShape{k}}}}
		cv := newCanvas("", func() (PluginControl, *PluginPage, bool) { return page.Controls[0], page, true })
		r := image.Rect(0, 0, 300, 300)
		cv.SetBounds(r)
		img := gfx.NewImage(320, 320)
		cv.Draw(img, ctx, img.Bounds())
		inside, outside := 0, 0
		for y := 0; y < 320; y++ {
			for x := 0; x < 320; x++ {
				px := img.At(x, y) & 0xffffff
				if px == 0 || px == uint32(ctx.Theme.Black)&0xffffff {
					continue
				}
				if image.Pt(x, y).In(ui.Inset(r, 12)) {
					inside++
				} else if image.Pt(x, y).In(r) {
					// the panel's own colour is fine there
				} else {
					outside++
				}
			}
		}
		if k.Text == "outside" {
			if inside != 0 {
				t.Errorf("%s: a shape outside the picture painted %d pixels inside", k.Shape, inside)
			}
			continue
		}
		if inside == 0 {
			t.Errorf("%s painted nothing", k.Shape)
		}
		if outside != 0 {
			t.Errorf("%s painted %d pixels outside its bounds", k.Shape, outside)
		}
	}
}

// TestCanvasGlide: a shape with an id moves part of the way between two
// descriptions; one without jumps.
func TestCanvasGlide(t *testing.T) {
	first := &PluginPage{Controls: []PluginControl{{ID: "c", Type: "canvas", Shapes: []PluginShape{
		{ID: "a", Shape: "circle", X: 0, Y: 0, R: 1}, {Shape: "circle", X: 0, Y: 0, R: 1}}}}}
	second := &PluginPage{Controls: []PluginControl{{ID: "c", Type: "canvas", Shapes: []PluginShape{
		{ID: "a", Shape: "circle", X: 100, Y: 50, R: 3}, {Shape: "circle", X: 100, Y: 50, R: 3}}}}}
	cur := first
	cv := newCanvas("", func() (PluginControl, *PluginPage, bool) { return cur.Controls[0], cur, true })
	cv.at = time.Now().Add(-200 * time.Millisecond)
	cur = second
	now := time.Now()
	if !cv.take(now) {
		t.Fatal("a new description was not taken")
	}
	if cv.period < 50*time.Millisecond || cv.period > 600*time.Millisecond {
		t.Fatalf("period %v", cv.period)
	}
	mid := cv.shapesAt(now.Add(cv.period / 2))
	if a := mid[0]; a.X < 40 || a.X > 60 || a.R < 1.9 || a.R > 2.1 {
		t.Errorf("the shape with an id should be half way: %+v", a)
	}
	if b := mid[1]; b.X != 100 || b.R != 3 {
		t.Errorf("the shape without an id should be at its new place: %+v", b)
	}
	if end := cv.shapesAt(now.Add(cv.period * 2)); end[0].X != 100 {
		t.Errorf("after the period the shape should have arrived: %+v", end[0])
	}
}

func TestCanvasPlace(t *testing.T) {
	r := image.Rect(0, 0, 1220, 592)
	cv, rest := canvasPlace(r, PluginControl{W: 100, H: 100}, 30)
	if cv.Dx() != cv.Dy() || cv.Dy() > 592 || rest.Min.X <= cv.Max.X || rest.Dx() < 400 {
		t.Errorf("square: canvas %v rest %v", cv, rest)
	}
	cv, rest = canvasPlace(r, PluginControl{W: 300, H: 100}, 30)
	if cv.Dy() > 356 || rest.Min.Y <= cv.Max.Y || rest.Dx() != r.Dx() {
		t.Errorf("wide: canvas %v rest %v", cv, rest)
	}
}

// TestCanvasPartial: frames between two descriptions redraw only where the
// moving shapes were and are, and what is on the screen after those
// partial redraws is exactly the picture drawn whole.
func TestCanvasPartial(t *testing.T) {
	ctx := ui.NewContext(ui.Dark(ui.Liberation()), nil)
	th := ctx.Theme
	r := image.Rect(40, 64, 632, 656)
	var pg *PluginPage
	get := func() (PluginControl, *PluginPage, bool) { return pg.Controls[0], pg, true }
	describe := func(az float64) {
		pg = &PluginPage{Controls: []PluginControl{{ID: "c", Type: "canvas", W: 100, H: 100, Shapes: radarShapes(az, true)}}}
	}
	describe(-30)
	now := time.Unix(1000, 0)
	cv := newCanvas("Radar", get)
	cv.now = func() time.Time { return now }
	cv.SetBounds(r)
	live := gfx.NewImage(1280, 720)
	live.Set(live.Bounds(), th.Bg)
	area := func(rs []image.Rectangle) (a int) {
		for _, x := range rs {
			a += x.Dx() * x.Dy()
		}
		return a
	}
	full := r.Dx() * r.Dy()
	glides := 0
	for step := 0; step < 30; step++ {
		if step%3 == 0 {
			describe(-30 + float64(step)*2)
		}
		rects := cv.Dirty()
		if step == 0 && (len(rects) != 1 || rects[0] != r) {
			t.Fatalf("the first frame should draw the whole canvas: %v", rects)
		}
		if step%3 != 0 && step > 3 {
			glides++
			if a := area(rects); a > full/3 {
				t.Errorf("step %d: a gliding frame redraws %d of %d pixels: %v", step, a, full, rects)
			}
		}
		for _, d := range rects { // as the root does
			d = d.Intersect(r)
			live.Set(d, th.Bg)
			cv.Draw(live, ctx, d)
		}
		// The same picture drawn whole by a fresh canvas.
		ref := newCanvas("Radar", get)
		ref.now = cv.now
		ref.SetBounds(r)
		ref.units = cv.units
		ref.next = append([]PluginShape(nil), cv.next...)
		ref.full = true
		whole := gfx.NewImage(1280, 720)
		whole.Set(whole.Bounds(), th.Bg)
		ref.Draw(whole, ctx, r)
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				if live.At(x, y) != whole.At(x, y) {
					t.Fatalf("step %d: pixel %d,%d differs after partial redraws: %08x, whole %08x", step, x, y, live.At(x, y), whole.At(x, y))
				}
			}
		}
		now = now.Add(33 * time.Millisecond)
	}
	if glides == 0 {
		t.Fatal("no gliding frames were checked")
	}
	if cv.static < 10 {
		t.Errorf("the grid and its labels should be still: static %d", cv.static)
	}
}

// TestCanvasStill: a change in the still part redraws everything once and
// the still part is found again.
func TestCanvasStill(t *testing.T) {
	ctx := ui.NewContext(ui.Dark(ui.Liberation()), nil)
	r := image.Rect(0, 0, 300, 300)
	mk := func(label string, x float64) *PluginPage {
		return &PluginPage{Controls: []PluginControl{{ID: "c", Type: "canvas", Shapes: []PluginShape{
			{Shape: "ring", X: 50, Y: 50, R: 40, Color: "line"},
			{Shape: "text", X: 50, Y: 10, Text: label},
			{ID: "b", Shape: "circle", X: x, Y: 50, R: 3},
		}}}}
	}
	pg := mk("A", 20)
	now := time.Unix(1000, 0)
	cv := newCanvas("", func() (PluginControl, *PluginPage, bool) { return pg.Controls[0], pg, true })
	cv.now = func() time.Time { return now }
	cv.SetBounds(r)
	img := gfx.NewImage(300, 300)
	frame := func() []image.Rectangle {
		rects := cv.Dirty()
		for _, d := range rects {
			cv.Draw(img, ctx, d)
		}
		now = now.Add(100 * time.Millisecond)
		return rects
	}
	frame()
	pg = mk("A", 80)
	frame()
	if cv.static != 2 {
		t.Fatalf("ring and label should be still: static %d", cv.static)
	}
	pg = mk("A", 20)
	if rects := frame(); len(rects) == 0 || area(rects) > 150*150 {
		t.Fatalf("a moved blip should redraw a small part: %v", rects)
	}
	pg = mk("B", 20)
	if rects := frame(); len(rects) != 1 || rects[0] != r {
		t.Fatalf("a changed label should redraw everything: %v", rects)
	}
	pg = mk("B", 60)
	frame()
	if cv.static != 2 {
		t.Fatalf("the still part should be found again: static %d", cv.static)
	}
}

func area(rs []image.Rectangle) (a int) {
	for _, x := range rs {
		a += x.Dx() * x.Dy()
	}
	return a
}
