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

// HomePage is the mixer, laid out as the device shows it: a status bar (REC,
// the clock, storage, the monitor knob, the master meter, the settings cog),
// the strips with their dBFS scales, meters and fader caps, and the SMART
// pads of the current bank on the right. The main knob drives the monitor
// level and, after a press, the virtual faders.
type HomePage struct {
	selected int       // the strip whose listen/mute bar is open (−1 none)
	barUntil time.Time // when the bar hides again
	knob     int       // the knob's focus: 0 monitor, 1… the virtual strips
	knobOn   bool      // a turn or press happened (shows the focus)
}

// NewHome makes the mixer page.
func NewHome() Page { return &HomePage{selected: -1} }

func (p *HomePage) Title() string   { return "" }
func (p *HomePage) FullBleed() bool { return true }

const (
	barH      = 118 // the status bar
	stripsW   = 872 // the strips' area, the pads take the rest
	meterTop  = 152 // 0 dBFS
	meterBase = 626 // −60 dBFS
)

func (p *HomePage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Backdrop{Plain: true}
	c.Fill = env.Ctx.Theme.Bg
	c.SetBounds(r)
	th := env.Ctx.Theme

	bar := image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+barH)
	c.Add(p.statusBar(env, bar)...)

	n := env.Desk.StripCount()
	if n == 0 {
		lbl := ui.NewLabel(env.TR("", "Verbinde mit dem Pult …", "Connecting to the desk …"), th.Fonts.Title, th.TextDim, font.Center)
		lbl.SetBounds(image.Rect(r.Min.X, bar.Max.Y, r.Max.X, r.Max.Y))
		c.Add(lbl)
		return c
	}
	area := image.Rect(r.Min.X, bar.Max.Y, r.Min.X+stripsW, r.Max.Y)
	w := area.Dx() / n
	physical := env.Desk.PhysicalFaders()
	for i := 0; i < n; i++ {
		cell := image.Rect(area.Min.X+i*w, area.Min.Y, area.Min.X+(i+1)*w, area.Max.Y)
		s := &stripView{env: env, page: p, strip: i, virtual: i >= physical, lastPhys: i == physical-1}
		s.SetBounds(cell)
		c.Add(s)
	}
	c.Add(p.pads(env, image.Rect(area.Max.X+10, bar.Max.Y+14, r.Max.X-10, r.Max.Y-10))...)

	// The listen / channel / mute bar of the selected strip, on top.
	if p.selected >= 0 && p.selected < n && time.Now().Before(p.barUntil) {
		c.Add(p.stripBar(env, area, w, n)...)
		c.Add(&updater{get: func() {
			if time.Now().After(p.barUntil) {
				p.selected = -1
				env.Nav.Refresh()
			}
		}})
	}
	return c
}

// select opens a strip's bar (and makes it the knob's strip if virtual).
func (p *HomePage) selectStrip(env *Env, strip int) {
	p.selected = strip
	p.barUntil = time.Now().Add(4 * time.Second)
	if strip >= env.Desk.PhysicalFaders() {
		p.knob = strip - env.Desk.PhysicalFaders() + 1
	}
	env.Nav.Refresh()
}

// --- the status bar ---

func (p *HomePage) statusBar(env *Env, bar image.Rectangle) []ui.View {
	th := env.Ctx.Theme
	var out []ui.View
	bg := &barBackground{}
	bg.SetBounds(bar)
	out = append(out, bg)

	// REC, with the clock (or the recording timer) beside it.
	rec := newRecButton(env)
	rec.SetBounds(image.Rect(bar.Min.X+28, bar.Min.Y+9, bar.Min.X+128, bar.Min.Y+109))
	out = append(out, rec)
	clock := &clockView{env: env}
	clock.SetBounds(image.Rect(bar.Min.X+150, bar.Min.Y, bar.Min.X+470, bar.Max.Y))
	out = append(out, clock)

	// Storage: icon, fill bar and remaining time (only with a card/drive).
	st := &storageView{env: env}
	st.SetBounds(image.Rect(bar.Min.X+488, bar.Min.Y+34, bar.Min.X+640, bar.Min.Y+84))
	out = append(out, st)

	// The monitor knob.
	mon := &monitorKnob{env: env, page: p}
	mon.SetBounds(image.Rect(bar.Min.X+712, bar.Min.Y+16, bar.Min.X+798, bar.Min.Y+102))
	out = append(out, mon)

	// Master meter L/R with its scale.
	mm := &masterMeter{env: env}
	mm.SetBounds(image.Rect(bar.Min.X+820, bar.Min.Y+30, bar.Min.X+1206, bar.Min.Y+108))
	out = append(out, mm)

	cog := ui.NewButton("", func() { env.Nav.Open(IDSettings) })
	cog.Icon, cog.Style = "settings_icon", ui.StylePlain
	cog.SetBounds(image.Rect(bar.Max.X-86, bar.Min.Y+24, bar.Max.X-14, bar.Min.Y+96))
	out = append(out, cog)
	_ = th
	return out
}

