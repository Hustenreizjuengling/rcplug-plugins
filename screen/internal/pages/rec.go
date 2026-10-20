package pages

import (
	"fmt"
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// recButton is the record control: it shows the state by colour and the
// elapsed time, and starts or stops recording on a tap (via the recorder's
// request property; the console applies the device's own rules).
type recButton struct {
	ui.Base
	env     *Env
	pressed bool
	lastMs  int64
	lastSt  int64
}

func newRecButton(env *Env) *recButton { return &recButton{env: env, lastSt: -1} }

func (b *recButton) state() int64  { return b.env.Desk.Int(desk.RecorderNode, 0, "recordState") }
func (b *recButton) timeMs() int64 { return b.env.Desk.Int(desk.RecorderNode, 0, "recordTimeMs") }

func (b *recButton) Touch(phase ui.TouchPhase, p image.Point) bool {
	switch phase {
	case ui.Down:
		b.pressed = true
		b.MarkDirty()
	case ui.Up:
		if b.pressed {
			b.toggle()
		}
		b.pressed = false
		b.MarkDirty()
	}
	return true
}

func (b *recButton) toggle() {
	st := b.state()
	if st == 2 || st == 1 { // recording or paused → stop
		msg := b.env.TR("HOLD REC TO STOP RECORDING", "Aufnahme stoppen?", "Stop the recording?")
		b.env.Nav.Confirm(b.env.T("REC"), msg, b.env.TR("STOP", "Stoppen", "Stop"), false, func() {
			b.env.Desk.SetInt(desk.RecorderNode, 0, "requestRecordState", 0)
		})
		return
	}
	// ready → start
	b.env.Desk.SetInt(desk.RecorderNode, 0, "requestRecordState", 2)
}

func (b *recButton) Dirty() []image.Rectangle {
	if ms, st := b.timeMs(), b.state(); ms != b.lastMs || st != b.lastSt {
		b.lastMs, b.lastSt = ms, st
		b.MarkDirty()
	}
	return b.Base.Dirty()
}

func (b *recButton) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	r := b.Bounds()
	st := b.state()
	fill := th.Panel
	dot := th.TextMuted
	switch st {
	case 2:
		fill, dot = th.Record, gfx.Hex(0xffffff)
	case 1:
		fill, dot = th.Warn, gfx.Hex(0x101010)
	case 3:
		fill = th.Panel
	}
	if b.pressed {
		fill = th.PanelHi
	}
	ui.Panel(dst, r, th.Radius, fill, clip)
	// A record dot + the time.
	cy := (r.Min.Y + r.Max.Y) / 2
	circ := &gfx.Path{}
	circ.Circle(float64(r.Min.X+22), float64(cy), 9)
	dst.Fill(circ, gfx.Identity, gfx.NonZero, gfx.Solid(dot), clip)
	label := recTime(b.timeMs())
	if st == 0 {
		label = "REC"
	} else if st == 3 {
		label = b.env.T("NO STORAGE AVAILABLE")
	}
	txtCol := th.Text
	if st == 1 {
		txtCol = gfx.Hex(0x101010)
	}
	ui.Text(dst, th.Fonts.Label, label, image.Rect(r.Min.X+38, r.Min.Y, r.Max.X-8, r.Max.Y), font.Left, txtCol, clip)
}

// recTime formats milliseconds as H:MM:SS or MM:SS.
func recTime(ms int64) string {
	s := ms / 1000
	h, m, sec := s/3600, (s%3600)/60, s%60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, sec)
	}
	return fmt.Sprintf("%02d:%02d", m, sec)
}
