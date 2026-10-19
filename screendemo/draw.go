package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"strings"
	"sync"
)

// The page is drawn on a landscape canvas, as RØDE's screen app lays out its
// pages (1280 × 720); the panel itself is portrait (720 × 1280), so the
// canvas is turned when it goes to the display (see turn).
const (
	canvasW = 1280
	canvasH = 720
)

// Colours of rcplug's look (sdk/ui/rcplug.css).
var (
	colBg         = color.NRGBA{0x0b, 0x0e, 0x12, 0xff}
	colSurface    = color.NRGBA{0x14, 0x19, 0x20, 0xff}
	colBorder     = color.NRGBA{0x30, 0x3a, 0x46, 0xff}
	colText       = color.NRGBA{0xe6, 0xeb, 0xf0, 0xff}
	colText2      = color.NRGBA{0xa0, 0xac, 0xb8, 0xff}
	colText3      = color.NRGBA{0x6f, 0x7c, 0x89, 0xff}
	colAccent     = color.NRGBA{0x34, 0xd3, 0x99, 0xff}
	colAccentDown = color.NRGBA{0x1f, 0x9e, 0x72, 0xff}
	colInk        = color.NRGBA{0x04, 0x28, 0x1b, 0xff}
	colWarn       = color.NRGBA{0xf5, 0xb0, 0x41, 0xff}
)

// ---- text from the glyph atlases (font/, made by tools/fontatlas from the
// Go fonts)

//go:embed font/*.png font/faces.json
var fontFS embed.FS

type glyph struct{ X, Y, W, H, DX, DY, Adv int }

type face struct {
	Ascent, Descent, LineHeight int
	Glyphs                      map[string]glyph
	atlas                       *image.NRGBA
}

var (
	facesOnce sync.Once
	faces     map[string]*face
	facesErr  error
)

func loadFaces() (map[string]*face, error) {
	facesOnce.Do(func() {
		raw, err := fontFS.ReadFile("font/faces.json")
		if err != nil {
			facesErr = err
			return
		}
		m := map[string]*face{}
		if facesErr = json.Unmarshal(raw, &m); facesErr != nil {
			return
		}
		for name, f := range m {
			b, err := fontFS.ReadFile("font/" + name + ".png")
			if err != nil {
				facesErr = err
				return
			}
			img, err := png.Decode(bytes.NewReader(b))
			if err != nil {
				facesErr = err
				return
			}
			n := image.NewNRGBA(img.Bounds())
			draw.Draw(n, n.Bounds(), img, image.Point{}, draw.Src)
			f.atlas = n
		}
		faces = m
	})
	return faces, facesErr
}

// width is how wide s is in face f.
func (f *face) width(s string) int {
	w := 0
	for _, r := range s {
		if g, ok := f.Glyphs[string(r)]; ok {
			w += g.Adv
		} else if g, ok := f.Glyphs["?"]; ok {
			w += g.Adv
		}
	}
	return w
}

// text draws s with its baseline at y, starting at x, and returns where it
// ended.
func (f *face) text(dst *image.NRGBA, x, y int, s string, c color.NRGBA) int {
	for _, r := range s {
		g, ok := f.Glyphs[string(r)]
		if !ok {
			g = f.Glyphs["?"]
		}
		for gy := 0; gy < g.H; gy++ {
			for gx := 0; gx < g.W; gx++ {
				a := f.atlas.NRGBAAt(g.X+gx, g.Y+gy).A
				if a != 0 {
					blend(dst, x+g.DX+gx, y+g.DY+gy, c, a)
				}
			}
		}
		x += g.Adv
	}
	return x
}

// fit cuts s with … until it is at most w wide.
func (f *face) fit(s string, w int) string {
	if f.width(s) <= w {
		return s
	}
	rs := []rune(s)
	for len(rs) > 0 && f.width(string(rs)+"…") > w {
		rs = rs[:len(rs)-1]
	}
	return string(rs) + "…"
}

func blend(dst *image.NRGBA, x, y int, c color.NRGBA, a uint8) {
	if !(image.Point{x, y}.In(dst.Rect)) {
		return
	}
	i := dst.PixOffset(x, y)
	p := dst.Pix[i : i+4 : i+4]
	al := uint32(a) * uint32(c.A) / 255
	for k, v := range [3]uint8{c.R, c.G, c.B} {
		p[k] = uint8((uint32(v)*al + uint32(p[k])*(255-al)) / 255)
	}
	p[3] = 0xff
}

// ---- shapes

func fill(dst *image.NRGBA, r image.Rectangle, c color.NRGBA) {
	draw.Draw(dst, r, image.NewUniform(c), image.Point{}, draw.Src)
}

// roundRect fills r with corners of radius rad, and a border of 2 px in
// border if it is not the zero colour.
func roundRect(dst *image.NRGBA, r image.Rectangle, rad int, c, border color.NRGBA) {
	in := func(x, y int) bool { // inside the rounded shape?
		cx, cy := x, y
		switch {
		case x < r.Min.X+rad:
			cx = r.Min.X + rad
		case x >= r.Max.X-rad:
			cx = r.Max.X - rad - 1
		}
		switch {
		case y < r.Min.Y+rad:
			cy = r.Min.Y + rad
		case y >= r.Max.Y-rad:
			cy = r.Max.Y - rad - 1
		}
		dx, dy := x-cx, y-cy
		return dx*dx+dy*dy <= rad*rad
	}
	edge := max(rad, 2) // beyond it from every side, a point is inside, off the border
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if x >= r.Min.X+edge && x < r.Max.X-edge && y >= r.Min.Y+edge && y < r.Max.Y-edge {
				blend(dst, x, y, c, 0xff)
				continue
			}
			if !in(x, y) {
				continue
			}
			col := c
			if border.A != 0 && (!in(x-2, y) || !in(x+2, y) || !in(x, y-2) || !in(x, y+2)) {
				col = border
			}
			blend(dst, x, y, col, 0xff)
		}
	}
}