// barBackground is the status bar's dark gradient with a line underneath.
type barBackground struct{ ui.Base }

func (b *barBackground) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	r := b.Bounds()
	ui.Gradient(dst, r, 0, gfx.Hex(0x3a3a3a), gfx.Hex(0x1a1a1a), clip)
	ui.Fill(dst, image.Rect(r.Min.X, r.Max.Y-2, r.Max.X, r.Max.Y), gfx.Hex(0x0a0a0a), clip)
}

// clockView shows the time and date, or the recording timer; a tap on the
// timer drops a marker.
type clockView struct {
	ui.Base
	env     *Env
	last    string
	pressed bool
}

func (v *clockView) text() (big, small, sub string) {
	d := v.env.Desk
	if st := d.RecordState(); st == 1 || st == 2 {
		return recTime(d.RecordTime()), "", ""
	}
	if !d.ClockOnHome() {
		return "", "", ""
	}
	now := time.Now()
	main, suffix := clockText(now, d.Clock24h())
	return main, suffix, dateText(now, v.env.Text.German())
}

func (v *clockView) Dirty() []image.Rectangle {
	a, b, c := v.text()
	if cur := a + b + c; cur != v.last {
		v.last = cur
		v.MarkDirty()
	}
	return v.Base.Dirty()
}

func (v *clockView) Touch(phase ui.TouchPhase, p image.Point) bool {
	st := v.env.Desk.RecordState()
	if st != 1 && st != 2 {
		return false
	}
	switch phase {
	case ui.Down:
		v.pressed = true
	case ui.Up:
		if v.pressed {
			v.env.Desk.SetInt(desk.RecorderNode, 0, "requestDropMarker", time.Now().UnixMilli())
			v.env.Nav.Toast(v.env.T("MARKER ADDED AT") + " " + recTime(v.env.Desk.RecordTime()))
		}
		v.pressed = false
	}
	return true
}

func (v *clockView) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	r := v.Bounds()
	big, small, sub := v.text()
	if big == "" {
		return
	}
	if sub == "" { // the recording timer, large and alone
		ui.Text(dst, th.Fonts.Huge, big, r, font.Left, th.Text, clip)
		return
	}
	bw := ui.TextWidth(th.Fonts.Big, big)
	ui.Text(dst, th.Fonts.Big, big, image.Rect(r.Min.X, r.Min.Y+6, r.Min.X+bw+4, r.Min.Y+74), font.Left, th.Text, clip)
	if small != "" {
		ui.Text(dst, th.Fonts.Label, small, image.Rect(r.Min.X+bw+8, r.Min.Y+18, r.Max.X, r.Min.Y+74), font.Left, th.Text, clip)
	}
	ui.Text(dst, th.Fonts.Label, sub, image.Rect(r.Min.X+6, r.Min.Y+70, r.Max.X, r.Max.Y-6), font.Left, th.Text, clip)
}

// storageView is the storage indicator: an icon, a fill bar and the time
// left at the recorder's rate. Nothing is drawn without storage.
type storageView struct {
	ui.Base
	env  *Env
	last desk.Storage
}

func (v *storageView) Dirty() []image.Rectangle {
	if s := v.env.Desk.Storage(); s != v.last {
		v.last = s
		v.MarkDirty()
	}
	return v.Base.Dirty()
}

