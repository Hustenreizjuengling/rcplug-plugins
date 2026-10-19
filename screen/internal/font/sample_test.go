package font

import (
	"image/png"
	"os"
	"testing"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
)

// TestSample writes a sample of the fonts for looking at:
// RCPLUG_FONT_SAMPLE=<out.png>.
func TestSample(t *testing.T) {
	out := os.Getenv("RCPLUG_FONT_SAMPLE")
	if out == "" {
		t.Skip("set RCPLUG_FONT_SAMPLE")
	}
	m := gfx.NewImage(900, 420)
	m.Clear(gfx.Hex(0x1d1d1d))
	y := 30.0
	for _, sz := range []float64{14, 18, 22, 28, 36, 48} {
		for i, face := range []*Face{Regular(), Bold()} {
			f := face.Sized(sz)
			f.Draw(m, "MISCHPULT Einstellungen Äöü ß 0123 -12.5 dB AV To", 10+float64(i)*0, y, gfx.Hex(0xffffff), m.Bounds())
			y += sz * 1.1
		}
	}
	fh, _ := os.Create(out)
	defer fh.Close()
	png.Encode(fh, m.ToNRGBA())
}
