package pages

import (
	"image"
	"strconv"
	"time"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// The recordings list (RecordingsPage): one row per recording the desk
// found on its storage — name, date, length, Multitrack or Stereo — that
// scrolls by touch or the knob; a tap opens the player. The podcast player
// (PodcastPlayerPage): the recording's name and date in the header, the
// teal time display with the MULTI/STEREO marks, the transport row
// (scissors, previous marker, back 15 s, play/pause, forward 30 s, next
// marker, bin), the waveform with the playhead and the scrubber; the
// scissors open the edit mode with the four envelope handles.

func init() {
	register(IDRecordings, func() Page { return &recordingsPage{} })
}

type recordingsPage struct{}

func (p *recordingsPage) Title() string { return "RECORDINGS" }

func (p *recordingsPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	d := env.Desk
	recs := d.Recordings()
	if len(recs) == 0 {
		hint := ui.NewParagraph(env.T("TO VIEW ALL RECORDINGS, CONNECT TO RØDE CENTRAL"), th.Fonts.Body, th.TextDim)
		hint.Align = font.Center
		hint.SetBounds(image.Rect(r.Min.X+100, r.Min.Y+r.Dy()/2-40, r.Max.X-100, r.Min.Y+r.Dy()/2+60))
		c.Add(hint)
		return c
	}
	list := ui.NewScrollList(98, 10)
	var rows []ui.View
	for _, rec := range recs {
		rec := rec
		row := &recordingRow{env: env, rec: rec, tap: func() { env.Nav.Push(newPlayerPage(rec.UID)) }}
		rows = append(rows, row)
	}
	list.SetRows(rows)
	list.SetBounds(r)
	c.Add(list)
	return c
}

// recordingRow is one list row: the type icon, the name, the date, the
// length and Multitrack/Stereo, as RØDE's item paints them (0x800ec0).
type recordingRow struct {
	ui.Base
	env     *Env
	rec     desk.Recording
	tap     func()
	pressed bool
}

func (w *recordingRow) CanFocus() bool { return true }
func (w *recordingRow) Activate() bool {
	w.tap()
	return false
}
func (w *recordingRow) Touch(phase ui.TouchPhase, p image.Point) bool {
	switch phase {
	case ui.Down:
		w.pressed = true
		w.MarkDirty()
	case ui.Up:
		if w.pressed && p.In(w.Bounds()) {
			w.tap()
		}
		w.pressed = false
		w.MarkDirty()
	}
	return true
}
func (w *recordingRow) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	r := w.Bounds()
	if w.pressed {
		ui.Panel(dst, r, th.Radius, th.PanelHi, clip)
	} else {
		ui.Card(dst, r, th, clip)
	}
	x := r.Min.X + 24
	c.Icon(dst, "recordings_icon", image.Rect(x, r.Min.Y+22, x+54, r.Max.Y-22), th.Text)
	x += 90
	ui.Text(dst, th.Fonts.Body, w.rec.Name, image.Rect(x, r.Min.Y, x+420, r.Max.Y), font.Left, th.Text, clip)
	x += 440
	if !w.rec.Date.IsZero() {
		ui.Text(dst, th.Fonts.Body, dateText(w.rec.Date), image.Rect(x, r.Min.Y, x+320, r.Max.Y), font.Left, th.TextDim, clip)
	}
	x += 340
	if w.rec.Duration > 0 {
		ui.Text(dst, th.Fonts.Body, recTime(w.rec.Duration), image.Rect(x, r.Min.Y, x+170, r.Max.Y), font.Left, th.TextDim, clip)
	}
	kind := w.env.T("STEREO")
	if w.rec.Multi {
		kind = w.env.T("MULTI")
	}
	ui.Text(dst, th.Fonts.Label, kind, image.Rect(r.Max.X-200, r.Min.Y, r.Max.X-24, r.Max.Y), font.Right, th.TextDim, clip)
	if w.Focused() {
		ui.Focus(dst, r, th.Radius, false, th, clip)
	}
}

// --- the player ---