func (v *storageView) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	s := v.env.Desk.Storage()
	if !s.Present {
		return
	}
	r := v.Bounds()
	icon := "IconMicroSD"
	if s.USB {
		icon = "IconUsbStorage"
	}
	if s.Warning {
		icon = "IconStorageWarning"
	}
	c.Icon(dst, icon, image.Rect(r.Min.X, r.Min.Y, r.Min.X+34, r.Max.Y), th.Text)
	bar := image.Rect(r.Min.X+42, r.Min.Y+4, r.Max.X, r.Min.Y+14)
	ui.Panel(dst, bar, 5, gfx.Hex(0x3c3c3c), clip)
	w := int(float64(bar.Dx()) * s.Used)
	ui.Panel(dst, image.Rect(bar.Min.X, bar.Min.Y, bar.Min.X+max(w, 6), bar.Max.Y), 5, th.Accent, clip)
	ui.Text(dst, th.Fonts.Label, hoursText(s.Remaining), image.Rect(r.Min.X+42, r.Min.Y+16, r.Max.X, r.Max.Y), font.Left, th.Text, clip)
}

// monitorKnob is the speaker in a ring whose arc shows the monitor level.
// A vertical drag changes the level; a tap gives it the main knob's focus.
type monitorKnob struct {
	ui.Base
	env   *Env
	page  *HomePage
	last  string
	lastY int
	drag  bool
	moved bool
}

func (k *monitorKnob) state() string {
	d := k.env.Desk
	s := ""
	if d.MonitorMuted() {
		s += "m"
	}
	if d.MonitorAutoMuted() {
		s += "a"
	}
	if k.page.knobOn && k.page.knob == 0 {
		s += "f"
	}
	return s + pct(d.MonitorLevel())
}

func (k *monitorKnob) Dirty() []image.Rectangle {
	if s := k.state(); s != k.last {
		k.last = s
		k.MarkDirty()
	}
	return k.Base.Dirty()
}

func (k *monitorKnob) Touch(phase ui.TouchPhase, p image.Point) bool {
	switch phase {
	case ui.Down:
		k.drag, k.moved, k.lastY = true, false, p.Y
	case ui.Move:
		if k.drag {
			if dy := k.lastY - p.Y; dy != 0 {
				k.moved = true
				k.env.Desk.SetMonitorLevel(k.env.Desk.MonitorLevel() + float64(dy)/200)
				k.lastY = p.Y
			}
		}
	case ui.Up:
		if k.drag && !k.moved {
			k.page.knob, k.page.knobOn = 0, true
			k.env.Nav.Toast(k.env.T("MONITOR") + ": " + pct(k.env.Desk.MonitorLevel()))
		}
		k.drag = false
	}
	return true
}

func (k *monitorKnob) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	r := ui.Square(k.Bounds())
	cx, cy := float64(r.Min.X+r.Max.X)/2, float64(r.Min.Y+r.Max.Y)/2
	rad := float64(r.Dx()) / 2
	d := k.env.Desk
	ui.Circle(dst, cx, cy, rad-4, gfx.Hex(0x101010), clip)
	ring := th.Accent
	switch {
	case d.MonitorMuted():
		ring = th.Record
	case d.MonitorAutoMuted():
		ring = th.Warn
	}
	// The track and the level arc, from the bottom around.
	const start = math.Pi * 0.5
	ui.Arc(dst, cx, cy, rad-6, 6, start, start+2*math.Pi-0.001, gfx.Hex(0x3a3a3a), clip)
	if lv := d.MonitorLevel(); lv > 0 {
		ui.Arc(dst, cx, cy, rad-6, 6, start, start+2*math.Pi*lv, ring, clip)
	}
	ic := image.Rect(int(cx)-18, int(cy)-18, int(cx)+18, int(cy)+18)
	c.Icon(dst, "IconSpeaker", ic, th.Text)
	if d.MonitorMuted() || d.MonitorAutoMuted() {
		cross := &gfx.Path{}
		cross.MoveTo(cx-16, cy-16)
		cross.LineTo(cx+16, cy+16)
		cross.MoveTo(cx+16, cy-16)
		cross.LineTo(cx-16, cy+16)
		dst.StrokePath(cross, gfx.Identity, gfx.Stroke{Width: 5, Cap: gfx.RoundCap}, gfx.Solid(ring), clip)
	}
	if k.page.knobOn && k.page.knob == 0 {
		ui.Ring(dst, cx, cy, rad-1, 3, th.Text, clip)
	}
}

