package pages

import (
	"image"
	"math"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// micCarousel is the microphone choice at the top of a combo input's card,
// as on RØDE's screen: a row of tiles with the model's picture and name, the
// chosen one in the middle on the channel's colour, the neighbours dimmed
// at the sides. It slides with a swipe; a tap on a neighbour chooses it, a
// tap on the middle opens the channel assignment (as the banner does).
type micCarousel struct {
	ui.Base
	env   *Env
	strip int
	input int

	pos    float64 // the tile at the middle, in tile units (fractional while sliding)
	target float64
	drag   bool
	moved  bool
	startX int
	startP float64
	last   string
}

// Tile geometry: the middle tile is 224 wide (RØDE's), the next starts 10
// further on.
const (
	micTileW   = 224
	micTileGap = 10
	micPitch   = micTileW + micTileGap
)

// micPictureSize is the size the carousel draws a picture at (the tile
// less its margins and the name).
const micPictureW, micPictureH = micTileW - 24, 250 - 20 - 8 - 56

// WarmMicPictures scales the microphone pictures to the carousel's size
// ahead of the first channel page, so it opens without that work.
func WarmMicPictures(c *ui.Context) {
	for _, m := range desk.Mics {
		c.Warm(m.Picture, micPictureW, micPictureH)
	}
}

func newMicCarousel(env *Env, strip, input int) *micCarousel {
	m := &micCarousel{env: env, strip: strip, input: input}
	m.target = float64(m.selected())
	m.pos = m.target
	return m
}

// selected is the index in desk.Mics of the input's microphone (0 if the
// value is not offered).
func (m *micCarousel) selected() int {
	if i := desk.MicIndex(m.env.Desk.MicType(m.input)); i >= 0 {
		return i
	}
	return 0
}

func (m *micCarousel) Dirty() []image.Rectangle {
	if !m.drag {
		m.target = float64(m.selected())
		if d := m.target - m.pos; math.Abs(d) > 0.005 {
			m.pos += d * 0.3 // ease to the chosen tile
			m.MarkDirty()
		} else if m.pos != m.target {
			m.pos = m.target
			m.MarkDirty()
		}
	}
	if s := intText(int64(m.pos*1000)) + m.env.Desk.InputColour(m.strip); s != m.last {
		m.last = s
		m.MarkDirty()
	}
	return m.Base.Dirty()
}

func (m *micCarousel) Touch(phase ui.TouchPhase, p image.Point) bool {
	switch phase {
	case ui.Down:
		m.drag, m.moved, m.startX, m.startP = true, false, p.X, m.pos
	case ui.Move:
		if !m.drag {
			return true
		}
		dx := p.X - m.startX
		if !m.moved && dx > -12 && dx < 12 {
			return true
		}
		m.moved = true
		m.pos = clampF(m.startP-float64(dx)/micPitch, -0.3, float64(len(desk.Mics)-1)+0.3)
		m.MarkDirty()
	case ui.Up:
		if !m.drag {
			return true
		}
		m.drag = false
		if m.moved {
			// Snap to the nearest tile and choose it.
			i := int(math.Round(clampF(m.pos, 0, float64(len(desk.Mics)-1))))
			m.choose(i)
			return true
		}
		// A tap: which tile is under the finger?
		r := m.Bounds()
		mid := (r.Min.X + r.Max.X) / 2
		off := int(math.Round(float64(p.X-mid) / micPitch))
		switch {
		case off == 0:
			m.env.Nav.Push(NewAssign(m.strip))
		default:
			m.choose(m.selected() + off)
		}
	}
	return true
}

// choose picks the i-th entry of desk.Mics (clamped) and writes it to the
// desk; the desk sets the model's gain, phantom power and voicing.
func (m *micCarousel) choose(i int) {
	if i < 0 {
		i = 0
	}
	if i >= len(desk.Mics) {
		i = len(desk.Mics) - 1
	}
	m.env.Desk.SetMicType(m.input, desk.Mics[i].Type)
	m.target = float64(i)
	m.MarkDirty()
}

func (m *micCarousel) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	r := m.Bounds()
	clip = clip.Intersect(r)
	if clip.Empty() {
		return
	}
	sel := m.selected()
	col := stripColour(m.env, m.strip)
	midX := (r.Min.X + r.Max.X) / 2
	top, bottom := r.Min.Y+10, r.Max.Y-10
	// The tiles within reach of the view: two either side of the middle.
	from, to := int(math.Floor(m.pos))-2, int(math.Ceil(m.pos))+2
	for i := from; i <= to; i++ {
		if i < 0 || i >= len(desk.Mics) {
			continue
		}
		x := midX + int(math.Round((float64(i)-m.pos)*micPitch)) - micTileW/2
		tile := image.Rect(x, top, x+micTileW, bottom)
		if !tile.Overlaps(clip) {
			continue
		}
		mic := desk.Mics[i]
		fill, fg, alpha := gfx.Hex(0x1a1a1a), th.TextMuted, uint8(150)
		if i == sel {
			fill, fg, alpha = col, th.Text, 255
		}
		ui.Panel(dst, tile, 10, fill, clip)
		pic := image.Rect(tile.Min.X+12, tile.Min.Y+8, tile.Max.X-12, tile.Max.Y-56)
		c.IconIn(dst, mic.Picture, pic, 0, alpha, clip)
		ui.Text(dst, th.Fonts.Body, mic.Name, image.Rect(tile.Min.X, tile.Max.Y-56, tile.Max.X, tile.Max.Y-8), font.Center, fg, clip)
	}
}

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
