// Command fontatlas renders the Go fonts into glyph atlases (PNG, alpha in
// the image's alpha channel) and their metrics (JSON) for plugins that draw
// text themselves, so the plugins stay with the standard library:
//
//	cd tools/fontatlas && go run . -out ../../plugrain/font
//
// It is its own module: golang.org/x/image is needed here only.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// The characters the atlases hold: printable ASCII, German and some signs.
const chars = " !\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~ÄÖÜäöüßØø°·–—…→←€„“”‘’×"

type face struct {
	Name string
	TTF  []byte
	Size float64
}

// Glyph is one character in an atlas: its box there, how it sits on the
// baseline (X, Y from the pen position to the box's top left) and how far
// the pen moves after it.
type Glyph struct {
	X, Y, W, H  int
	DX, DY, Adv int
}

type Face struct {
	Size       float64          `json:"size"`
	Ascent     int              `json:"ascent"`
	Descent    int              `json:"descent"`
	LineHeight int              `json:"lineHeight"`
	Glyphs     map[string]Glyph `json:"glyphs"`
}

func main() {
	out := flag.String("out", "font", "output folder")
	flag.Parse()
	faces := []face{
		{"small", goregular.TTF, 24}, {"text", goregular.TTF, 30},
		{"big", gobold.TTF, 40}, {"title", gobold.TTF, 60},
		{"mono", gomono.TTF, 22}, {"rain", gomonobold.TTF, 24},
	}
	all := map[string]Face{}
	for _, f := range faces {
		atlas, meta, err := render(f)
		if err != nil {
			log.Fatal(err)
		}
		w, err := os.Create(filepath.Join(*out, f.Name+".png"))
		if err != nil {
			log.Fatal(err)
		}
		if err := png.Encode(w, atlas); err != nil {
			log.Fatal(err)
		}
		w.Close()
		all[f.Name] = meta
	}
	b, _ := json.Marshal(all)
	if err := os.WriteFile(filepath.Join(*out, "faces.json"), append(b, '\n'), 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Println("wrote", len(faces), "atlases to", *out)
}

func render(f face) (*image.NRGBA, Face, error) {
	ft, err := opentype.Parse(f.TTF)
	if err != nil {
		return nil, Face{}, err
	}
	fc, err := opentype.NewFace(ft, &opentype.FaceOptions{Size: f.Size, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return nil, Face{}, err
	}
	defer fc.Close()
	m := fc.Metrics()
	meta := Face{Size: f.Size, Ascent: m.Ascent.Ceil(), Descent: m.Descent.Ceil(), LineHeight: m.Height.Ceil(), Glyphs: map[string]Glyph{}}
	const width = 1024
	cell := meta.Ascent + meta.Descent + 4
	// Lay the glyphs out in rows, then draw them.
	type place struct {
		r   rune
		box image.Rectangle
		at  image.Point
		adv int
	}
	var places []place
	x, y := 2, 2
	for _, r := range chars {
		b, adv, ok := fc.GlyphBounds(r)
		if !ok {
			return nil, Face{}, fmt.Errorf("%s: no glyph for %q", f.Name, r)
		}
		box := image.Rect(b.Min.X.Floor(), b.Min.Y.Floor(), b.Max.X.Ceil(), b.Max.Y.Ceil())
		w := box.Dx() + 2
		if x+w > width {
			x, y = 2, y+cell
		}
		places = append(places, place{r, box, image.Pt(x, y), adv.Round()})
		x += w + 2
	}
	atlas := image.NewNRGBA(image.Rect(0, 0, width, y+cell+2))
	d := &font.Drawer{Dst: atlas, Src: image.White, Face: fc}
	for _, p := range places {
		// The pen sits so that the glyph's box lands at p.at.
		d.Dot = fixed.P(p.at.X-p.box.Min.X, p.at.Y-p.box.Min.Y)
		d.DrawString(string(p.r))
		meta.Glyphs[string(p.r)] = Glyph{X: p.at.X, Y: p.at.Y, W: p.box.Dx(), H: p.box.Dy(), DX: p.box.Min.X, DY: p.box.Min.Y, Adv: p.adv}
	}
	// Only the alpha matters; keep the colour white.
	out := image.NewNRGBA(atlas.Bounds())
	draw.Draw(out, out.Bounds(), atlas, image.Point{}, draw.Src)
	return out, meta, nil
}
