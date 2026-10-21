package pages

import (
	"image"
	"math"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// The sound pad editor (PadsEditSoundPage): the pad's name in the header
// with back and home; the sound's envelope box with four round handles
// (trim at the bottom corners, fades at the top) and the playhead; the
// mode bar with the three ‹ value › carousels (play mode, loop,
// replay/continue); the trim level; the red record button and the play
// button. A pad without a sound shows RØDE's hint instead of the box.
// While the pad recorder records this pad the box gives way to the record
// view: the time, pause and stop.

// Play modes (padPlayMode), as RØDE's mode selector writes them (1.7.6,
// PadsEditSoundPage::ModeSelector at 0x7e3960): 0 Toggle, 1 Hold, 2 One
// Shot. Hold and One Shot also switch the loop off and replay on.
const (
	playToggle  = 0
	playHold    = 1
	playOneShot = 2
)

// Pad recorder states (PADRECORDER.padRecordState) and requests, as the
// sound page writes them (0x7df5f0): 2 records, 1 pauses, 0 stops; the
// pause button sends 2 when the state is 1 (paused) and 1 otherwise.
const (
	padRecIdle      = 0
	padRecPaused    = 1
	padRecRecording = 2
)

type padSoundPage struct {
	padPage
	savedMode int64 // the play mode before a preview (-1: none)
}

func newPadSoundPage(idx int) Page { return &padSoundPage{padPage{idx}, -1} }

func (p *padSoundPage) Title() string { return "" }

func (p *padSoundPage) Header(env *Env, bar image.Rectangle) ([]ui.View, bool) {
	return titleHeader(env, bar, p.name(env)), false
}

func (p *padSoundPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	d := env.Desk
	pad := d.PadByIndex(p.idx)
	node := pad.Node
	if node < 0 {
		return c
	}
	num := func(prop string) float64 { return d.Num(desk.PadNode, node, prop) }
	set := func(prop string, v float64) { d.SetFloat(desk.PadNode, node, prop, v) }
	hasSound := func() bool { return d.Str(desk.PadNode, node, "padFilePath") != "" }
	rec := d.PadRecorder()
	recording := rec.Idx == int64(p.idx) && rec.State != padRecIdle

	// The envelope box, or the record view, or RØDE's hint.
	boxR := image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+290)
	switch {
	case recording:
		c.Add(p.recordView(env, boxR)...)
	case hasSound():
		ev := &envelopeView{get: num, set: set, progress: func() float64 { return num("padProgress") },
			seek: func(f float64) { d.PadSeek(node, f) }}
		ev.SetBounds(boxR)
		c.Add(ev)
	default:
		card := ui.NewPanelView(th.Black)
		card.SetBounds(boxR)
		hint := ui.NewParagraph(env.T("RECORD A SOUND OR ADD VIA THE RØDE CENTRAL APPLICATION"), th.Fonts.Body, th.TextDim)
		hint.Align = font.Center
		hint.SetBounds(image.Rect(boxR.Min.X+100, boxR.Min.Y+110, boxR.Max.X-100, boxR.Max.Y-80))
		c.Add(card, hint)
	}
	// The recorder's state changes the page: watch it.
	c.Add(&updater{get: func() {
		now := d.PadRecorder()
		if (now.Idx == int64(p.idx) && now.State != padRecIdle) != recording {
			env.Nav.Refresh()
		}
	}})

	// The mode bar: play mode, loop, replay/continue as ‹ value › carousels.
	bar := image.Rect(r.Min.X, boxR.Max.Y+20, r.Max.X, boxR.Max.Y+20+96)
	cells := columns(bar, 3, 20)
	playKeys := map[int64]string{playToggle: "TOGGLE", playHold: "HOLD", playOneShot: "ONE SHOT"}
	playIcons := []string{"iconToggle", "iconHold", "icon1shot"}
	mode := func() int64 {
		m := d.Int(desk.PadNode, node, "padPlayMode")
		if m < 0 || m > 2 {
			return playToggle
		}
		return m
	}
	setMode := func(m int64) {
		d.SetInt(desk.PadNode, node, "padPlayMode", m)
		if m != playToggle {
			// RØDE's selector switches the loop off and replay on with Hold
			// and One Shot (loop and continue belong to Toggle).
			d.SetBool(desk.PadNode, node, "padLoop", false)
			d.SetBool(desk.PadNode, node, "padReplay", true)
		}
	}
	c.Add(carousel(env, cells[0], playIcons, func() string { return env.T(playKeys[mode()]) },
		func() int { return int(mode()) },
		func(delta int) { setMode(((mode()+int64(delta))%3 + 3) % 3) })...)
	loopOK := func() bool { return mode() == playToggle }
	c.Add(carousel(env, cells[1], []string{"iconLoopOff", "iconLoopOn"}, func() string {
		if d.Bool(desk.PadNode, node, "padLoop") {
			return env.T("LOOP ON")
		}
		return env.T("LOOP OFF")
	}, func() int {
		if d.Bool(desk.PadNode, node, "padLoop") {
			return 1
		}
		return 0
	}, func(int) {
		if loopOK() {
			d.SetBool(desk.PadNode, node, "padLoop", !d.Bool(desk.PadNode, node, "padLoop"))
		}
	})...)
	c.Add(carousel(env, cells[2], []string{"iconContinue", "iconReplay"}, func() string {
		if d.Bool(desk.PadNode, node, "padReplay") {
			return env.T("REPLAY")
		}
		return env.T("CONTINUE")
	}, func() int {
		if d.Bool(desk.PadNode, node, "padReplay") {
			return 1
		}
		return 0
	}, func(int) {
		if loopOK() {
			d.SetBool(desk.PadNode, node, "padReplay", !d.Bool(desk.PadNode, node, "padReplay"))
		}
	})...)

	// Trim (the pad's level), the record button and the play button.
	bottom := image.Rect(r.Min.X, bar.Max.Y+20, r.Max.X, r.Max.Y)
	card := ui.NewCard()
	card.SetBounds(image.Rect(bottom.Min.X, bottom.Min.Y, bottom.Max.X-280, bottom.Max.Y))
	c.Add(card)
	sl := ui.NewSlider(func() float64 { return (num("padGain") + 60) / 60 },
		func(v float64) { set("padGain", math.Round(-60+v*60)) })
	sl.Label = env.T("TRIM")
	sl.Text = func(v float64) string { return dbText(-60 + v*60) }
	sl.Unity = 48.0 / 60 // RØDE's default of −12 dB
	sl.Step = 1.0 / 60
	sl.SetBounds(ui.Inset(card.Bounds(), 16))
	c.Add(sl)
	recR := centred(image.Rect(bottom.Max.X-260, bottom.Min.Y, bottom.Max.X-140, bottom.Max.Y), 100, 100)
	recBtn := &recordButton{tap: func() { p.record(env, node, hasSound()) }}
	recBtn.SetBounds(recR)
	c.Add(recBtn)
	play := ui.NewButton("", func() { p.preview(env, node) })
	play.Icon = "IconPlay"
	play.Style = ui.StyleRound
	play.SetBounds(centred(image.Rect(bottom.Max.X-120, bottom.Min.Y, bottom.Max.X, bottom.Max.Y), 100, 100))
	c.Add(&updater{get: func() {
		if ic := "IconPlay"; d.Bool(desk.PadNode, node, "padActive") {
			ic = "IconRecStop"
			if play.Icon != ic {
				play.Icon = ic
				play.MarkDirty()
			}
		} else if play.Icon != ic {
			play.Icon = ic
			play.MarkDirty()
		}
	}}, play)
	return c
}