// masterMeter is the L/R output meter with its "60 45 30 15 0" scale.
type masterMeter struct {
	ui.Base
	env *Env
	l   *ui.Meter
	r   *ui.Meter
}

func (m *masterMeter) SetBounds(r image.Rectangle) {
	m.Base.SetBounds(r)
	if m.l == nil {
		d := m.env.Desk
		m.l = ui.NewMeter(func() float64 { l, _ := d.MasterLR(); return desk.DbFrac(l) })
		m.l.Peak = func() float64 { l, _ := d.MasterPeakLR(); return desk.DbFrac(l) }
		m.r = ui.NewMeter(func() float64 { _, rr := d.MasterLR(); return desk.DbFrac(rr) })
		m.r.Peak = func() float64 { _, rr := d.MasterPeakLR(); return desk.DbFrac(rr) }
		m.l.Vertical, m.r.Vertical = false, false
	}
	m.l.SetBounds(image.Rect(r.Min.X+30, r.Min.Y+8, r.Max.X-24, r.Min.Y+26))
	m.r.SetBounds(image.Rect(r.Min.X+30, r.Min.Y+34, r.Max.X-24, r.Min.Y+52))
}

func (m *masterMeter) Dirty() []image.Rectangle {
	out := m.Base.Dirty()
	out = append(out, m.l.Dirty()...)
	out = append(out, m.r.Dirty()...)
	return out
}

func (m *masterMeter) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	r := m.Bounds()
	ui.Text(dst, th.Fonts.Label, "L", image.Rect(r.Min.X, r.Min.Y+2, r.Min.X+26, r.Min.Y+32), font.Left, th.Text, clip)
	ui.Text(dst, th.Fonts.Label, "R", image.Rect(r.Min.X, r.Min.Y+28, r.Min.X+26, r.Min.Y+58), font.Left, th.Text, clip)
	m.l.Draw(dst, c, clip)
	m.r.Draw(dst, c, clip)
	// The scale under the bars: 60 … 0 left to right.
	bar := m.l.Bounds()
	for db := 60; db >= 0; db -= 15 {
		x := bar.Min.X + int(float64(60-db)/60*float64(bar.Dx()))
		ui.Text(dst, th.Fonts.Tiny, intText(int64(db)), image.Rect(x-20, r.Min.Y+54, x+20, r.Max.Y), font.Center, gfx.Hex(0x8a8a8a), clip)
	}
	for db := 60; db >= 0; db -= 5 {
		x := bar.Min.X + int(float64(60-db)/60*float64(bar.Dx()))
		ui.Fill(dst, image.Rect(x, r.Min.Y+55, x+1, r.Min.Y+59), gfx.Hex(0x8a8a8a), clip)
	}
}

// --- strips ---

// stripView is one channel strip: its scale, meter(s), fader cap, source
// icon and colour bar. Tapping the icon opens the channel page; tapping the
// meter opens the strip's bar; dragging a virtual strip moves its fader.
type stripView struct {
	ui.Base
	env      *Env
	page     *HomePage
	strip    int
	virtual  bool
	lastPhys bool // the last physical strip (a brighter divider after it)

	last    string
	down    image.Point
	dragged bool
	dragY   int
	shown   [2]float64 // the levels as drawn: they fall smoothly between readings
	shownAt time.Time
}

// meterFall is how fast a drawn level falls after the reading dropped, in
// scale fractions per second (the 60 dB scale: 20 dB/s), so the 20 Hz
// readings from the console look like the device's own meters.
const meterFall = 1.0 / 3

// levels returns the levels to draw: the readings when they rose, a smooth
// fall when they dropped.
func (s *stripView) levels() (l, r float64) {
	lv, rv := s.env.Desk.MeterLR(s.strip)
	tl, tr := desk.DbFrac(lv), desk.DbFrac(rv)
	now := time.Now()
	dt := now.Sub(s.shownAt).Seconds()
	if dt > 0.5 {
		dt = 0.5
	}
	s.shownAt = now
	fall := meterFall * dt
	s.shown[0] = math.Max(tl, s.shown[0]-fall)
	s.shown[1] = math.Max(tr, s.shown[1]-fall)
	return s.shown[0], s.shown[1]
}

