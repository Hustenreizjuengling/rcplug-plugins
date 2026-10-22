package pages

import (
	"encoding/json"
	"fmt"
	"image"
	"math"
	"testing"
	"time"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// radarShapes is a picture like Sonar's: grid, labels, trail, beam, glow,
// blip, readouts and two bars.
func radarShapes(az float64, text bool) []PluginShape {
	const cx, cy, R = 50.0, 56.0, 44.0
	var sh []PluginShape
	for k := 1; k <= 4; k++ {
		sh = append(sh, PluginShape{Shape: "arc", X: cx, Y: cy, R: R * float64(k) / 4, A0: 180, A1: 360, Color: "line", Alpha: 0.6})
	}
	for _, d := range []float64{-90, -45, 0, 45, 90} {
		a := (270 + d) * math.Pi / 180
		sh = append(sh, PluginShape{Shape: "line", Points: [][2]float64{{cx, cy}, {cx + R*math.Cos(a), cy + R*math.Sin(a)}}, Color: "line", Alpha: 0.6})
		if text {
			sh = append(sh, PluginShape{Shape: "text", X: cx + (R+5)*math.Cos(a), Y: cy + (R+5)*math.Sin(a), Text: fmt.Sprintf("%+d°", int(d)), Size: "tiny", Color: "dim"})
		}
	}
	for i := 0; i < 24; i++ {
		a := (270 + az + float64(i-12)*2) * math.Pi / 180
		r := R * (0.12 + 0.83*0.5)
		sh = append(sh, PluginShape{Shape: "circle", X: cx + r*math.Cos(a), Y: cy + r*math.Sin(a), R: 0.7, Alpha: 0.3})
	}
	a := (270 + az) * math.Pi / 180
	sh = append(sh,
		PluginShape{ID: "beam", Shape: "line", Points: [][2]float64{{cx, cy}, {cx + R*math.Cos(a), cy + R*math.Sin(a)}}, Width: 0.5, Alpha: 0.6},
		PluginShape{ID: "glow", Shape: "glow", X: cx + 22*math.Cos(a), Y: cy + 22*math.Sin(a), R: 5, Alpha: 0.8},
		PluginShape{ID: "blip", Shape: "circle", X: cx + 22*math.Cos(a), Y: cy + 22*math.Sin(a), R: 0.9, Color: "text"},
	)
	if text {
		for i := 0; i < 4; i++ {
			sh = append(sh,
				PluginShape{Shape: "text", X: 12.5 + 25*float64(i), Y: 72, Text: "-12°", Size: "title"},
				PluginShape{Shape: "text", X: 12.5 + 25*float64(i), Y: 80, Text: "Richtung", Size: "tiny", Color: "dim"})
		}
	}
	sh = append(sh,
		PluginShape{Shape: "rect", X: 10, Y: 89, W: 80, H: 2, Radius: 1, Color: "panel"},
		PluginShape{ID: "l", Shape: "rect", X: 10, Y: 89, W: 40, H: 2, Radius: 1},
		PluginShape{Shape: "rect", X: 10, Y: 94, W: 80, H: 2, Radius: 1, Color: "panel"},
		PluginShape{ID: "r", Shape: "rect", X: 10, Y: 94, W: 38, H: 2, Radius: 1},
	)
	return sh
}

func benchCanvas(b *testing.B, text bool, size int) {
	ctx := ui.NewContext(ui.Dark(ui.Liberation()), nil)
	pages := []*PluginPage{
		{Controls: []PluginControl{{ID: "c", Type: "canvas", W: 100, H: 100, Shapes: radarShapes(-20, text)}}},
		{Controls: []PluginControl{{ID: "c", Type: "canvas", W: 100, H: 100, Shapes: radarShapes(20, text)}}},
	}
	i := 0
	cv := newCanvas("", func() (PluginControl, *PluginPage, bool) { return pages[i%2].Controls[0], pages[i%2], true })
	r := image.Rect(0, 0, size, size)
	cv.SetBounds(r)
	img := gfx.NewImage(1280, 720)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		i++
		cv.Draw(img, ctx, r)
	}
}

func BenchmarkCanvasRadar(b *testing.B)       { benchCanvas(b, true, 592) }
func BenchmarkCanvasRadarNoText(b *testing.B) { benchCanvas(b, false, 592) }

// BenchmarkCanvasFrames is a run of frames as the screen draws them: a new
// description every third frame, the frames between gliding.
func BenchmarkCanvasFrames(b *testing.B) {
	ctx := ui.NewContext(ui.Dark(ui.Liberation()), nil)
	var pg *PluginPage
	describe := func(az float64) {
		pg = &PluginPage{Controls: []PluginControl{{ID: "c", Type: "canvas", W: 100, H: 100, Shapes: radarShapes(az, true)}}}
	}
	describe(0)
	now := time.Unix(1000, 0)
	cv := newCanvas("Radar", func() (PluginControl, *PluginPage, bool) { return pg.Controls[0], pg, true })
	cv.now = func() time.Time { return now }
	r := image.Rect(40, 64, 632, 656)
	cv.SetBounds(r)
	img := gfx.NewImage(1280, 720)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if n%3 == 0 {
			describe(math.Sin(float64(n)/20) * 60)
		}
		for _, d := range cv.Dirty() {
			img.Set(d, ctx.Theme.Bg)
			cv.Draw(img, ctx, d)
		}
		now = now.Add(33 * time.Millisecond)
	}
}

func BenchmarkCanvasJSON(b *testing.B) {
	pg := PluginPage{Title: "Sonar", Every: 100, Controls: []PluginControl{{ID: "c", Type: "canvas", W: 100, H: 100, Shapes: radarShapes(10, true)}}}
	data, _ := json.Marshal(pg)
	b.ReportMetric(float64(len(data)), "bytes")
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		var out PluginPage
		_ = json.Unmarshal(data, &out)
	}
}
