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

// Colours of rcplug's look (sdk/ui/rcplug.css), and the rain's green.
var (
	colBg         = color.NRGBA{0x0b, 0x0e, 0x12, 0xff}
	colBorder     = color.NRGBA{0x30, 0x3a, 0x46, 0xff}
	colText       = color.NRGBA{0xe6, 0xeb, 0xf0, 0xff}
	colText2      = color.NRGBA{0xa0, 0xac, 0xb8, 0xff}
	colText3      = color.NRGBA{0x6f, 0x7c, 0x89, 0xff}
	colAccent     = color.NRGBA{0x34, 0xd3, 0x99, 0xff}
	colAccentDown = color.NRGBA{0x1f, 0x9e, 0x72, 0xff}
	colInk        = color.NRGBA{0x04, 0x28, 0x1b, 0xff}
	colTerm       = color.NRGBA{0x02, 0x06, 0x04, 0xff}
	colBarBg      = color.NRGBA{0x1c, 0x22, 0x2a, 0xff}
	colGreen      = color.NRGBA{0x3c, 0xff, 0x7a, 0xff}
	colGreenDim   = color.NRGBA{0x9c, 0xd8, 0xae, 0xff}
)

// Where things are on the canvas.
var (
	// rainArea is where the rain falls (inside its frame), redrawn every
	// frame.
	rainArea = image.Rect(72, 202, 528, 608)
	// termArea is the terminal with the proof of root.
	termArea = image.Rect(560, 200, 1210, 528)
	// backButton is where "Back to RØDE" is.
	backButton = image.Rect(canvasW-430, canvasH-110, canvasW-70, canvasH-30)
)

// The terminal: a title bar, then lines of the mono face.
const (
	termBar  = 38
	termPad  = 22
	termLine = 29
)

// TermLine is a line of the terminal: a command after the prompt, or what
// it printed.
type TermLine struct {
	Cmd  bool
	Text string
}

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

func (f *face) glyph(r rune) glyph {
	if g, ok := f.Glyphs[string(r)]; ok {
		return g
	}
	return f.Glyphs["?"]
}

// width is how wide s is in face f.
func (f *face) width(s string) int {
	w := 0
	for _, r := range s {
		w += f.glyph(r).Adv
	}
	return w
}

// text draws s with its baseline at y, starting at x, and returns where it
// ended.
func (f *face) text(dst *image.NRGBA, x, y int, s string, c color.NRGBA) int {
	for _, r := range s {
		g := f.glyph(r)
		f.draw(dst, g, x+g.DX, y+g.DY, c, false)
		x += g.Adv
	}
	return x
}

// draw puts glyph g with its box's top left at x, y (mirrored: flipped left
// to right), in c (its alpha scales the glyph's).
func (f *face) draw(dst *image.NRGBA, g glyph, x, y int, c color.NRGBA, mirrored bool) {
	for gy := 0; gy < g.H; gy++ {
		dy := y + gy
		if dy < dst.Rect.Min.Y || dy >= dst.Rect.Max.Y {
			continue
		}
		row := f.atlas.Pix[f.atlas.PixOffset(g.X, g.Y+gy):]
		for gx := 0; gx < g.W; gx++ {
			a := row[gx*4+3]
			if a == 0 {
				continue
			}
			dx := x + gx
			if mirrored {
				dx = x + g.W - 1 - gx
			}
			blend(dst, dx, dy, c, a)
		}
	}
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
	p[0] = uint8((uint32(c.R)*al + uint32(p[0])*(255-al)) / 255)
	p[1] = uint8((uint32(c.G)*al + uint32(p[1])*(255-al)) / 255)
	p[2] = uint8((uint32(c.B)*al + uint32(p[2])*(255-al)) / 255)
	p[3] = 0xff
}

// ---- shapes

// fill sets every pixel of r to c (draw.Draw has no fast way for NRGBA).
func fill(dst *image.NRGBA, r image.Rectangle, c color.NRGBA) {
	r = r.Intersect(dst.Rect)
	if r.Empty() {
		return
	}
	w := r.Dx() * 4
	first := dst.Pix[dst.PixOffset(r.Min.X, r.Min.Y):][:w]
	for i := 0; i < w; i += 4 {
		first[i], first[i+1], first[i+2], first[i+3] = c.R, c.G, c.B, c.A
	}
	for y := r.Min.Y + 1; y < r.Max.Y; y++ {
		copy(dst.Pix[dst.PixOffset(r.Min.X, y):][:w], first)
	}
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
	Rcplug   string     // rcplug's version
	Model    string     // the device's model
	Firmware string     // its firmware
	Mode     string     // how the console holds the mixer
	Title    string     // the terminal's title: user@host
	Prompt   string     // "# " for root, "$ " otherwise
	Proof    []TermLine // the terminal: commands and what they printed
	Leave    int        // seconds until the page gives the screen back
	Pressed  bool       // the finger is on the button
	German   bool
}