// record starts recording into the pad; with a sound loaded RØDE asks
// first, since the recording replaces it.
func (p *padSoundPage) record(env *Env, node int, hasSound bool) {
	start := func() { env.Desk.PadRecordRequest(int64(p.idx), padRecRecording) }
	if !hasSound {
		start()
		return
	}
	env.Nav.Confirm("", env.T("RECORDING A NEW SOUND WILL OVERWRITE YOUR CURRENT LOADED SOUND."), env.T("OVERWRITE"), true, start)
}

// preview plays or stops the pad from the editor as RØDE's play button
// does: the play mode is set to Toggle for the preview (and put back when
// it stops), then the pad's button is pressed.
func (p *padSoundPage) preview(env *Env, node int) {
	d := env.Desk
	bank, slot := p.idx/d.PadsPerBank(), p.idx%d.PadsPerBank()
	if d.Bool(desk.PadNode, node, "padActive") {
		d.PreviewPad(bank, slot)
		if p.savedMode >= 0 {
			d.SetInt(desk.PadNode, node, "padPlayMode", p.savedMode)
			p.savedMode = -1
		}
		return
	}
	p.savedMode = d.Int(desk.PadNode, node, "padPlayMode")
	d.SetInt(desk.PadNode, node, "padPlayMode", playToggle)
	d.PreviewPad(bank, slot)
}

