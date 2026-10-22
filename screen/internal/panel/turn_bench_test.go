package panel

import (
	"image"
	"testing"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
)

func BenchmarkTurnCanvas(b *testing.B) {
	page := gfx.NewImage(1280, 720)
	dst := make([]byte, 720*1280*4)
	r := image.Rect(0, 64, 592, 656)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Turn(page, r, dst, 720*4, 90)
	}
}

func BenchmarkTurnFull(b *testing.B) {
	page := gfx.NewImage(1280, 720)
	dst := make([]byte, 720*1280*4)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Turn(page, page.Bounds(), dst, 720*4, 90)
	}
}