// Player states (PLAYER.playerState): 0 stopped, 1 playing, 2 paused;
// RØDE's page sets 0 when it closes (0x815cc0). Speeds (playerSpeed) as
// RØDE's pictures playbackSpeedHalf, playbackSpeed1, playbackSpeedTwo: 0
// normal, 1 half, 2 double.
const (
	playerStopped = 0
	playerPlaying = 1
	playerPaused  = 2
)

type playerPage struct {
	uid     int64
	edit    bool
	loaded  bool
	saved   [4]float64
	lastRec string
}

func newPlayerPage(uid int64) Page { return &playerPage{uid: uid} }

func (p *playerPage) recording(env *Env) (desk.Recording, bool) {
	for _, r := range env.Desk.Recordings() {
		if r.UID == p.uid {
			return r, true
		}
	}
	return desk.Recording{}, false
}

func (p *playerPage) Title() string { return "" }

func (p *playerPage) Header(env *Env, bar image.Rectangle) ([]ui.View, bool) {
	th := env.Ctx.Theme
	rec, _ := p.recording(env)
	mid := (bar.Min.X + bar.Max.X) / 2
	name := ui.NewLabel(rec.Name, th.Fonts.Title, th.Text, font.Center)
	name.SetBounds(image.Rect(mid-350, bar.Min.Y+10, mid+350, bar.Min.Y+70))
	sub := ""
	if !rec.Date.IsZero() {
		clock, suffix := clockText(rec.Date, env.Desk.Clock24h())
		if len(clock) > 5 {
			clock = clock[:len(clock)-3]
		}
		sub = dateText(rec.Date) + ", " + clock + suffix
	}
	date := ui.NewLabel(sub, th.Fonts.Small, th.TextDim, font.Center)
	date.SetBounds(image.Rect(mid-350, bar.Min.Y+68, mid+350, bar.Min.Y+104))
	if p.edit {
		views, _ := editHeader(env, bar, "")
		// The cross drops the edit, the tick keeps it; both leave edit mode.
		views[0].(*ui.Button).OnTap = func() {
			for i, prop := range desk.PlayerEnvProps {
				env.Desk.PlayerSetEnv(prop, p.saved[i])
			}
			p.edit = false
			env.Nav.Refresh()
		}
		views[1].(*ui.Button).OnTap = func() { p.edit = false; env.Nav.Refresh() }
		return append(views, name, date), true
	}
	return []ui.View{name, date}, false
}

