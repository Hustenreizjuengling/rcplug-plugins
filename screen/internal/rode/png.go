package rode

import (
	"bytes"
	"image/png"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
)

// decodePNG decodes a PNG into premultiplied pixels.
func decodePNG(b []byte) (*gfx.Image, error) {
	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	return gfx.FromImage(img), nil
}