func (s *stripView) icons() image.Rectangle {
	r := s.Bounds()
	return image.Rect(r.Min.X, meterBase+8, r.Max.X, r.Max.Y)
}

// bars is where the meter column(s) sit.
func (s *stripView) bars() (l, r image.Rectangle, stereo bool) {
	b := s.Bounds()
	stereo = s.env.Desk.MeterStereo(s.strip)
	// The scale takes 34 px at the left; the bars are centred in the rest.
	cx := (b.Min.X + 38 + b.Max.X - 4) / 2
	if stereo {
		l = image.Rect(cx-17, meterTop, cx-3, meterBase)
		r = image.Rect(cx+3, meterTop, cx+17, meterBase)
		return
	}
	l = image.Rect(cx-11, meterTop, cx+11, meterBase)
	return l, l, false
}

func (s *stripView) state() string {
	d := s.env.Desk
	l, r := s.levels()
	pl, pr := d.MeterPeakLR(s.strip)
	focus := s.page.knobOn && s.virtual && s.page.knob == s.strip-d.PhysicalFaders()+1
	return intText(d.SourceOf(s.strip)) + "|" + d.InputColour(s.strip) + "|" + intText(d.FaderLevel(s.strip)) +
		"|" + fine(l) + fine(r) + pct(pl) + pct(pr) + "|" + boolText(d.Muted(s.strip)) + boolText(d.Cue(s.strip)) +
		boolText(focus) + boolText(d.BroadcastMeters()) + pct(d.GainReductionDb(s.strip)/24)
}

// fine is a level rounded to the meter's pixel (for change detection).
func fine(v float64) string { return intText(int64(v*float64(meterBase-meterTop))) + "," }