// recordView is the box while this pad records: the time, pause/resume
// and stop.
func (p *padSoundPage) recordView(env *Env, r image.Rectangle) []ui.View {
	th := env.Ctx.Theme
	d := env.Desk
	card := ui.NewPanelView(th.Black)
	card.SetBounds(r)
	tm := ui.NewLabel("", th.Fonts.Huge, th.Record, font.Center)
	tm.SetBounds(image.Rect(r.Min.X, r.Min.Y+40, r.Max.X, r.Min.Y+140))
	state := ui.NewLabel("", th.Fonts.Body, th.TextDim, font.Center)
	state.SetBounds(image.Rect(r.Min.X, r.Min.Y+140, r.Max.X, r.Min.Y+180))
	pause := ui.NewButton("", func() {
		rec := d.PadRecorder()
		if rec.State == padRecPaused {
			d.PadRecordRequest(int64(p.idx), padRecRecording)
		} else {
			d.PadRecordRequest(int64(p.idx), padRecPaused)
		}
	})
	pause.Icon = "IconRecPause"
	pause.Style = ui.StyleRound
	pause.SetBounds(centred(image.Rect(r.Min.X, r.Max.Y-110, (r.Min.X+r.Max.X)/2, r.Max.Y-10), 90, 90))
	stop := ui.NewButton("", func() { d.PadRecordRequest(int64(p.idx), padRecIdle) })
	stop.Icon = "IconRecStop"
	stop.Style = ui.StyleRound
	stop.SetBounds(centred(image.Rect((r.Min.X+r.Max.X)/2, r.Max.Y-110, r.Max.X, r.Max.Y-10), 90, 90))
	upd := &updater{get: func() {
		rec := d.PadRecorder()
		tm.Set(timeText(rec.Seconds))
		switch {
		case rec.Full:
			state.Set(env.T("STORAGE FULL"))
		case rec.State == padRecPaused:
			state.Set(env.T("PAUSED"))
		default:
			state.Set(env.T("RECORDING"))
		}
		if ic := "IconRecPause"; rec.State == padRecPaused {
			ic = "IconRec"
			if pause.Icon != ic {
				pause.Icon = ic
				pause.MarkDirty()
			}
		} else if pause.Icon != ic {
			pause.Icon = ic
			pause.MarkDirty()
		}
	}}
	return []ui.View{card, upd, tm, state, pause, stop}
}

// recordButton is the big red record button.
type recordButton struct {
	ui.Base
	tap     func()
	pressed bool
}

