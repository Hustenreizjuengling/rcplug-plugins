package pages

import (
	"context"
	"errors"
	"image"
	"time"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// DateTimePage (System > Date & Time), as RØDE's screen builds it: a 24h/12h
// switch and, in 12-hour mode, AM/PM; spinners for Year, Month, Day and
// Hours : Minutes : Seconds (tap one, then turn the knob); the time zone,
// which opens the zone list; and the switch "Show date and time on home
// screen" bottom left. The cross discards, the tick applies: the zone index
// goes to SYSTEM.systemDateTimezone, the 24-hour flag to
// SYSTEM.systemDateTime24h, and the clock itself to the device through the
// host (see desk.SetClock); the home-screen switch writes at once, as
// RØDE's OptionButton does.
type dateTimePage struct {
	init bool
	t    time.Time // the time being edited
	zone int       // the index into desk.TimeZones
	h24  bool
	busy bool
	note string
}

func newDateTimePage() Page { return &dateTimePage{} }

func (p *dateTimePage) Title() string { return "DATE & TIME" }

func (p *dateTimePage) Header(env *Env, bar image.Rectangle) ([]ui.View, bool) {
	th := env.Ctx.Theme
	x := ui.NewButton("", func() { env.Nav.Pop() })
	x.Icon, x.Style, x.Fill = "Red_Cross", ui.StyleRound, gfx.Hex(0xb81e1e)
	x.SetBounds(image.Rect(bar.Min.X+30, bar.Min.Y+22, bar.Min.X+102, bar.Min.Y+94))
	tick := ui.NewButton("", func() { p.apply(env) })
	tick.Icon, tick.Style, tick.Fill = "Green_Tick", ui.StyleRound, th.Accent
	tick.SetBounds(image.Rect(bar.Max.X-102, bar.Min.Y+22, bar.Max.X-30, bar.Min.Y+94))
	title := ui.NewLabel(env.T(p.Title()), th.Fonts.Title, th.Text, font.Center)
	title.SetBounds(image.Rect(bar.Min.X+120, bar.Min.Y, bar.Max.X-120, bar.Max.Y))
	return []ui.View{x, title, tick}, true
}

// load takes the device's current time and settings once.
func (p *dateTimePage) load(env *Env) {
	if p.init {
		return
	}
	p.init = true
	p.t = time.Now().Truncate(time.Second)
	p.zone = int(env.Desk.Int(desk.SystemNode, 0, "systemDateTimezone"))
	p.h24 = env.Desk.Bool(desk.SystemNode, 0, "systemDateTime24h")
}

// set replaces one field of the edited time, keeping the others (a day
// past the month's end rolls over, as time.Date does).
func (p *dateTimePage) set(year, month, day, hour, min, sec int) {
	p.t = time.Date(year, time.Month(month), day, hour, min, sec, 0, p.t.Location())
}

// apply writes the settings and asks the host to set the clock.
func (p *dateTimePage) apply(env *Env) {
	d := env.Desk
	if h := p.h24; h != d.Bool(desk.SystemNode, 0, "systemDateTime24h") {
		d.SetBool(desk.SystemNode, 0, "systemDateTime24h", h)
	}
	zoneName, zone := desk.TimeZone(int64(p.zone))
	if int64(p.zone) != d.Int(desk.SystemNode, 0, "systemDateTimezone") && zoneName != "" {
		d.SetInt(desk.SystemNode, 0, "systemDateTimezone", int64(p.zone))
	}
	t := p.t
	p.busy = true
	env.Nav.Refresh()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := d.SetClock(ctx, &t, zone)
		p.busy = false
		switch {
		case err == nil:
			env.Nav.Pop()
		case errors.Is(err, desk.ErrNoClock):
			p.note = env.TR("", "Uhrzeit und Zeitzone kann rcplug auf diesem Gerät noch nicht stellen; Format und Anzeige sind übernommen.",
				"rcplug cannot set the clock and the zone on this device yet; the format and the display are applied.")
			env.Nav.Refresh()
		default:
			p.note = env.TR("", "Die Uhr konnte nicht gestellt werden.", "The clock could not be set.") + " (" + err.Error() + ")"
			env.Nav.Refresh()
		}
	}()
}

