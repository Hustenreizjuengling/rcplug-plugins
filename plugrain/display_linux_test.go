package main

import (
	"image"
	"os"
	"syscall"
	"testing"
)

// The DRM calls against a real card, where this machine has one: finding
// the panel and making, mapping and freeing two buffers. Nothing is shown
// (that needs the card's master, which a desktop holds).
func TestDisplayBuffers(t *testing.T) {
	f, err := os.OpenFile("/dev/dri/card0", os.O_RDWR|syscall.O_CLOEXEC, 0)
	if err != nil {
		t.Skip(err)
	}
	d := &display{f: f}
	defer d.Close()
	if err := d.buffers(2); err != nil {
		t.Skip(err) // no connected output, or no dumb buffers
	}
	t.Logf("%s: %d × %d at %d Hz, connector %d, CRTC %d, pitch %d, %d buffers", d.ModeName, d.W, d.H, d.Refresh, d.conn, d.crtc, d.pitch, len(d.bufs))
	if d.W <= 0 || d.H <= 0 || d.pitch < d.W*4 || d.crtc == 0 || len(d.bufs) != 2 || !d.Flips || d.Refresh <= 0 {
		t.Fatalf("%+v", d)
	}
	for _, b := range d.bufs {
		if len(b.mem) < d.pitch*d.H || b.fb == 0 {
			t.Fatalf("buffer %+v", b.fb)
		}
	}
	back := d.Back()
	back[len(back)-1] = 0xff
	if d.bufs[1].mem[len(back)-1] != 0xff || d.bufs[0].mem[len(back)-1] != 0 {
		t.Fatal("the back buffer is not the one off the panel")
	}
}

// Events as the touchscreen sends them, through a pipe: a finger down,
// moving, up, scaled to the panel.
func TestTouchEvents(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	ts := &touchscreen{f: r, x: axis{0, 799}, y: axis{0, 1279}}
	out := make(chan touchEvent, 8)
	go ts.Run(720, 1280, out)
	ev := func(typ, code uint16, val int32) []byte {
		b := make([]byte, eventSize)
		le.PutUint16(b[16:], typ)
		le.PutUint16(b[18:], code)
		le.PutUint32(b[20:], uint32(val))
		return b
	}
	var b []byte
	frame := func(e ...[]byte) {
		for _, x := range e {
			b = append(b, x...)
		}
		b = append(b, ev(evSyn, synReport, 0)...)
	}
	frame(ev(evKey, btnTouch, 1), ev(evAbs, absX, 400), ev(evAbs, absY, 640))
	frame(ev(evAbs, absX, 799))
	frame(ev(evAbs, absY, 0))
	frame(ev(evKey, btnTouch, 0))
	frame(ev(evAbs, absX, 10)) // up: nothing
	w.Write(b)
	w.Close()
	var got []touchEvent
	for e := range out {
		got = append(got, e)
	}
	want := []touchEvent{{true, image.Pt(360, 640)}, {true, image.Pt(719, 640)}, {true, image.Pt(719, 0)}, {false, image.Pt(719, 0)}}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%d: got %v, want %v", i, got[i], want[i])
		}
	}
}