func (b *recordButton) CanFocus() bool { return true }
func (b *recordButton) Activate() bool {
	b.tap()
	return false
}
func (b *recordButton) Touch(phase ui.TouchPhase, p image.Point) bool {
	switch phase {
	case ui.Down:
		b.pressed = true
	case ui.Up:
		if b.pressed {
			b.tap()
		}
		b.pressed = false
	}
	b.MarkDirty()
	return true
}
func (b *recordButton) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	r := ui.Square(b.Bounds())
	cx, cy := float64(r.Min.X+r.Max.X)/2, float64(r.Min.Y+r.Max.Y)/2
	fill := th.CardTop
	if b.pressed {
		fill = gfx.Mix(fill, th.Text, 0.2)
	}
	ui.Circle(dst, cx, cy, float64(r.Dx())/2, fill, clip)
	recDot(dst, centred(r, r.Dx()/2, r.Dy()/2), clip)
	if b.Focused() {
		ui.Focus(dst, r, r.Dx()/2, false, th, clip)
	}
}

// carousel is a ‹ icon value › row in a card: arrows step the value.
func carousel(env *Env, r image.Rectangle, icons []string, text func() string, sel func() int, step func(delta int)) []ui.View {
	th := env.Ctx.Theme
	card := ui.NewCard()
	card.SetBounds(r)
	prev := chevron(env, false, true, func() { step(-1); env.Nav.Refresh() })
	prev.Font = th.Fonts.Big
	prev.SetBounds(image.Rect(r.Min.X, r.Min.Y, r.Min.X+64, r.Max.Y))
	next := chevron(env, true, true, func() { step(1); env.Nav.Refresh() })
	next.Font = th.Fonts.Big
	next.SetBounds(image.Rect(r.Max.X-64, r.Min.Y, r.Max.X, r.Max.Y))
	icon := ui.NewIcon("", th.Text)
	icon.SetBounds(image.Rect(r.Min.X+72, r.Min.Y+20, r.Min.X+72+56, r.Max.Y-20))
	lbl := ui.NewLabel("", th.Fonts.Body, th.Text, font.Center)
	lbl.SetBounds(image.Rect(r.Min.X+136, r.Min.Y, r.Max.X-64, r.Max.Y))
	upd := &updater{get: func() {
		lbl.Set(text())
		if i := sel(); i >= 0 && i < len(icons) && icon.Name != icons[i] {
			icon.Name = icons[i]
			icon.MarkDirty()
		}
	}}
	return []ui.View{card, prev, upd, icon, lbl, next}
}

// envelopeView draws a sound's play region with four round handles: the
// bottom ones trim the start and the end, the top ones end the fade-in and
// start the fade-out. Dragging a handle writes its fraction; a tap in the
// box asks the pad to jump there (the scrubber), and the playhead follows
// the pad's progress.
type envelopeView struct {
	ui.Base
	get      func(prop string) float64
	set      func(prop string, v float64)
	progress func() float64
	seek     func(frac float64)
	drag     string // the handle being dragged
	downAt   image.Point
	last     [5]float64
}

var envProps = [4]string{"padEnvStart", "padEnvFadeIn", "padEnvFadeOut", "padEnvStop"}

func (e *envelopeView) box() image.Rectangle { return ui.Inset(e.Bounds(), 30) }

func (e *envelopeView) xOf(v float64) int {
	b := e.box()
	return b.Min.X + int(clamp01(v)*float64(b.Dx()))
}

func (e *envelopeView) handles() map[string]image.Point {
	b := e.box()
	return map[string]image.Point{
		"padEnvStart":   {e.xOf(e.get("padEnvStart")), b.Max.Y},
		"padEnvStop":    {e.xOf(e.get("padEnvStop")), b.Max.Y},
		"padEnvFadeIn":  {e.xOf(e.get("padEnvFadeIn")), b.Min.Y},
		"padEnvFadeOut": {e.xOf(e.get("padEnvFadeOut")), b.Min.Y},
	}
}

