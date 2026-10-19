package ui

import (
	"image"
	"sync"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/svg"
)

// Font is a face at a size (from package font).
type Font = font.Font

// FontSet is the regular and bold face.
type FontSet struct {
	Regular, Bold *font.Face
}

// Liberation is the device's font (shipped with the plugin, OFL).
func Liberation() *FontSet { return &FontSet{Regular: font.Regular(), Bold: font.Bold()} }

// IconSource gives an icon's picture by its RØDE name (such as "IconHome").
type IconSource interface {
	// Icon returns the parsed picture, or nil if there is no such icon.
	Icon(name string) *svg.Doc
	// Raster returns a bitmap icon (a PNG resource), or nil.
	Raster(name string) *gfx.Image
}

// Context carries what views need to draw: the theme and the icons, with a
// cache of icons rendered at a size and colour.
type Context struct {
	Theme *Theme
	Icons IconSource

	mu    sync.Mutex
	cache map[iconKey]*gfx.Image
}

// NewContext makes a drawing context.
func NewContext(t *Theme, icons IconSource) *Context {
	return &Context{Theme: t, Icons: icons, cache: map[iconKey]*gfx.Image{}}
}

type iconKey struct {
	name   string
	w, h   int
	tint   gfx.Color
	tinted bool
}

// Icon draws the named icon to fit r. When tint has an alpha it is drawn in
// that one colour (for single-colour icons); otherwise it keeps its own
// colours. Missing icons draw nothing.
func (c *Context) Icon(dst *gfx.Image, name string, r image.Rectangle, tint gfx.Color) {
	if r.Empty() || c.Icons == nil {
		return
	}
	img := c.iconImage(name, r.Dx(), r.Dy(), tint)
	if img == nil {
		return
	}
	dst.Draw(img, r.Min.X+(r.Dx()-img.W)/2, r.Min.Y+(r.Dy()-img.H)/2, 255)
}

func (c *Context) iconImage(name string, w, h int, tint gfx.Color) *gfx.Image {
	tinted := tint.A() != 0
	key := iconKey{name, w, h, tint, tinted}
	c.mu.Lock()
	img, ok := c.cache[key]
	c.mu.Unlock()
	if ok {
		return img
	}
	img = c.renderIcon(name, w, h, tint, tinted)
	c.mu.Lock()
	c.cache[key] = img
	c.mu.Unlock()
	return img
}

func (c *Context) renderIcon(name string, w, h int, tint gfx.Color, tinted bool) *gfx.Image {
	if r := c.Icons.Raster(name); r != nil {
		fw, fh := fit(r.W, r.H, w, h)
		return r.Scaled(fw, fh)
	}
	doc := c.Icons.Icon(name)
	if doc == nil {
		return nil
	}
	fw, fh := fit(int(doc.Width), int(doc.Height), w, h)
	if fw <= 0 || fh <= 0 {
		return nil
	}
	img := doc.Render(fw, fh)
	if tinted {
		return tintMask(img, tint)
	}
	return img
}

// tintMask recolours an icon to one colour, keeping its coverage.
func tintMask(m *gfx.Image, c gfx.Color) *gfx.Image {
	out := gfx.NewImage(m.W, m.H)
	base := c.Premul()
	for i, p := range m.Pix {
		if a := p >> 24; a != 0 {
			out.Pix[i] = scalePremul(base, a)
		}
	}
	return out
}

func scalePremul(p, a uint32) uint32 {
	var out uint32
	for sh := 0; sh < 32; sh += 8 {
		out |= ((p>>sh&0xff)*a + 127) / 255 << sh
	}
	return out
}

// fit scales w × h into box, keeping the aspect ratio.
func fit(w, h, bw, bh int) (int, int) {
	if w <= 0 || h <= 0 {
		return bw, bh
	}
	if w*bh <= bw*h {
		return max(1, w*bh/h), bh
	}
	return bw, max(1, h*bw/w)
}

// T is a shortcut for the theme.
func (c *Context) T() *Theme { return c.Theme }
