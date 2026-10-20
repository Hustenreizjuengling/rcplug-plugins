package ui

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
)

// Root holds the view tree and the picture it draws into. It redraws only
// the parts that changed and reports them, so the panel copies little.
type Root struct {
	ctx   *Context
	img   *gfx.Image
	root  View
	first bool

	// the view that took the current touch
	grab    View
	grabOff image.Point
}

// NewRoot makes a root for a w × h screen.
func NewRoot(ctx *Context, w, h int) *Root {
	r := &Root{ctx: ctx, img: gfx.NewImage(w, h), first: true}
	return r
}

// Context is the drawing context.
func (r *Root) Context() *Context { return r.ctx }

// Image is the picture (for the panel to show).
func (r *Root) Image() *gfx.Image { return r.img }

// SetView puts v on the screen (filling it) and draws it next time.
func (r *Root) SetView(v View) {
	r.root = v
	if v != nil {
		v.SetBounds(r.img.Bounds())
	}
	r.first = true
	r.grab = nil
}

// View is the current view.
func (r *Root) View() View { return r.root }

// Render draws what changed and returns those rectangles (nil: nothing
// changed; the first render draws and returns the whole screen).
func (r *Root) Render() []image.Rectangle {
	if r.root == nil {
		return nil
	}
	if r.first {
		r.first = false
		r.root.Dirty() // the views pull their state (and the marks clear)
		r.img.Set(r.img.Bounds(), r.ctx.Theme.Bg)
		r.root.Draw(r.img, r.ctx, r.img.Bounds())
		return []image.Rectangle{r.img.Bounds()}
	}
	dirty := r.root.Dirty()
	if len(dirty) == 0 {
		return nil
	}
	dirty = merge(dirty)
	for _, d := range dirty {
		d = d.Intersect(r.img.Bounds())
		if d.Empty() {
			continue
		}
		r.img.Set(d, r.ctx.Theme.Bg)
		r.root.Draw(r.img, r.ctx, d)
	}
	return dirty
}

// Touch routes a touch to the view under it. The view that takes the Down
// keeps the Move and Up even if the finger leaves it.
func (r *Root) Touch(phase TouchPhase, p image.Point) {
	if r.root == nil {
		return
	}
	switch phase {
	case Down:
		if p.In(r.root.Bounds()) && r.root.Touch(Down, p) {
			r.grab = r.root
		}
	case Move:
		if r.grab != nil {
			r.grab.Touch(Move, p)
		}
	case Up:
		if r.grab != nil {
			r.grab.Touch(Up, p)
			r.grab = nil
		}
	}
}

// merge joins overlapping rectangles so a changed area is drawn once. It
// unions any two that touch, repeatedly; the lists are short.
func merge(rs []image.Rectangle) []image.Rectangle {
	out := make([]image.Rectangle, 0, len(rs))
	for _, r := range rs {
		if r.Empty() {
			continue
		}
		merged := false
		for i := range out {
			if out[i].Overlaps(r) || adjacent(out[i], r) {
				out[i] = out[i].Union(r)
				merged = true
				break
			}
		}
		if !merged {
			out = append(out, r)
		}
	}
	// A second pass unions rectangles that became neighbours.
	changed := true
	for changed {
		changed = false
		for i := 0; i < len(out); i++ {
			for j := i + 1; j < len(out); j++ {
				if out[i].Overlaps(out[j]) || adjacent(out[i], out[j]) {
					out[i] = out[i].Union(out[j])
					out = append(out[:j], out[j+1:]...)
					changed = true
					j--
				}
			}
		}
	}
	return out
}

func adjacent(a, b image.Rectangle) bool {
	return a.Inset(-2).Overlaps(b)
}