func (e *envelopeView) Touch(phase ui.TouchPhase, p image.Point) bool {
	b := e.box()
	switch phase {
	case ui.Down:
		best, bestD := "", 1<<30
		for name, h := range e.handles() {
			dx, dy := p.X-h.X, p.Y-h.Y
			if d := dx*dx + dy*dy; d < bestD && d < 70*70 {
				best, bestD = name, d
			}
		}
		e.drag, e.downAt = best, p
	case ui.Move:
		if e.drag == "" {
			return true
		}
		v := clamp01(float64(p.X-b.Min.X) / float64(b.Dx()))
		// Keep start ≤ fade-in ≤ fade-out ≤ stop.
		switch e.drag {
		case "padEnvStart":
			v = min(v, e.get("padEnvFadeIn"))
		case "padEnvFadeIn":
			v = max(e.get("padEnvStart"), min(v, e.get("padEnvFadeOut")))
		case "padEnvFadeOut":
			v = max(e.get("padEnvFadeIn"), min(v, e.get("padEnvStop")))
		case "padEnvStop":
			v = max(v, e.get("padEnvFadeOut"))
		}
		e.set(e.drag, float64(int(v*1000+0.5))/1000)
		e.MarkDirty()
	case ui.Up:
		if e.drag == "" && e.seek != nil && p.In(b) && abs(p.X-e.downAt.X) < 12 {
			e.seek(clamp01(float64(p.X-b.Min.X) / float64(b.Dx())))
		}
		e.drag = ""
	}
	return true
}

func (e *envelopeView) Dirty() []image.Rectangle {
	var cur [5]float64
	for i, p := range envProps {
		cur[i] = e.get(p)
	}
	if e.progress != nil {
		cur[4] = e.progress()
	}
	if cur != e.last {
		e.last = cur
		e.MarkDirty()
	}
	return e.Base.Dirty()
}

func (e *envelopeView) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	r := e.Bounds()
	ui.Panel(dst, r, th.Radius, th.Black, clip)
	b := e.box()
	// A stylised waveform (the file itself stays with the device).
	mid := (b.Min.Y + b.Max.Y) / 2
	for x := b.Min.X; x < b.Max.X; x += 4 {
		k := float64(x-b.Min.X) / float64(b.Dx())
		h := float64(b.Dy()/2) * (0.25 + 0.55*absf(sinf(k*41))*(1-0.6*k)*(0.6+0.4*absf(sinf(k*7))))
		col := th.TextMuted
		if k >= e.get("padEnvStart") && k <= e.get("padEnvStop") {
			col = th.Accent
		}
		ui.Fill(dst, image.Rect(x, mid-int(h), x+3, mid+int(h)), col, clip)
	}
	// The envelope line: start → fade-in top → fade-out top → stop.
	s, fi, fo, st := e.xOf(e.get("padEnvStart")), e.xOf(e.get("padEnvFadeIn")), e.xOf(e.get("padEnvFadeOut")), e.xOf(e.get("padEnvStop"))
	p := &gfx.Path{}
	p.MoveTo(float64(s), float64(b.Max.Y))
	p.LineTo(float64(fi), float64(b.Min.Y))
	p.LineTo(float64(fo), float64(b.Min.Y))
	p.LineTo(float64(st), float64(b.Max.Y))
	dst.StrokePath(p, gfx.Identity, gfx.Stroke{Width: 3, Join: gfx.RoundJoin}, gfx.Solid(th.Text), clip)
	// The playhead.
	if e.progress != nil {
		if k := e.progress(); k > 0 {
			x := e.xOf(k)
			ui.Fill(dst, image.Rect(x-1, b.Min.Y, x+2, b.Max.Y), th.Text, clip)
		}
	}
	for name, h := range e.handles() {
		col := th.Text
		if name == e.drag {
			col = th.Accent
		}
		ui.Circle(dst, float64(h.X), float64(h.Y), 16, col, clip)
		ui.Circle(dst, float64(h.X), float64(h.Y), 8, th.Black, clip)
	}
}

func absf(v float64) float64    { return math.Abs(v) }
func clamp01(v float64) float64 { return math.Max(0, math.Min(1, v)) }
func sinf(v float64) float64    { return math.Sin(v) }