func (p *dateTimePage) Build(env *Env, r image.Rectangle) ui.View {
	p.load(env)
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	spin := func(label string, get func() int, set func(int), lo, hi int, pad bool) *ui.Spinner {
		s := ui.NewSpinner(env.T(label), get, set)
		s.Min, s.Max, s.Pad = lo, hi, pad
		return s
	}
	ymd := func() (int, int, int) { return p.t.Year(), int(p.t.Month()), p.t.Day() }
	hms := func() (int, int, int) { return p.t.Hour(), p.t.Minute(), p.t.Second() }
	setDate := func(y, m, d int) { _, _, _ = y, m, d; h, mi, s := hms(); p.set(y, m, d, h, mi, s) }
	setTime := func(h, mi, s int) { y, m, d := ymd(); p.set(y, m, d, h, mi, s) }
	// Date, left.
	fieldH, labelH := 110, 44
	top := r.Min.Y + 10
	dateR := image.Rect(r.Min.X, top, r.Min.X+560, top+fieldH+labelH)
	dcells := columns(dateR, 3, 16)
	year := spin("YEAR", func() int { return p.t.Year() }, func(v int) { _, m, d := ymd(); setDate(v, m, d) }, 2000, 2099, false)
	month := spin("MONTH", func() int { return int(p.t.Month()) }, func(v int) { y, _, d := ymd(); setDate(y, v, d) }, 1, 12, true)
	day := spin("DAY", func() int { return p.t.Day() }, func(v int) { y, m, _ := ymd(); setDate(y, m, v) }, 1, 31, true)
	year.SetBounds(dcells[0])
	month.SetBounds(dcells[1])
	day.SetBounds(dcells[2])
	c.Add(year, month, day)
	// Time, right, with the colons between the fields.
	timeR := image.Rect(r.Max.X-560, top, r.Max.X, top+fieldH+labelH)
	tcells := columns(timeR, 3, 36)
	hours := spin("HOURS", func() int {
		h := p.t.Hour()
		if !p.h24 {
			h %= 12
			if h == 0 {
				h = 12
			}
		}
		return h
	}, func(v int) {
		_, mi, s := hms()
		if !p.h24 {
			v %= 12
			if p.t.Hour() >= 12 {
				v += 12
			}
		}
		setTime(v, mi, s)
	}, 0, 23, true)
	if !p.h24 {
		hours.Min, hours.Max = 1, 12
	}
	mins := spin("MINUTES", func() int { return p.t.Minute() }, func(v int) { h, _, s := hms(); setTime(h, v, s) }, 0, 59, true)
	secs := spin("SECONDS", func() int { return p.t.Second() }, func(v int) { h, mi, _ := hms(); setTime(h, mi, v) }, 0, 59, true)
	hours.SetBounds(tcells[0])
	mins.SetBounds(tcells[1])
	secs.SetBounds(tcells[2])
	c.Add(hours, mins, secs)
	for _, x := range []int{tcells[0].Max.X, tcells[1].Max.X} {
		colon := ui.NewLabel(":", th.Fonts.Big, th.Text, font.Center)
		colon.SetBounds(image.Rect(x, top, x+36, top+fieldH))
		c.Add(colon)
	}
	// 24h / 12h and AM / PM under the time.
	segY := top + fieldH + labelH + 12
	seg := ui.NewSegmented([]string{env.T("24H"), env.T("12H")}, func() int {
		if p.h24 {
			return 0
		}
		return 1
	}, func(i int) { p.h24 = i == 0; env.Nav.Refresh() })
	seg.SetBounds(image.Rect(timeR.Min.X, segY, timeR.Min.X+260, segY+72))
	c.Add(seg)
	if !p.h24 {
		ampm := ui.NewSegmented([]string{env.T("AM"), env.T("PM")}, func() int {
			if p.t.Hour() >= 12 {
				return 1
			}
			return 0
		}, func(i int) {
			h, mi, s := hms()
			h = h%12 + 12*i
			setTime(h, mi, s)
		})
		ampm.SetBounds(image.Rect(timeR.Max.X-260, segY, timeR.Max.X, segY+72))
		c.Add(ampm)
	}
	// The time zone: a wide button opening the list.
	zoneName, _ := desk.TimeZone(int64(p.zone))
	if zoneName == "" {
		zoneName = "—"
	}
	zl := ui.NewLabel(env.T("TIME ZONE"), th.Fonts.Small, th.TextDim, font.Left)
	zl.SetBounds(image.Rect(r.Min.X+8, segY+56, r.Min.X+560, segY+96))
	zb := ui.NewButton(zoneName, func() { env.Nav.Push(newTimeZonePage(p)) })
	zb.Icon = "IconDateTime"
	zb.SetBounds(image.Rect(r.Min.X, segY+100, r.Max.X, segY+184))
	c.Add(zl, zb)
	// Bottom left: show the clock on the home screen (writes at once).
	rr := image.Rect(r.Min.X, r.Max.Y-th.Row-10, r.Min.X+760, r.Max.Y-10)
	card := ui.NewCard()
	card.SetBounds(rr)
	tog := ui.NewToggle(func() bool { return env.Desk.Bool(desk.SystemNode, 0, "systemDateTimeOnHome") },
		func(on bool) { env.Desk.SetBool(desk.SystemNode, 0, "systemDateTimeOnHome", on) })
	tog.Label = env.T("SHOW DATE AND TIME ON HOME SCREEN")
	tog.SetBounds(ui.Inset(rr, 14))
	c.Add(card, tog)
	// A note when the host cannot set the clock, or while it does.
	text := p.note
	if p.busy {
		text = env.TR("", "Uhr wird gestellt …", "Setting the clock…")
	}
	if text != "" {
		n := ui.NewParagraph(text, th.Fonts.Small, th.Warn)
		n.Align = font.Right
		n.SetBounds(image.Rect(rr.Max.X+20, rr.Min.Y, r.Max.X, rr.Max.Y))
		c.Add(n)
	}
	return c
}

// timeZonePage is the zone list: one row per entry, the chosen one light;
// a tap chooses and goes back.
type timeZonePage struct {
	owner *dateTimePage
}

func newTimeZonePage(owner *dateTimePage) Page { return &timeZonePage{owner: owner} }

func (p *timeZonePage) Title() string { return "TIME ZONE" }

func (p *timeZonePage) Build(env *Env, r image.Rectangle) ui.View {
	list := ui.NewScrollList(84, 10)
	list.SetBounds(r)
	rows := make([]ui.View, len(desk.TimeZones))
	for i, z := range desk.TimeZones {
		i := i
		b := ui.NewButton(z.Name, func() {
			p.owner.zone = i
			env.Nav.Pop()
		})
		if i == p.owner.zone {
			b.Style = ui.StyleLight
		}
		rows[i] = b
	}
	list.SetRows(rows)
	list.Reveal(rows[clampIndex(p.owner.zone, len(rows))])
	return list
}

func clampIndex(i, n int) int {
	if i < 0 {
		return 0
	}
	if i >= n {
		return n - 1
	}
	return i
}
