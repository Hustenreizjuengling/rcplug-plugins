package main

import (
	"image"
	"time"
)

// The picture as the panel shows it, made of the page (drawn again only
// when it changes) and what moves over it (the rain, the cursor), which is
// drawn every frame and copied to the display on its own: about a fifth of
// the panel instead of all of it.

// renderedPage is a page drawn and turned for the panel.
type renderedPage struct {
	canvas   *image.NRGBA // landscape
	portrait []byte       // as the panel shows it, pitch bytes a row
	cursor   image.Rectangle
}

func renderPage(in Info, pitch int) (renderedPage, error) {
	img, err := page(in)
	if err != nil {
		return renderedPage{}, err
	}
	p := make([]byte, pitch*canvasW)
	turn(img, p, pitch, panelTurn)
	return renderedPage{canvas: img, portrait: p, cursor: cursorRect(in)}, nil
}

type scene struct {
	pitch  int
	start  time.Time
	frame  []byte // the whole picture, as the panel shows it
	canvas *image.NRGBA
	cursor image.Rectangle
	rain   *rain
	// full counts the display's buffers that still need the whole frame
	// (after a new page; each buffer once).
	full int
}

func newScene(pitch int, start time.Time) (*scene, error) {
	r, err := newRain(rainArea, start.UnixNano())
	if err != nil {
		return nil, err
	}
	return &scene{pitch: pitch, start: start, frame: make([]byte, pitch*canvasW), rain: r}, nil
}

// setPage puts a new page under the moving parts; the display's buffers
// (n of them) get the whole frame next.
func (sc *scene) setPage(p renderedPage, n int) {
	sc.canvas, sc.cursor = p.canvas, p.cursor
	copy(sc.frame, p.portrait)
	sc.full = n
}

// step moves the rain on by dt, draws what moves at now, and says where
// the panel changed.
func (sc *scene) step(now time.Time, dt time.Duration) []image.Rectangle {
	sc.rain.step(dt)
	sc.rain.draw(sc.canvas)
	on := now.Sub(sc.start)%time.Second < time.Second/2
	drawCursor(sc.canvas, sc.cursor, on)
	var out []image.Rectangle
	for _, r := range []image.Rectangle{rainArea, sc.cursor} {
		if r.Empty() {
			continue
		}
		turnRect(sc.canvas, r, sc.frame, sc.pitch, panelTurn)
		out = append(out, panelRect(r, panelTurn))
	}
	return out
}

// copyTo copies the frame into a buffer of the display: whole while it
// needs it, otherwise only the parts that changed.
func (sc *scene) copyTo(dst []byte, changed []image.Rectangle) {
	if sc.full > 0 {
		copy(dst, sc.frame)
		sc.full--
		return
	}
	for _, r := range changed {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			o := y*sc.pitch + r.Min.X*4
			copy(dst[o:o+r.Dx()*4], sc.frame[o:o+r.Dx()*4])
		}
	}
}
