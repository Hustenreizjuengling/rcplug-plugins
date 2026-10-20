package preview

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sync"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/rode"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/svg"
)

// DirIcons serves the device's icons from a folder of extracted resource
// files (name.svg / name.png), for the browser preview on a development
// machine. The folder is provided at runtime and never shipped.
type DirIcons struct {
	dir  string
	mu   sync.Mutex
	svgs map[string]*svg.Doc
	pngs map[string]*gfx.Image
	none map[string]bool
}

// NewDirIcons reads icons from dir.
func NewDirIcons(dir string) *DirIcons {
	return &DirIcons{dir: dir, svgs: map[string]*svg.Doc{}, pngs: map[string]*gfx.Image{}, none: map[string]bool{}}
}

// Icon returns the named icon parsed from dir/name.svg.
func (d *DirIcons) Icon(name string) *svg.Doc {
	d.mu.Lock()
	defer d.mu.Unlock()
	if doc, ok := d.svgs[name]; ok {
		return doc
	}
	if d.none[name] {
		return nil
	}
	b, err := os.ReadFile(filepath.Join(d.dir, name+".svg"))
	if err != nil {
		d.none[name] = true
		return nil
	}
	doc, err := svg.Parse(b)
	if err != nil {
		d.none[name] = true
		return nil
	}
	d.svgs[name] = doc
	return doc
}

// Raster returns the named icon decoded from dir/name.png.
func (d *DirIcons) Raster(name string) *gfx.Image {
	d.mu.Lock()
	defer d.mu.Unlock()
	if img, ok := d.pngs[name]; ok {
		return img
	}
	f, err := os.Open(filepath.Join(d.dir, name+".png"))
	if err != nil {
		return nil
	}
	defer f.Close()
	src, err := png.Decode(f)
	if err != nil {
		return nil
	}
	d.pngs[name] = fromImage(src)
	return d.pngs[name]
}

// fromImage converts a decoded image to a premultiplied gfx.Image.
func fromImage(src image.Image) *gfx.Image {
	b := src.Bounds()
	out := gfx.NewImage(b.Dx(), b.Dy())
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			r, g, bl, a := src.At(b.Min.X+x, b.Min.Y+y).RGBA() // 16-bit premultiplied
			out.Pix[y*out.Stride+x] = uint32(a>>8)<<24 | uint32(r>>8)<<16 | uint32(g>>8)<<8 | uint32(bl>>8)
		}
	}
	return out
}

// LoadTexts reads a language file (e.g. german.txt) from dir and returns its
// key→text table. An empty map is returned when it is missing.
func LoadTexts(dir, file string) map[string]string {
	b, err := os.ReadFile(filepath.Join(dir, file))
	if err != nil {
		return map[string]string{}
	}
	return rode.ParseTranslations(b)
}