func (p *playerPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	d := env.Desk
	rec, ok := p.recording(env)
	if !ok {
		return c
	}
	if !p.loaded {
		p.loaded = true
		if rec.Path != "" {
			d.PlayerLoad(rec.Path)
		}
	}
	teal := gfx.Hex(0x3ec6f0)
	// The time display.
	disp := ui.NewPanelView(gfx.Hex(0x0a1e24))
	disp.Radius = 10
	disp.SetBounds(image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+124))
	c.Add(disp)
	tm := ui.NewLabel("", th.Fonts.Huge, teal, font.Center)
	tm.SetBounds(image.Rect(r.Min.X, r.Min.Y+14, r.Max.X, r.Min.Y+110))
	c.Add(tm)
	multi := ui.NewLabel(env.T("MULTI"), th.Fonts.Label, th.TextMuted, font.Right)
	multi.SetBounds(image.Rect(r.Max.X-240, r.Min.Y+22, r.Max.X-30, r.Min.Y+62))
	stereo := ui.NewLabel(env.T("STEREO"), th.Fonts.Label, th.TextMuted, font.Right)
	stereo.SetBounds(image.Rect(r.Max.X-240, r.Min.Y+64, r.Max.X-30, r.Min.Y+104))
	if rec.Multi {
		multi.Color = teal
	} else {
		stereo.Color = teal
	}
	c.Add(multi, stereo)
	total := rec.Duration
	c.Add(&updater{get: func() {
		pl := d.Player()
		if p.edit {
			s := pl.Env[0] * total.Seconds()
			e := pl.Env[3] * total.Seconds()
			tm.Set(recTime(time.Duration(s*float64(time.Second))) + " ▸ " + recTime(time.Duration(e*float64(time.Second))))
			return
		}
		tm.Set(recTime(pl.Position))
	}})

	// The transport row.
	row := image.Rect(r.Min.X, r.Min.Y+144, r.Max.X, r.Min.Y+144+128)
	w := row.Dx()
	cut := func(x0, x1 int) image.Rectangle {
		return image.Rect(row.Min.X+x0*w/1000, row.Min.Y, row.Min.X+x1*w/1000, row.Max.Y)
	}
	sampleRate := 48000.0
	seek := func(dt time.Duration) {
		pl := d.Player()
		pos := pl.Position + dt
		if pos < 0 {
			pos = 0
		}
		d.PlayerJump(int64(pos.Seconds() * sampleRate))
	}
	mk := func(icon string, rc image.Rectangle, tint gfx.Color, tap func()) *ui.Button {
		b := ui.NewButton("", tap)
		b.Icon = icon
		b.Tint = tint
		b.SetBounds(rc)
		c.Add(b)
		return b
	}
	scissors := mk("IconScissor", cut(0, 100), gfx.Hex(0xf5c518), func() {
		p.saved = d.Player().Env
		p.edit = true
		env.Nav.Refresh()
	})
	scissors.Enabled = !p.edit
	marker := gfx.Hex(0xa08c48)
	mk("IconArrowBackFlag", cut(105, 205), marker, func() { seek(-time.Hour) })
	mk("IconArrowBack15", cut(210, 310), th.Text, func() { seek(-15 * time.Second) })
	play := mk("IconPlay", cut(315, 680), th.Text, func() {
		if d.Player().State == playerPlaying {
			d.PlayerSetState(playerPaused)
		} else {
			d.PlayerSetState(playerPlaying)
		}
	})
	c.Add(&updater{get: func() {
		ic := "IconPlay"
		if d.Player().State == playerPlaying {
			ic = "Icon_pause"
		}
		if play.Icon != ic {
			play.Icon = ic
			play.MarkDirty()
		}
	}})
	mk("IconArrowFrwd30", cut(685, 785), th.Text, func() { seek(30 * time.Second) })
	mk("IconArrowFrwdFlag", cut(790, 890), marker, func() { seek(time.Hour) })
	bin := mk("Bin", cut(895, 1000), th.Record, func() {
		env.Nav.Confirm(env.T("DELETE"), env.T("ARE YOU SURE YOU WANT TO DELETE ")+" \""+rec.Name+"\"?", env.T("DELETE"), true, func() {
			d.DeleteRecording(rec.UID)
			env.Nav.Pop()
		})
	})
	bin.Enabled = !p.edit

	// The waveform with the playhead, the envelope handles in edit mode.
	waveR := image.Rect(r.Min.X, row.Max.Y+40, r.Max.X, r.Max.Y-60)
	wave := &playerWave{env: env, page: p}
	wave.SetBounds(waveR)
	c.Add(wave)
	// The scrubber.
	sc := ui.NewSlider(func() float64 { return d.Player().Progress }, func(v float64) {
		d.PlayerJump(int64(v * total.Seconds() * sampleRate))
	})
	sc.SetBounds(image.Rect(r.Min.X-20, r.Max.Y-56, r.Max.X+20, r.Max.Y))
	c.Add(sc)
	// The speed, RØDE's 1.7.6 addition, in the display's corner.
	speed := ui.NewButton("", func() { d.PlayerSetSpeed((d.Player().Speed + 1) % 3) })
	speed.Style = ui.StylePlain
	speed.SetBounds(image.Rect(r.Min.X+20, r.Min.Y+24, r.Min.X+120, r.Min.Y+100))
	c.Add(&updater{get: func() {
		ic := []string{"playbackSpeed1", "playbackSpeedHalf", "playbackSpeedTwo"}[max(0, min(2, int(d.Player().Speed)))]
		if speed.Icon != ic {
			speed.Icon = ic
			speed.MarkDirty()
		}
	}}, speed)
	return c
}

// playerWave draws the recording's waveform box, the playhead and, in edit
// mode, the envelope handles over PLAYER's envelope properties.
type playerWave struct {
	ui.Base
	env  *Env
	page *playerPage
	drag string
	last [5]float64
}

func (w *playerWave) box() image.Rectangle { return ui.Inset(w.Bounds(), 24) }

func (w *playerWave) xOf(v float64) int {
	b := w.box()
	return b.Min.X + int(clamp01(v)*float64(b.Dx()))
}

