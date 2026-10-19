package ui

import (
	"image"
	"testing"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
)

func testContext() *Context {
	return NewContext(Dark(Liberation()), nil)
}

func TestRootPartialRedraw(t *testing.T) {
	ctx := testContext()
	root := NewRoot(ctx, 200, 100)
	page := &Container{}
	page.SetBounds(image.Rect(0, 0, 200, 100))
	val := 0.0
	f := NewFader(func() float64 { return val }, func(v float64) { val = v })
	f.SetBounds(image.Rect(10, 10, 60, 90))
	lbl := NewLabel("hi", ctx.Theme.Fonts.Body, gfx.Hex(0xffffff), 0)
	lbl.SetBounds(image.Rect(100, 10, 190, 40))
	page.Add(f, lbl)
	root.SetView(page)

	// First render covers the whole screen.
	if d := root.Render(); len(d) != 1 || d[0] != root.Image().Bounds() {
		t.Fatalf("first render %v", d)
	}
	// Nothing changed: no rectangles.
	if d := root.Render(); d != nil {
		t.Fatalf("idle render %v", d)
	}
	// Only the label changes: only its area is redrawn.
	lbl.Set("bye")
	d := root.Render()
	if len(d) != 1 || !d[0].In(lbl.Bounds().Inset(-3)) {
		t.Fatalf("label change redrew %v, want ~%v", d, lbl.Bounds())
	}
	// A touch on the fader moves the value and marks it dirty.
	root.Touch(Down, image.Pt(35, 85))
	root.Touch(Move, image.Pt(35, 15))
	root.Touch(Up, image.Pt(35, 15))
	if val < 0.9 {
		t.Fatalf("fader drag set %v", val)
	}
	d = root.Render()
	if len(d) != 1 || !f.Bounds().Overlaps(d[0]) {
		t.Fatalf("fader redraw %v", d)
	}
}

func TestMergeRects(t *testing.T) {
	got := merge([]image.Rectangle{
		image.Rect(0, 0, 10, 10), image.Rect(5, 5, 15, 15), image.Rect(100, 100, 110, 110),
	})
	if len(got) != 2 {
		t.Fatalf("merged to %v", got)
	}
	// The first two join into one box 0..15; the far one stays.
	var near, far image.Rectangle
	for _, r := range got {
		if r.Overlaps(image.Rect(0, 0, 1, 1)) {
			near = r
		} else {
			far = r
		}
	}
	if near != image.Rect(0, 0, 15, 15) || far != image.Rect(100, 100, 110, 110) {
		t.Fatalf("union %v / %v", near, far)
	}
}

func TestTouchGrab(t *testing.T) {
	ctx := testContext()
	root := NewRoot(ctx, 100, 100)
	page := &Container{}
	page.SetBounds(image.Rect(0, 0, 100, 100))
	got := ""
	b := NewButton("x", func() { got = "tap" })
	b.SetBounds(image.Rect(0, 0, 50, 50))
	page.Add(b)
	root.SetView(page)
	// A press inside, release outside the button still fires (grabbed).
	root.Touch(Down, image.Pt(10, 10))
	root.Touch(Move, image.Pt(80, 80))
	root.Touch(Up, image.Pt(80, 80))
	if got != "tap" {
		t.Fatal("grabbed button did not fire on release outside")
	}
}