func boolText(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func (s *stripView) Dirty() []image.Rectangle {
	if st := s.state(); st != s.last {
		s.last = st
		s.MarkDirty()
	}
	return s.Base.Dirty()
}

func (s *stripView) Touch(phase ui.TouchPhase, p image.Point) bool {
	switch phase {
	case ui.Down:
		s.down, s.dragged, s.dragY = p, false, p.Y
	case ui.Move:
		if s.virtual && !p.In(s.icons()) && (s.dragged || abs(p.Y-s.down.Y) > 8) {
			s.dragged = true
			s.env.Desk.SetFaderFrac(s.strip, 1-float64(p.Y-meterTop)/float64(meterBase-meterTop))
			s.page.knob, s.page.knobOn = s.strip-s.env.Desk.PhysicalFaders()+1, true
		}
	case ui.Up:
		if s.dragged {
			return true
		}
		if p.In(s.icons()) {
			s.env.Nav.Push(NewChannel(s.strip))
			return true
		}
		if s.page.selected == s.strip {
			s.page.selected = -1
			s.env.Nav.Refresh()
			return true
		}
		s.page.selectStrip(s.env, s.strip)
	}
	return true
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (s *stripView) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	d := s.env.Desk
	b := s.Bounds()
	// Dividers.
	ui.Fill(dst, image.Rect(b.Max.X-1, b.Min.Y+8, b.Max.X, b.Max.Y), gfx.Hex(0x262626), clip)
	if s.lastPhys {
		ui.Fill(dst, image.Rect(b.Max.X-2, b.Min.Y+8, b.Max.X, b.Max.Y), gfx.Hex(0x5a5a5a), clip)
	}
	l, r, stereo := s.bars()
	// The dBFS scale: labels left of the bars, every 15 dB; dots every 3.
	ui.MeterScale(dst, l, l.Min.X-6, c, clip)
	// Meter columns with level and peak.
	lv, rv := s.shown[0], s.shown[1]
	pl, pr := d.MeterPeakLR(s.strip)
	s.column(dst, l, lv, desk.DbFrac(pl), c, clip)
	if stereo {
		s.column(dst, r, rv, desk.DbFrac(pr), c, clip)
	}
	// Gain reduction in Broadcast metering: a thin amber bar right of the strip.
	if d.BroadcastMeters() {
		gr := math.Min(d.GainReductionDb(s.strip), 24) / 24
		gx := r.Max.X + 6
		ui.Fill(dst, image.Rect(gx, meterTop, gx+5, meterBase), gfx.Hex(0x1e1e1e), clip)
		if gr > 0 {
			ui.Fill(dst, image.Rect(gx, meterTop, gx+5, meterTop+int(gr*float64(meterBase-meterTop))), th.Warn, clip)
		}
	}
	// The fader cap over the bars.
	frac := d.FaderFrac(s.strip)
	cy := meterBase - int(frac*float64(meterBase-meterTop))
	capR := image.Rect(l.Min.X-12, cy-11, max(r.Max.X, l.Max.X)+12, cy+11)
	focus := s.page.knobOn && s.virtual && s.page.knob == s.strip-d.PhysicalFaders()+1
	capCol := th.Text
	if focus {
		capCol = th.Accent
	}
	if d.Muted(s.strip) {
		capCol = th.Record
	}
	if frac <= 0 {
		ui.Panel(dst, capR, 11, gfx.Hex(0x101010), clip)
		ui.PanelOutline(dst, capR, 11, 3, capCol, clip)
	} else {
		ui.Panel(dst, capR, 11, capCol, clip)
	}
	// Unity marks beside the cap when the fader sits at unity.
	if unity := d.UnityFrac(); math.Abs(frac-unity) < 0.01 {
		tri := func(x0, dir int) {
			p := &gfx.Path{}
			p.MoveTo(float64(x0), float64(cy-7))
			p.LineTo(float64(x0+dir*10), float64(cy))
			p.LineTo(float64(x0), float64(cy+7))
			p.Close()
			dst.Fill(p, gfx.Identity, gfx.NonZero, gfx.Solid(th.Text), clip)
		}
		tri(capR.Min.X-3, -1)
		if capR.Max.X+13 < b.Max.X {
			tri(capR.Max.X+3, 1)
		}
	}
	// Source icon and colour bar.
	ic := s.icons()
	src := d.SourceOf(s.strip)
	name := d.SourceName(src)
	iconCol := th.Text
	if d.Cue(s.strip) {
		iconCol = th.Accent
	}
	icon := sourceIcon(name.ID)
	if src < 0 {
		icon = "IconNotAssigned"
		iconCol = th.TextMuted
	}
	iw := min(ic.Dx()-16, 62)
	ir := image.Rect((ic.Min.X+ic.Max.X)/2-iw/2, ic.Min.Y+2, (ic.Min.X+ic.Max.X)/2+iw/2, ic.Min.Y+2+58)
	if icon != "" {
		c.Icon(dst, icon, ir, iconCol)
	} else {
		ui.Text(dst, th.Fonts.Small, name.Label, ir, font.Center, iconCol, clip)
	}
	if col := d.InputColour(s.strip); col != "" && src >= 0 {
		bw := min(ic.Dx()-24, 64)
		bx := (ic.Min.X + ic.Max.X) / 2
		ui.Panel(dst, image.Rect(bx-bw/2, ic.Max.Y-24, bx+bw/2, ic.Max.Y-12), 6, parseColour(col, th.Accent), clip)
	}
	if d.Muted(s.strip) {
		ui.Panel(dst, image.Rect(ir.Min.X-4, ir.Min.Y-2, ir.Max.X+4, ir.Max.Y+2), 8, th.Record.WithAlpha(0.35), clip)
	}
}

// column draws one meter column with its level fill and peak line.
func (s *stripView) column(dst *gfx.Image, r image.Rectangle, level, peak float64, c *ui.Context, clip image.Rectangle) {
	ui.Panel(dst, r, 4, gfx.Hex(0x141414), clip)
	if level > 0 {
		top := r.Max.Y - int(level*float64(r.Dy()))
		col := gfx.Hex(0x2fd12f)
		switch {
		case level >= 0.95:
			col = gfx.Hex(0xe82020)
		case level >= 0.85:
			col = gfx.Hex(0xf0d020)
		}
		ui.Panel(dst, image.Rect(r.Min.X, top, r.Max.X, r.Max.Y), 4, col, clip)
	}
	if peak > level && peak > 0 {
		py := r.Max.Y - int(peak*float64(r.Dy()))
		ui.Fill(dst, image.Rect(r.Min.X, py-2, r.Max.X, py+1), gfx.Hex(0x2fd12f), clip)
	}
}

// stripBar is the listen / channel / mute bar under the selected strip.
func (p *HomePage) stripBar(env *Env, area image.Rectangle, w, n int) []ui.View {
	th := env.Ctx.Theme
	strip := p.selected
	cx := area.Min.X + strip*w + w/2
	bw, bh := 262, 92
	x0 := min(max(cx-bw/2, area.Min.X+6), area.Max.X-bw-6)
	r := image.Rect(x0, area.Max.Y-bh-6, x0+bw, area.Max.Y-6)
	card := ui.NewPanelView(gfx.Hex(0x1c1c1c))
	card.Outline = th.Line
	card.SetBounds(r)
	cells := columns(ui.Inset(r, 8), 3, 8)
	listen := ui.NewButton("", func() { env.Desk.SetCue(strip, !env.Desk.Cue(strip)); p.barUntil = time.Now().Add(4 * time.Second) })
	listen.Icon, listen.Toggle = "IconListen", true
	listen.SetBounds(cells[0])
	ch := ui.NewButton("", func() { env.Nav.Push(NewChannel(strip)) })
	ch.Icon = sourceIcon(env.Desk.SourceName(env.Desk.SourceOf(strip)).ID)
	if ch.Icon == "" {
		ch.Icon = "IconNotAssigned"
	}
	ch.SetBounds(cells[1])
	mute := ui.NewButton("", func() { env.Desk.SetMute(strip, !env.Desk.Muted(strip)); p.barUntil = time.Now().Add(4 * time.Second) })
	mute.Icon, mute.Toggle = "IconMute", true
	mute.SetBounds(cells[2])
	return []ui.View{card,
		&updater{get: func() { listen.SetOn(env.Desk.Cue(strip)); mute.SetOn(env.Desk.Muted(strip)) }},
		listen, ch, mute}
}

// --- the pads ---

func (p *HomePage) pads(env *Env, r image.Rectangle) []ui.View {
	per := env.Desk.PadsPerBank()
	cols := 2
	rows := (per + cols - 1) / cols
	cells := grid(r, cols, rows, 14)
	var out []ui.View
	for slot := 0; slot < per && slot < len(cells); slot++ {
		t := &padTile{env: env, slot: slot, compact: true}
		t.SetBounds(cells[slot])
		out = append(out, t)
	}
	return out
}

// --- the main knob on the mixer ---

// Encoder drives the monitor level or the focused virtual fader; a press
// moves the focus on (monitor → virtual strips → monitor); a long press
// mutes the focused strip (or the monitor).
func (p *HomePage) Encoder(env *Env, ev desk.EncoderEvent) bool {
	d := env.Desk
	physical := d.PhysicalFaders()
	nVirtual := max(0, d.StripCount()-physical)
	if p.knob > nVirtual {
		p.knob = 0
	}
	p.knobOn = true
	switch {
	case ev.Delta != 0:
		if p.knob == 0 {
			d.SetMonitorLevel(d.MonitorLevel() + float64(ev.Delta)*0.01)
			return true
		}
		strip := physical + p.knob - 1
		d.SetFaderFrac(strip, d.FaderFrac(strip)+float64(ev.Delta)/127)
		p.selected, p.barUntil = strip, time.Now().Add(4*time.Second)
		return true
	case ev.Release && ev.Held >= 900*time.Millisecond:
		if p.knob == 0 {
			d.SetMonitorMute(!d.MonitorMuted())
		} else {
			strip := physical + p.knob - 1
			d.SetMute(strip, !d.Muted(strip))
		}
		return true
	case ev.Release:
		p.knob = (p.knob + 1) % (nVirtual + 1)
		if p.knob == 0 {
			p.selected = -1
		} else {
			p.selected, p.barUntil = physical+p.knob-1, time.Now().Add(4*time.Second)
		}
		env.Nav.Refresh()
		return true
	}
	return true
}
