package pages

import (
	"image"
	"math"
	"time"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// The small indicators of the mixer's status bar and the passing level
// overlay, as RØDE's screen has them (SystemIndicator, OutputControlOverlay).

// Indicator colours as the screen program paints them: lit green, else grey.
var (
	indicatorOn  = gfx.Hex(0x2ad159)
	indicatorOff = gfx.Hex(0x515151)
)

// sysIndicator shows the network: the Wi-Fi glyph while Wi-Fi is on, else
// the wired glyph, lit green when that link is up; the cellular glyph
// beside it when cellular data is enabled. (The screen program's
// SystemIndicator: IconWired / IconWifi / IconCell in 2ad159 or 515151.)
type sysIndicator struct {
	ui.Base
	env  *Env
	last string
}

// netState reads the network node: which glyphs to show and whether lit.
func (v *sysIndicator) netState() (icon string, lit bool, cell bool, cellLit bool) {
	d := v.env.Desk
	n := desk.NetworkNode
	if d.Bool(n, 0, "wifi") {
		icon = "IconWifi"
		lit = d.Str(n, 0, "wifiSSID") != "" && d.Str(n, 0, "wifiIpAddress") != ""
	} else {
		icon = "IconWired"
		lit = d.Bool(n, 0, "wiredConnected") && d.Str(n, 0, "ipAddress") != ""
	}
	cell = d.Has(n, 0, "cellEnabled") && d.Bool(n, 0, "cellEnabled")
	cellLit = cell && d.Bool(n, 0, "cellUSBFound")
	return
}

func (v *sysIndicator) Dirty() []image.Rectangle {
	icon, lit, cell, cellLit := v.netState()
	if s := icon + boolText(lit) + boolText(cell) + boolText(cellLit); s != v.last {
		v.last = s
		v.MarkDirty()
	}
	return v.Base.Dirty()
}

func (v *sysIndicator) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	r := v.Bounds()
	icon, lit, cell, cellLit := v.netState()
	col := indicatorOff
	if lit {
		col = indicatorOn
	}
	side := r.Dy()
	x := r.Max.X - side
	c.Icon(dst, icon, image.Rect(x, r.Min.Y, x+side, r.Max.Y), col)
	if cell {
		col = indicatorOff
		if cellLit {
			col = indicatorOn
		}
		c.Icon(dst, "IconCell", image.Rect(x-side-8, r.Min.Y, x-8, r.Max.Y), col)
	}
}

// callMeIndicator is the CallMe arrows next to the cog: with a dot while a
// call slot is busy, the no-network glyph without a network. A tap opens the
// CallMe menu. Shown only on a desk with CallMe.
type callMeIndicator struct {
	ui.Base
	env     *Env
	last    string
	pressed bool
}

// callMeState reads the call slots: "" when the desk has no CallMe.
func (v *callMeIndicator) callMeState() string {
	d := v.env.Desk
	if d.Count("SIPCALLSLOTS") == 0 {
		return ""
	}
	_, lit, _, cellLit := (&sysIndicator{env: v.env}).netState()
	if !lit && !cellLit {
		return "CallMeNoNetwork"
	}
	for i := 0; i < d.Count("SIPCALLSLOTS"); i++ {
		if st := d.Str("SIPCALLSLOTS", i, "sipSlotCallState"); st != "" && st != "Idle" {
			return "CallMeArrows_WithDot"
		}
	}
	return "CallMeArrows"
}

func (v *callMeIndicator) Dirty() []image.Rectangle {
	if s := v.callMeState(); s != v.last {
		v.last = s
		v.MarkDirty()
	}
	return v.Base.Dirty()
}

func (v *callMeIndicator) Touch(phase ui.TouchPhase, p image.Point) bool {
	if v.callMeState() == "" {
		return false
	}
	switch phase {
	case ui.Down:
		v.pressed = true
	case ui.Up:
		if v.pressed {
			v.env.Nav.Open(IDCallMe)
		}
		v.pressed = false
	}
	return true
}

func (v *callMeIndicator) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	icon := v.callMeState()
	if icon == "" {
		return
	}
	c.Icon(dst, icon, v.Bounds(), c.Theme.Text)
}

// --- the level overlay ---

// levelOverlay is the passing card that shows an output's level while the
// main knob turns it or after its icon was tapped: the output's ring with
// the level arc, its name and a slider that sets it too.
type levelOverlay struct {
	ui.Container
	env *Env
	bt  bool
}

// overlayFor is how long the level overlay stays after the last change.
const overlayFor = 2500 * time.Millisecond

// newLevelOverlay builds the overlay for the monitor (bt false) or the
// Bluetooth output, laid out for a w × h screen.
func newLevelOverlay(env *Env, bt bool, w, h int) *levelOverlay {
	th := env.Ctx.Theme
	d := env.Desk
	o := &levelOverlay{env: env, bt: bt}
	r := image.Rect(w/2-330, 150, w/2+330, 150+170)
	o.SetBounds(r)
	card := ui.NewPanelView(th.PanelHi)
	card.Radius = 20
	card.SetBounds(r)
	ring := &monitorKnob{env: env, bt: bt, plain: true}
	ring.SetBounds(image.Rect(r.Min.X+30, r.Min.Y+25, r.Min.X+150, r.Min.Y+145))
	key := "MONITOR"
	level := d.MonitorLevel
	set := d.SetMonitorLevel
	if bt {
		key = "BLUETOOTH"
		level = func() float64 { return d.Num(desk.OutputNode, 0, "outputBTLevel") }
		set = func(v float64) { d.SetFloat(desk.OutputNode, 0, "outputBTLevel", math.Max(0, math.Min(1, v))) }
	}
	sl := ui.NewSlider(level, set)
	if !bt && d.Bool(desk.OutputNode, 0, "outputMonFixed") {
		sl.Change = nil // a fixed output level cannot be adjusted
	}
	sl.Label = env.T(key)
	sl.Text = func(v float64) string { return pct(v) }
	sl.Step = 0.01
	sl.SetBounds(image.Rect(r.Min.X+170, r.Min.Y+20, r.Max.X-24, r.Max.Y-20))
	o.Add(card, ring, sl)
	return o
}

// Touch keeps the overlay from passing taps to the mixer behind it.
func (o *levelOverlay) Touch(phase ui.TouchPhase, p image.Point) bool {
	if o.Container.Touch(phase, p) {
		return true
	}
	return p.In(o.Bounds())
}

// monitorStateText is the caption under a level ring (for tests).
func monitorStateText(env *Env, bt bool) string {
	k := &monitorKnob{env: env, bt: bt}
	return k.state()
}

var _ = font.Center