// page draws the page without what moves (the rain, the cursor) on a fresh
// canvas.
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

	// The logo (a ring with three bars, as on rcplug's pages), the name and
	// the claim.
	cx, cy := 104, 96
	for y := cy - 34; y <= cy+34; y++ {
		for x := cx - 34; x <= cx+34; x++ {
			d := (x-cx)*(x-cx) + (y-cy)*(y-cy)
			if d <= 34*34 && d >= 29*29 {
				blend(img, x, y, colGreen, 0xff)
			}
		}
	}
	for i, h := range []int{18, 32, 24} {
		fill(img, image.Rect(cx-14+i*12, cy+14-h, cx-9+i*12, cy+14), colGreen)
	}
	x := fs["title"].text(img, 160, 118, "PlugRain", colText)
	fs["text"].text(img, x+24, 118, fs["text"].fit("World's first jailbroken RØDECaster Duo", canvasW-70-x-24), colGreen)
	var what []string
	for _, s := range []string{in.Model, firmwareText(in.Firmware, in.German), "rcplug " + in.Rcplug} {
		if strings.TrimSpace(s) != "" && s != "rcplug " {
			what = append(what, s)
		}
	}
	fs["small"].text(img, 162, 160, fs["small"].fit(strings.Join(what, " · "), canvasW-232), colText2)

	// The rain's frame; the rain itself is drawn every frame (rain.go).
	roundRect(img, rainArea.Inset(-3), 6, color.NRGBA{0, 0, 0, 0xff}, colBorder)

	// The terminal with the proof of root: a window with its title bar.
	roundRect(img, termArea, 14, colTerm, colBorder)
	bar := image.Rect(termArea.Min.X+2, termArea.Min.Y+2, termArea.Max.X-2, termArea.Min.Y+termBar)
	roundRect(img, bar, 12, colBarBg, color.NRGBA{})
	fill(img, image.Rect(bar.Min.X, bar.Max.Y-12, bar.Max.X, bar.Max.Y), colBarBg)
	for i, c := range []color.NRGBA{{0xff, 0x5f, 0x57, 0xff}, {0xfe, 0xbc, 0x2e, 0xff}, {0x28, 0xc8, 0x40, 0xff}} {
		roundRect(img, image.Rect(bar.Min.X+16+i*22, bar.Min.Y+11, bar.Min.X+30+i*22, bar.Min.Y+25), 7, c, color.NRGBA{})
	}
	m := fs["mono"]
	tw := m.width(in.Title)
	m.text(img, bar.Min.X+(bar.Dx()-tw)/2, bar.Min.Y+25, in.Title, colText2)
	y := termArea.Min.Y + termBar + termPad + m.Ascent - 6
	for _, l := range in.Proof {
		if l.Cmd {
			x := m.text(img, termArea.Min.X+termPad, y, in.Prompt, colGreen)
			m.text(img, x, y, m.fit(l.Text, termArea.Max.X-termPad-x), colText)
		} else {
			m.text(img, termArea.Min.X+termPad, y, m.fit(l.Text, termArea.Dx()-2*termPad), colGreenDim)
		}
		y += termLine
	}
	m.text(img, termArea.Min.X+termPad, y, in.Prompt, colGreen) // the cursor follows (cursorRect)
	mode := in.Mode
	if mode == "" {
		mode = "–"
	}
	fs["small"].text(img, termArea.Min.X+4, termArea.Max.Y+44, fs["small"].fit(t("Console: ", "Konsole: ")+mode, termArea.Dx()-8), colText2)

	// The way back.
	fs["small"].text(img, 72, canvasH-58, t("RØDE's screen comes back with the button,", "RØDEs Bildschirm kommt mit dem Knopf zurück,"), colText3)
	fs["small"].text(img, 72, canvasH-26, t(
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
	return img, nil
}

func firmwareText(fw string, german bool) string {
	switch {
	case fw == "":
		return ""
	case german:
		return "Firmware " + fw
	}
	return "firmware " + fw
}

// cursorRect is where the terminal's cursor blinks: after the prompt below
// the last line.
func cursorRect(in Info) image.Rectangle {
	fs, err := loadFaces()
	if err != nil {
		return image.Rectangle{}
	}
	m := fs["mono"]
	y := termArea.Min.Y + termBar + termPad + m.Ascent - 6 + len(in.Proof)*termLine
	x := termArea.Min.X + termPad + m.width(in.Prompt)
	return image.Rect(x, y-m.Ascent+3, x+m.glyph('M').Adv, y+m.Descent-1)
}

// drawCursor paints the cursor, on or off.
func drawCursor(dst *image.NRGBA, r image.Rectangle, on bool) {
	if on {
		fill(dst, r, colGreen)
	} else {
		fill(dst, r, colTerm)
	}
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

// turn copies the landscape canvas into a portrait frame (XRGB8888, pitch
// bytes a row), turned by deg (90: clockwise, 270: counter-clockwise).
func turn(src *image.NRGBA, dst []byte, pitch int, deg int) {
	turnRect(src, src.Rect, dst, pitch, deg)
}

// turnRect copies the part r of the canvas into the portrait frame.
func turnRect(src *image.NRGBA, r image.Rectangle, dst []byte, pitch int, deg int) {
	r = r.Intersect(src.Rect).Intersect(image.Rect(0, 0, canvasW, canvasH))
	for y := r.Min.Y; y < r.Max.Y; y++ {
		row := src.Pix[src.PixOffset(r.Min.X, y):]
		for x := r.Min.X; x < r.Max.X; x++ {
			px, py := canvasH-1-y, x // 90° clockwise
			if deg == 270 {
				px, py = y, canvasW-1-x
			}
			o := py*pitch + px*4
			s := row[(x-r.Min.X)*4:]
			dst[o], dst[o+1], dst[o+2], dst[o+3] = s[2], s[1], s[0], 0xff
		}
	}
}

// panelRect is where the part r of the canvas lies on the portrait panel.
func panelRect(r image.Rectangle, deg int) image.Rectangle {
	if deg == 270 {
		return image.Rect(r.Min.Y, canvasW-r.Max.X, r.Max.Y, canvasW-r.Min.X)
	}
	return image.Rect(canvasH-r.Max.Y, r.Min.X, canvasH-r.Min.Y, r.Max.X)
}

// untouch maps a point on the portrait panel (as the touchscreen reports
// it) back onto the canvas.
func untouch(p image.Point, deg int) image.Point {
	if deg == 270 {
		return image.Pt(canvasW-1-p.Y, p.X)
	}
	return image.Pt(p.Y, canvasH-1-p.X)
}
