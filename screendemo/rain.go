package main

import (
	"image"
	"image/color"
	"math/rand"
	"time"
)

// The rain: columns of glyphs falling at their own speed, the leading one
// bright, the trail fading, now and then a glyph changing. It moves by time,
// not by frame, so it falls as fast at 20 frames per second as at 60; it is
// there to show how smoothly the display runs.

// rainGlyphs are what falls (mirrored, as in the film).
const rainGlyphs = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ:.=*+-<>|Ø#$%&@×"

type drop struct {
	x      int     // left edge, in the area
	head   float64 // where the leading glyph is, px from the area's top
	speed  float64 // px per second
	glyphs []rune  // the leading one first
}

type rain struct {
	area  image.Rectangle
	cell  int // a glyph's height and the space between columns
	col   int
	drops []drop
	rng   *rand.Rand
	face  *face
}

func newRain(area image.Rectangle, seed int64) (*rain, error) {
	fs, err := loadFaces()
	if err != nil {
		return nil, err
	}
	f := fs["rain"]
	r := &rain{area: area, face: f, cell: f.LineHeight - 2, col: f.glyph('M').Adv + 4, rng: rand.New(rand.NewSource(seed))}
	for x := 2; x+r.col <= area.Dx(); x += r.col {
		d := drop{x: x}
		r.reset(&d)
		d.head = r.rng.Float64() * float64(area.Dy()+len(d.glyphs)*r.cell) // already falling
		r.drops = append(r.drops, d)
	}
	return r, nil
}

// reset starts a drop again above the area.
func (r *rain) reset(d *drop) {
	d.speed = 90 + r.rng.Float64()*260
	d.glyphs = make([]rune, 6+r.rng.Intn(16))
	for i := range d.glyphs {
		d.glyphs[i] = r.randGlyph()
	}
	d.head = -r.rng.Float64() * float64(r.area.Dy()) / 2
}

func (r *rain) randGlyph() rune {
	gs := []rune(rainGlyphs)
	return gs[r.rng.Intn(len(gs))]
}

// step moves the rain on by dt.
func (r *rain) step(dt time.Duration) {
	s := dt.Seconds()
	if s > 0.25 {
		s = 0.25 // after a pause, no jump
	}
	for i := range r.drops {
		d := &r.drops[i]
		d.head += d.speed * s
		if d.head-float64(len(d.glyphs)*r.cell) > float64(r.area.Dy()) {
			r.reset(d)
		}
		// Glyphs change now and then, about three a second per drop.
		if r.rng.Float64() < 3*s {
			d.glyphs[r.rng.Intn(len(d.glyphs))] = r.randGlyph()
		}
	}
}

// draw paints the rain over its area of dst (in canvas coordinates).
func (r *rain) draw(dst *image.NRGBA) {
	fill(dst, r.area, color.NRGBA{0, 0, 0, 0xff})
	clip := dst.SubImage(r.area).(*image.NRGBA)
	for _, d := range r.drops {
		n := len(d.glyphs)
		for k, g := range d.glyphs {
			y := int(d.head) - k*r.cell // the glyph's baseline
			if y < 0 || y-r.cell > r.area.Dy() {
				continue
			}
			c := colGreen
			switch {
			case k == 0:
				c = color.NRGBA{0xd8, 0xff, 0xe4, 0xff}
			default:
				f := 1 - float64(k)/float64(n)
				c.A = uint8(40 + 215*f*f)
			}
			gl := r.face.glyph(g)
			r.face.draw(clip, gl, r.area.Min.X+d.x+gl.DX, r.area.Min.Y+y+gl.DY, c, true)
		}
	}
}
