package ui

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
)

// ScrollList is a vertical list of fixed-height rows that scrolls when there
// are more than fit. A short press on a row is passed to it; a drag scrolls
// instead of pressing.
type ScrollList struct {
	Base
	Rows   []View
	RowH   int
	Gap    int
	offset int
	max    int

	grab      View
	startY    int
	startOff  int
	downAt    image.Point
	scrolling bool
}

// NewScrollList makes a list with rows of height rowH.
func NewScrollList(rowH, gap int) *ScrollList {
	return &ScrollList{RowH: rowH, Gap: gap}
}

// SetRows replaces the rows and relays out.
func (l *ScrollList) SetRows(rows []View) {
	l.Rows = rows
	l.offset = 0
	l.layout()
	l.MarkDirty()
}

func (l *ScrollList) SetBounds(r image.Rectangle) {
	l.Base.SetBounds(r)
	l.layout()
}

// content is the full height of all rows.
func (l *ScrollList) content() int {
	n := len(l.Rows)
	if n == 0 {
		return 0
	}
	return n*l.RowH + (n-1)*l.Gap
}

func (l *ScrollList) layout() {
	l.max = max(0, l.content()-l.bounds.Dy())
	if l.offset > l.max {
		l.offset = l.max
	}
	if l.offset < 0 {
		l.offset = 0
	}
	y := l.bounds.Min.Y - l.offset
	for _, row := range l.Rows {
		row.SetBounds(image.Rect(l.bounds.Min.X, y, l.bounds.Max.X, y+l.RowH))
		y += l.RowH + l.Gap
	}
}

func (l *ScrollList) rowAt(p image.Point) View {
	for _, row := range l.Rows {
		if p.In(row.Bounds()) {
			return row
		}
	}
	return nil
}

func (l *ScrollList) Touch(phase TouchPhase, p image.Point) bool {
	switch phase {
	case Down:
		l.downAt = p
		l.startY = p.Y
		l.startOff = l.offset
		l.scrolling = false
		l.grab = l.rowAt(p)
		return true
	case Move:
		if !l.scrolling && abs(p.Y-l.startY) > 8 && l.max > 0 {
			l.scrolling = true
			if l.grab != nil {
				l.grab.Touch(Up, p) // cancel the press we may have begun
				l.grab = nil
			}
		}
		if l.scrolling {
			l.offset = clampInt(l.startOff-(p.Y-l.startY), 0, l.max)
			l.layout()
			l.MarkDirty()
		} else if l.grab != nil {
			l.grab.Touch(Move, p)
		}
		return true
	case Up:
		if !l.scrolling && l.grab != nil {
			l.grab.Touch(Down, l.downAt)
			l.grab.Touch(Up, p)
		}
		l.grab = nil
		l.scrolling = false
		return true
	}
	return true
}

func (l *ScrollList) Draw(dst *gfx.Image, c *Context, clip image.Rectangle) {
	area := clip.Intersect(l.bounds)
	if area.Empty() {
		return
	}
	for _, row := range l.Rows {
		r := row.Bounds().Intersect(area)
		if !r.Empty() {
			row.Draw(dst, c, r)
		}
	}
	// A hint that there is more below or above.
	if l.offset > 0 {
		Fill(dst, image.Rect(l.bounds.Min.X, l.bounds.Min.Y, l.bounds.Max.X, l.bounds.Min.Y+2), c.Theme.Line, clip)
	}
	if l.offset < l.max {
		Fill(dst, image.Rect(l.bounds.Min.X, l.bounds.Max.Y-2, l.bounds.Max.X, l.bounds.Max.Y), c.Theme.Line, clip)
	}
}

func (l *ScrollList) Dirty() []image.Rectangle {
	out := l.Base.Dirty()
	if out != nil {
		for _, row := range l.Rows {
			row.Dirty()
		}
		return out
	}
	for _, row := range l.Rows {
		for _, d := range row.Dirty() {
			if r := d.Intersect(l.bounds); !r.Empty() {
				out = append(out, r)
			}
		}
	}
	return out
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