// ---- the page

// Info is what the page shows.
type Info struct {
	Rcplug   string // rcplug's version
	Device   string // the device's name
	Model    string
	Firmware string
	Mode     string // how the console reaches the mixer
	Leave    int    // seconds until the page gives the screen back
	Touches  []image.Point
	Pressed  bool // the finger is on the button
	German   bool
}

// backButton is where "Back to RØDE" is on the canvas.
var backButton = image.Rect(canvasW-470, canvasH-150, canvasW-70, canvasH-60)

// page draws the demo page on a fresh canvas.
func page(in Info) (*image.NRGBA, error) {
	fs, err := loadFaces()
	if err != nil {
		return nil, err
	}
	t := func(en, de string) string {
		if in.German {
			return de
		}
		return en
	}
	img := image.NewNRGBA(image.Rect(0, 0, canvasW, canvasH))
	fill(img, img.Rect, colBg)

	// The logo: a ring with three bars, as on rcplug's pages.
	cx, cy := 104, 112
	for y := cy - 34; y <= cy+34; y++ {
		for x := cx - 34; x <= cx+34; x++ {
			d := (x-cx)*(x-cx) + (y-cy)*(y-cy)
			if d <= 34*34 && d >= 29*29 {
				blend(img, x, y, colAccent, 0xff)
			}
		}
	}
	for i, h := range []int{18, 32, 24} {
		fill(img, image.Rect(cx-14+i*12, cy+14-h, cx-9+i*12, cy+14), colAccent)
	}
	fs["title"].text(img, 160, 134, "rcplug", colText)
	fs["text"].text(img, 160+fs["title"].width("rcplug")+20, 134, t("on the device's screen", "auf dem Bildschirm des Geräts"), colText2)

	// The facts, as a card.
	card := image.Rect(70, 200, canvasW-70, 470)
	roundRect(img, card, 22, colSurface, colBorder)
	rows := [][2]string{
		{t("rcplug", "rcplug"), in.Rcplug},
		{t("Device", "Gerät"), strings.TrimSpace(in.Device)},
		{t("Model and firmware", "Modell und Firmware"), strings.TrimSpace(in.Model + " " + in.Firmware)},
		{t("Console", "Konsole"), in.Mode},
	}
	y := card.Min.Y + 62
	for _, r := range rows {
		v := r[1]
		if v == "" {
			v = "–"
		}
		fs["text"].text(img, card.Min.X+40, y, r[0], colText2)
		fs["big"].text(img, card.Min.X+420, y+4, fs["big"].fit(v, card.Dx()-460), colText)
		y += 58
	}

	// What this is, and the way back.
	fs["text"].text(img, 70, canvasH-118, t("A demo of a plugin with its own page here.", "Eine Demo: ein Plugin mit eigener Seite hier."), colText)
	fs["small"].text(img, 70, canvasH-78, t("RØDE's screen comes back with the button,", "RØDEs Bildschirm kommt mit dem Knopf zurück,"), colText3)
	fs["small"].text(img, 70, canvasH-48, t(
		"a long press anywhere, or by itself in "+mins(in.Leave, false)+".",
		"mit langem Drücken irgendwo oder von selbst in "+mins(in.Leave, true)+"."), colText3)
	btn := colAccent
	if in.Pressed {
		btn = colAccentDown
	}
	roundRect(img, backButton, 20, btn, color.NRGBA{})
	label := t("Back to RØDE", "Zurück zu RØDE")
	lw := fs["big"].width(label)
	fs["big"].text(img, backButton.Min.X+(backButton.Dx()-lw)/2, backButton.Min.Y+backButton.Dy()/2+14, label, colInk)

	// Where the screen was touched: for checking the turn on the device.
	for _, p := range in.Touches {
		roundRect(img, image.Rect(p.X-14, p.Y-14, p.X+14, p.Y+14), 14, colWarn, color.NRGBA{})
	}
	return img, nil
}

func mins(secs int, german bool) string {
	m := (secs + 59) / 60
	switch {
	case secs < 60 && german:
		return "weniger als einer Minute"
	case secs < 60:
		return "less than a minute"
	case m == 1 && german:
		return "einer Minute"
	case m == 1:
		return "a minute"
	case german:
		return itoa(m) + " Minuten"
	}
	return itoa(m) + " minutes"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}

// ---- turning the canvas onto the portrait panel

// turn copies the landscape canvas into a portrait frame buffer (XRGB8888,
// pitch bytes a row), turned by deg (90: clockwise, 270: counter-clockwise).
func turn(src *image.NRGBA, dst []byte, pitch int, deg int) {
	w, h := src.Rect.Dx(), src.Rect.Dy() // 1280 × 720 → the panel is h × w
	for y := 0; y < h; y++ {
		row := src.Pix[y*src.Stride:]
		for x := 0; x < w; x++ {
			px, py := h-1-y, x // 90° clockwise
			if deg == 270 {
				px, py = y, w-1-x
			}
			o := py*pitch + px*4
			s := row[x*4:]
			dst[o], dst[o+1], dst[o+2], dst[o+3] = s[2], s[1], s[0], 0xff
		}
	}
}

// untouch maps a point on the portrait panel (as the touchscreen reports
// it) back onto the canvas.
func untouch(p image.Point, deg int) image.Point {
	if deg == 270 {
		return image.Pt(canvasW-1-p.Y, p.X)
	}
	return image.Pt(p.Y, canvasH-1-p.X)
}
