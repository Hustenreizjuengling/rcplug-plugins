package ui

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
)

// View is one part of the screen: it is laid out into a rectangle, draws
// itself and takes touches. Views form a tree; a container lays out and
// draws its children.
type View interface {
	// SetBounds places the view (and lays out its children).
	SetBounds(r image.Rectangle)
	// Bounds is where the view is.
	Bounds() image.Rectangle
	// Draw paints the view into dst, within clip.
	Draw(dst *gfx.Image, c *Context, clip image.Rectangle)
	// Touch handles a touch at p (inside the view). phase is Down, Move or
	// Up. It returns true if the view took the touch.
	Touch(phase TouchPhase, p image.Point) bool
	// Dirty reports the parts that changed since the last Draw and clears
	// the mark. A view that marks itself dirty is redrawn.
	Dirty() []image.Rectangle
}

// TouchPhase is a step of a touch.
type TouchPhase uint8

const (
	Down TouchPhase = iota
	Move
	Up
)

// Base is embedded by views: it keeps the bounds and the dirty mark.
type Base struct {
	bounds image.Rectangle
	dirty  bool
}

func (b *Base) SetBounds(r image.Rectangle) {
	if r != b.bounds {
		b.bounds = r
		b.dirty = true
	}
}
func (b *Base) Bounds() image.Rectangle            { return b.bounds }
func (b *Base) Touch(TouchPhase, image.Point) bool { return false }

// MarkDirty asks for the view to be redrawn.
func (b *Base) MarkDirty() { b.dirty = true }

// Dirty returns the whole view when it was marked, else nothing.
func (b *Base) Dirty() []image.Rectangle {
	if b.dirty {
		b.dirty = false
		return []image.Rectangle{b.bounds}
	}
	return nil
}

// Container holds child views in a fixed layout set by the embedding view.
type Container struct {
	Base
	Children []View
	grab     View // the child that took the current touch
}

// Add appends a child.
func (c *Container) Add(v ...View) { c.Children = append(c.Children, v...) }

// Clear removes the children.
func (c *Container) Clear() {
	c.Children = nil
	c.MarkDirty()
}

func (c *Container) Draw(dst *gfx.Image, ctx *Context, clip image.Rectangle) {
	for _, ch := range c.Children {
		r := ch.Bounds().Intersect(clip)
		if !r.Empty() {
			ch.Draw(dst, ctx, r)
		}
	}
}

func (c *Container) Touch(phase TouchPhase, p image.Point) bool {
	switch phase {
	case Down:
		// Front to back (last drawn is on top).
		for i := len(c.Children) - 1; i >= 0; i-- {
			ch := c.Children[i]
			if p.In(ch.Bounds()) && ch.Touch(Down, p) {
				c.grab = ch
				return true
			}
		}
		return false
	default:
		if c.grab == nil {
			return false
		}
		took := c.grab.Touch(phase, p)
		if phase == Up {
			c.grab = nil
		}
		return took
	}
}

// Dirty is the container's own mark (a relayout) plus its children's.
func (c *Container) Dirty() []image.Rectangle {
	out := c.Base.Dirty()
	if out != nil {
		// A relayout repaints everything; the children's marks are covered.
		for _, ch := range c.Children {
			ch.Dirty()
		}
		return out
	}
	for _, ch := range c.Children {
		out = append(out, ch.Dirty()...)
	}
	return out
}
