package pages

import (
	"image"
	"time"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// recButton is the round REC control of the status bar: a ring whose colour
// is the recorder's state (green ready, red recording, amber paused, blue
// without storage). A tap starts, pauses or resumes; holding it stops, as
// on the device.
type recButton struct {
	ui.Base
	env     *Env
	pressed bool
	downAt  time.Time
	lastSt  int64
	held    bool
}

func newRecButton(env *Env) *recButton { return &recButton{env: env, lastSt: -1} }

func (b *recButton) Touch(phase ui.TouchPhase, p image.Point) bool {
	switch phase {
	case ui.Down:
		b.pressed, b.held, b.downAt = true, false, time.Now()
		b.MarkDirty()
	case ui.Move:
		b.hold()
	case ui.Up:
		b.hold()
		if b.pressed && !b.held {
			b.tap()
		}
		b.pressed = false
		b.MarkDirty()
	}
	return true
}

// holdStops reports the device's REC button setting: 0 = a press pauses
// and holding stops (the default), 1 = a press stops and holding pauses.
func (b *recButton) holdStops() bool {
	return b.env.Desk.Int(desk.SystemNode, 0, "systemRecButtonSetting") != 1
}

// hold acts once the finger has rested for 1.2 s: it stops the recording
// (or pauses it, with the other setting).
func (b *recButton) hold() {
	if b.pressed && !b.held && time.Since(b.downAt) > 1200*time.Millisecond {
		b.held = true
		if st := b.env.Desk.RecordState(); st == 1 || st == 2 {
			if b.holdStops() || st == 1 {
				b.env.Desk.RequestRecord(0)
			} else {
				b.env.Desk.RequestRecord(1)
			}
		}
	}
}

func (b *recButton) tap() {
	switch b.env.Desk.RecordState() {
	case 2:
		if b.holdStops() {
			b.env.Desk.RequestRecord(1)
			b.env.Nav.Toast(b.env.T("HOLD REC TO STOP RECORDING"))
		} else {
			b.env.Desk.RequestRecord(0)
		}
	case 1:
		b.env.Desk.RequestRecord(2)
	case 3:
		b.env.Nav.Toast(b.env.T("NO STORAGE AVAILABLE"))
	default:
		b.env.Desk.RequestRecord(2)
	}
}

func (b *recButton) Dirty() []image.Rectangle {
	if st := b.env.Desk.RecordState(); st != b.lastSt {
		b.lastSt = st
		b.MarkDirty()
	}
	// While held, redraw so the ring can show the hold.
	if b.pressed && !b.held && time.Since(b.downAt) > 1200*time.Millisecond {
		b.hold()
		b.MarkDirty()
	}
	return b.Base.Dirty()
}

func (b *recButton) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	r := ui.Square(b.Bounds())
	cx, cy := float64(r.Min.X+r.Max.X)/2, float64(r.Min.Y+r.Max.Y)/2
	rad := float64(r.Dx()) / 2
	ring := th.Accent
	fill := gfx.Hex(0x101010)
	switch b.env.Desk.RecordState() {
	case 2:
		ring, fill = th.Record, gfx.Hex(0x7a1010)
	case 1:
		ring, fill = th.Warn, gfx.Hex(0x6a4600)
	case 3:
		ring = th.Blue
	}
	if b.pressed {
		fill = gfx.Mix(fill, th.Text, 0.15)
	}
	ui.Circle(dst, cx, cy, rad-2, fill, clip)
	ui.Ring(dst, cx, cy, rad-6, 9, ring, clip)
	ui.Text(dst, th.Fonts.Label, "REC", r, font.Center, th.Text, clip)
}