func (w *playerWave) Dirty() []image.Rectangle {
	pl := w.env.Desk.Player()
	cur := [5]float64{pl.Env[0], pl.Env[1], pl.Env[2], pl.Env[3], pl.Progress}
	if cur != w.last {
		w.last = cur
		w.MarkDirty()
	}
	return w.Base.Dirty()
}

func (w *playerWave) handles() map[string]image.Point {
	b := w.box()
	pl := w.env.Desk.Player()
	return map[string]image.Point{
		"playerEnvStart":   {w.xOf(pl.Env[0]), b.Max.Y},
		"playerEnvFadeIn":  {w.xOf(pl.Env[1]), b.Min.Y},
		"playerEnvFadeOut": {w.xOf(pl.Env[2]), b.Min.Y},
		"playerEnvStop":    {w.xOf(pl.Env[3]), b.Max.Y},
	}
}

func (w *playerWave) Touch(phase ui.TouchPhase, p image.Point) bool {
	if !w.page.edit {
		return false
	}
	d := w.env.Desk
	b := w.box()
	switch phase {
	case ui.Down:
		best, bestD := "", 1<<30
		for name, h := range w.handles() {
			dx, dy := p.X-h.X, p.Y-h.Y
			if dd := dx*dx + dy*dy; dd < bestD && dd < 70*70 {
				best, bestD = name, dd
			}
		}
		w.drag = best
	case ui.Move:
		if w.drag == "" {
			return true
		}
		v := clamp01(float64(p.X-b.Min.X) / float64(b.Dx()))
		pl := d.Player()
		switch w.drag {
		case "playerEnvStart":
			v = min(v, pl.Env[1])
		case "playerEnvFadeIn":
			v = max(pl.Env[0], min(v, pl.Env[2]))
		case "playerEnvFadeOut":
			v = max(pl.Env[1], min(v, pl.Env[3]))
		case "playerEnvStop":
			v = max(v, pl.Env[2])
		}
		d.PlayerSetEnv(w.drag, float64(int(v*1000+0.5))/1000)
		w.MarkDirty()
	case ui.Up:
		w.drag = ""
	}
	return true
}

func (w *playerWave) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	r := w.Bounds()
	ui.Panel(dst, r, th.Radius, gfx.Hex(0x141414), clip)
	b := w.box()
	pl := w.env.Desk.Player()
	mid := (b.Min.Y + b.Max.Y) / 2
	// A stylised waveform: the recording's samples stay on the storage.
	for x := b.Min.X; x < b.Max.X; x += 10 {
		k := float64(x-b.Min.X) / float64(b.Dx())
		h := float64(b.Dy()/2) * (0.15 + 0.6*absf(sinf(k*37))*(0.5+0.5*absf(sinf(k*5))))
		col := gfx.Hex(0x8a8a8a)
		if w.page.edit && (k < pl.Env[0] || k > pl.Env[3]) {
			col = gfx.Hex(0x4a4a4a)
		}
		ui.Panel(dst, image.Rect(x, mid-int(h), x+6, mid+int(h)), 3, col, clip)
	}
	if w.page.edit {
		// The kept region, shaded, with the envelope line and handles.
		s, fi, fo, st := w.xOf(pl.Env[0]), w.xOf(pl.Env[1]), w.xOf(pl.Env[2]), w.xOf(pl.Env[3])
		p := &gfx.Path{}
		p.MoveTo(float64(s), float64(b.Max.Y))
		p.LineTo(float64(fi), float64(b.Min.Y))
		p.LineTo(float64(fo), float64(b.Min.Y))
		p.LineTo(float64(st), float64(b.Max.Y))
		p.Close()
		dst.Fill(p, gfx.Identity, gfx.NonZero, gfx.Solid(gfx.Hex(0x3ec6f0).WithAlpha(0.25)), clip)
		for name, h := range w.handles() {
			col := th.Text
			if name == w.drag {
				col = th.Accent
			}
			ui.Circle(dst, float64(h.X), float64(h.Y), 16, col, clip)
		}
	}
	x := w.xOf(pl.Progress)
	ui.Fill(dst, image.Rect(x-1, b.Min.Y, x+2, b.Max.Y), th.Text, clip)
}

var _ = strconv.Itoa
