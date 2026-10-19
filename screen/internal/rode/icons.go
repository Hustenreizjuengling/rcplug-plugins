package rode

import (
	"sync"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/svg"
)

// Icons gives the device's icons as parsed pictures, keeping each one after
// the first use. It satisfies the ui.IconSource interface.
type Icons struct {
	res *Resources

	mu    sync.Mutex
	svgs  map[string]*svg.Doc
	pngs  map[string]*gfx.Image
	tried map[string]bool
}

// NewIcons reads icons from the resources.
func NewIcons(res *Resources) *Icons {
	return &Icons{res: res, svgs: map[string]*svg.Doc{}, pngs: map[string]*gfx.Image{}, tried: map[string]bool{}}
}

// Icon returns the named icon as a parsed SVG (nil if none, or if it is a
// bitmap).
func (ic *Icons) Icon(name string) *svg.Doc {
	ic.mu.Lock()
	defer ic.mu.Unlock()
	if d, ok := ic.svgs[name]; ok {
		return d
	}
	if ic.tried[name] {
		return nil
	}
	ic.tried[name] = true
	if ic.res.Kind(name) != "svg" {
		return nil
	}
	b, ok := ic.res.Bytes(name)
	if !ok {
		return nil
	}
	d, err := svg.Parse(b)
	if err != nil {
		return nil
	}
	ic.svgs[name] = d
	return d
}

// Raster returns the named icon as a bitmap (nil if it is not a PNG).
func (ic *Icons) Raster(name string) *gfx.Image {
	ic.mu.Lock()
	defer ic.mu.Unlock()
	if d, ok := ic.pngs[name]; ok {
		return d
	}
	if ic.res.Kind(name) != "png" {
		return nil
	}
	b, ok := ic.res.Bytes(name)
	if !ok {
		return nil
	}
	img, err := decodePNG(b)
	if err != nil {
		return nil
	}
	ic.pngs[name] = img
	return img
}

// Resources is the underlying reader (for texts and names).
func (ic *Icons) Resources() *Resources { return ic.res }
