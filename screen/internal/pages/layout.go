package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// Layout helpers shared across the pages.

// grid cuts r into a cols × rows grid of cells with a gap between them.
func grid(r image.Rectangle, cols, rows, gap int) []image.Rectangle {
	out := make([]image.Rectangle, 0, cols*rows)
	cw := (r.Dx() - (cols-1)*gap) / cols
	ch := (r.Dy() - (rows-1)*gap) / rows
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			cx := r.Min.X + x*(cw+gap)
			cy := r.Min.Y + y*(ch+gap)
			out = append(out, image.Rect(cx, cy, cx+cw, cy+ch))
		}
	}
	return out
}

// rowRects cuts r into rows of height h with a gap, from the top, until they
// no longer fit.
func rowRects(r image.Rectangle, h, gap int) []image.Rectangle {
	var out []image.Rectangle
	for y := r.Min.Y; y+h <= r.Max.Y; y += h + gap {
		out = append(out, image.Rect(r.Min.X, y, r.Max.X, y+h))
	}
	return out
}

// menuTile is a large icon-and-label button for a menu grid.
func menuTile(label, icon string, onTap func()) *ui.Button {
	b := ui.NewButton(label, onTap)
	b.Icon = icon
	b.TileLayout = true
	return b
}

// column cuts r into n equal columns with a gap.
func columns(r image.Rectangle, n, gap int) []image.Rectangle {
	out := make([]image.Rectangle, 0, n)
	cw := (r.Dx() - (n-1)*gap) / n
	for i := 0; i < n; i++ {
		x := r.Min.X + i*(cw+gap)
		out = append(out, image.Rect(x, r.Min.Y, x+cw, r.Max.Y))
	}
	return out
}
