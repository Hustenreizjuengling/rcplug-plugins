package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
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

// columns cuts r into n equal columns with a gap.
func columns(r image.Rectangle, n, gap int) []image.Rectangle {
	out := make([]image.Rectangle, 0, n)
	cw := (r.Dx() - (n-1)*gap) / n
	for i := 0; i < n; i++ {
		x := r.Min.X + i*(cw+gap)
		out = append(out, image.Rect(x, r.Min.Y, x+cw, r.Max.Y))
	}
	return out
}

// centred places a w × h box in the middle of r.
func centred(r image.Rectangle, w, h int) image.Rectangle {
	x := r.Min.X + (r.Dx()-w)/2
	y := r.Min.Y + (r.Dy()-h)/2
	return image.Rect(x, y, x+w, y+h)
}

// tileGrid lays tiles (a square card plus a caption underneath) in rows of
// cols, centred in r, and returns the cells.
func tileGrid(r image.Rectangle, n, cols, side, gap int) []image.Rectangle {
	if n == 0 {
		return nil
	}
	if cols > n {
		cols = n
	}
	rows := (n + cols - 1) / cols
	cellH := side + 48
	totalH := rows*cellH + (rows-1)*gap
	y0 := r.Min.Y + max(0, (r.Dy()-totalH)/2)
	var out []image.Rectangle
	for i := 0; i < n; i++ {
		row, col := i/cols, i%cols
		inRow := cols
		if row == rows-1 {
			inRow = n - row*cols
		}
		totalW := inRow*side + (inRow-1)*gap
		x0 := r.Min.X + (r.Dx()-totalW)/2
		x := x0 + col*(side+gap)
		y := y0 + row*(cellH+gap)
		out = append(out, image.Rect(x, y, x+side, y+cellH))
	}
	return out
}

// headerButton is a square header control in RØDE's grey (back, home, the
// channel cog).
func headerButton(icon string, onTap func()) *ui.Button {
	b := ui.NewButton("", onTap)
	b.Icon = icon
	return b
}

// chevron is a plain "‹" / "›" header control (previous / next).
func chevron(env *Env, next bool, enabled bool, onTap func()) *ui.Button {
	label := "‹"
	if next {
		label = "›"
	}
	b := ui.NewButton(label, onTap)
	b.Style = ui.StylePlain
	b.Font = env.Ctx.Theme.Fonts.Huge
	b.Enabled = enabled
	return b
}

// caption is a dim line of text.
func caption(env *Env, text string, r image.Rectangle, a font.Align) *ui.Label {
	l := ui.NewLabel(text, env.Ctx.Theme.Fonts.Body, env.Ctx.Theme.TextDim, a)
	l.SetBounds(r)
	return l
}
