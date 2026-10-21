package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// The first-run welcome (RØDE's WelcomePage with its WelcomeNavigation,
// 1.7.6): the device's name with "Get Started!", the language, date and
// time, getting connected for the firmware check (Wi-Fi, wired, or the
// RØDECaster App over USB), then the setup assistant. The language and
// date-and-time steps show the settings pages of the same name with a
// cross or back arrow at the left and a tick at the right.

func init() {
	register(IDWelcome, func() Page { return NewWelcome(nil) })
}

// WelcomePage runs the welcome flow; done runs when it ends (nil: the
// mixer).
type WelcomePage struct {
	step int
	done func()
}

// The welcome's steps.
const (
	welcomeStart     = iota // the device's name and "Get Started!"
	welcomeLanguage         // the language page
	welcomeDateTime         // the date and time page
	welcomeConnected        // Wi-Fi, wired or the app over USB
	welcomeApp              // how to connect the RØDECaster App
	welcomeSteps
)

// NewWelcome makes the welcome flow at its first screen.
func NewWelcome(done func()) Page { return &WelcomePage{done: done} }

// Step is the current step (for tests).
func (p *WelcomePage) Step() int { return p.step }

// SetStep jumps to a step (previews and tests).
func (p *WelcomePage) SetStep(n int) {
	if n >= 0 && n < welcomeSteps {
		p.step = n
	}
}

// welcomeTitles are the steps' captions (RØDE's keys; "" for a full screen).
var welcomeTitles = []string{"", "LANGUAGE", "DATE & TIME", "GET CONNECTED", "CONNECT TO RØDECASTER APP"}

func (p *WelcomePage) Title() string { return welcomeTitles[p.step] }

// FullBleed: the first screen has no header.
func (p *WelcomePage) FullBleed() bool { return p.step == welcomeStart }

// Header: the cross or back arrow, the title and the tick.
func (p *WelcomePage) Header(env *Env, bar image.Rectangle) ([]ui.View, bool) {
	th := env.Ctx.Theme
	var left *ui.Button
	if p.step == welcomeLanguage {
		left = ui.NewButton("", func() { p.finish(env) })
		left.Icon, left.Style, left.Fill = "Red_Cross", ui.StyleRound, gfx.Hex(0xb81e1e)
	} else {
		left = ui.NewButton("", func() { p.back(env) })
		left.Icon = "IconBack"
	}
	left.SetBounds(image.Rect(bar.Min.X+30, bar.Min.Y+22, bar.Min.X+102, bar.Min.Y+94))
	title := ui.NewLabel(env.T(p.Title()), th.Fonts.Title, th.Text, font.Center)
	title.SetBounds(image.Rect(bar.Min.X+120, bar.Min.Y, bar.Max.X-120, bar.Max.Y))
	views := []ui.View{left, title}
	if p.step != welcomeApp {
		tick := ui.NewButton("", func() { p.next(env) })
		tick.Icon, tick.Style, tick.Fill = "Green_Tick", ui.StyleRound, th.Accent
		tick.SetBounds(image.Rect(bar.Max.X-102, bar.Min.Y+22, bar.Max.X-30, bar.Min.Y+94))
		views = append(views, tick)
	}
	return views, true
}

// next goes on; after getting connected the setup assistant takes over.
func (p *WelcomePage) next(env *Env) {
	if p.step < welcomeConnected {
		p.step++
		env.Nav.Refresh()
		return
	}
	env.Nav.Replace(NewSetupAssistant(p.done))
}

// back returns a step.
func (p *WelcomePage) back(env *Env) {
	if p.step > welcomeStart {
		p.step--
		if p.step == welcomeApp {
			p.step = welcomeConnected
		}
		env.Nav.Refresh()
	}
}

// finish leaves the flow.
func (p *WelcomePage) finish(env *Env) {
	if p.done != nil {
		p.done()
		return
	}
	env.Nav.Home()
}

func (p *WelcomePage) Build(env *Env, r image.Rectangle) ui.View {
	switch p.step {
	case welcomeStart:
		return p.start(env, r)
	case welcomeLanguage:
		return Detail("language").Build(env, r)
	case welcomeDateTime:
		return Detail("datetime").Build(env, r)
	case welcomeApp:
		return p.app(env, r)
	}
	return p.connected(env, r)
}

// start: the device's name and the way in.
func (p *WelcomePage) start(env *Env, r image.Rectangle) ui.View {
	th := env.Ctx.Theme
	c := &ui.Container{}
	c.SetBounds(r)
	logo := &pictureView{name: deviceLogo(env)}
	logo.SetBounds(centred(image.Rect(r.Min.X, r.Min.Y+200, r.Max.X, r.Min.Y+300), 760, 66))
	go1 := ui.NewButton(env.T("GET STARTED!"), func() { p.next(env) })
	go1.Fill = th.Blue
	go1.SetBounds(image.Rect(r.Min.X+r.Dx()/2-200, r.Max.Y-240, r.Min.X+r.Dx()/2+200, r.Max.Y-150))
	c.Add(logo, go1)
	return c
}

// connected: the three ways to a network for the firmware check, and skip.
func (p *WelcomePage) connected(env *Env, r image.Rectangle) ui.View {
	th := env.Ctx.Theme
	c := &ui.Container{}
	c.SetBounds(r)
	tiles := []struct {
		label, icon string
		open        func()
	}{
		{env.T("WI-FI"), "IconWifi", func() { env.Nav.Open(IDNetwork) }},
		{env.T("WIRED"), "IconWired", func() {
			if !env.Desk.Bool(desk.NetworkNode, 0, "wiredConnected") {
				env.Nav.Toast(env.T("PLEASE CONNECT A NETWORK CABLE"))
				return
			}
			env.Nav.Open(IDNetwork)
		}},
		// The app's name is not translated on the device either.
		{"RODECaster App " + env.T("FOR PC/MAC"), "IconUsb", func() { p.step = welcomeApp; env.Nav.Refresh() }},
	}
	w, h, gap := 210, 260, 24
	x := r.Min.X + (r.Dx()-(len(tiles)*w+(len(tiles)-1)*gap))/2
	y := r.Min.Y + 20
	for _, t := range tiles {
		b := ui.NewTile(t.label, t.icon, t.open)
		b.Fill = th.Blue
		b.SetBounds(image.Rect(x, y, x+w, y+h+48))
		c.Add(b)
		x += w + gap
	}
	skip := ui.NewButton(env.T("SKIP"), func() { p.next(env) })
	skip.Style = ui.StylePlain
	skip.SetBounds(image.Rect(r.Min.X+r.Dx()/2-120, r.Max.Y-90, r.Min.X+r.Dx()/2+120, r.Max.Y-20))
	c.Add(skip)
	return c
}

// app: how to reach the firmware update through the RØDECaster App.
func (p *WelcomePage) app(env *Env, r image.Rectangle) ui.View {
	th := env.Ctx.Theme
	c := &ui.Container{}
	c.SetBounds(r)
	icon := &pictureView{name: "IconRodeCentral"}
	icon.SetBounds(image.Rect(r.Min.X+80, r.Min.Y+40, r.Min.X+260, r.Min.Y+220))
	lines := []string{
		env.T("DOWNLOAD RØDECASTER APP FOR PC/MAC FROM"),
		"www.rode.com/rode-central",
		env.T("OPEN RØDECASTER APP"),
		env.T("CONNECT YOUR RØDECASTER PRO II VIA USB"),
	}
	for i, s := range lines {
		col := th.Text
		if i == 1 {
			col = th.Accent
		}
		l := ui.NewLabel(s, th.Fonts.Body, col, font.Left)
		l.SetBounds(image.Rect(r.Min.X+300, r.Min.Y+50+i*70, r.Max.X-40, r.Min.Y+110+i*70))
		c.Add(l)
	}
	qr := &pictureView{name: "UserGuideQR"}
	qr.SetBounds(image.Rect(r.Max.X-260, r.Max.Y-240, r.Max.X-60, r.Max.Y-40))
	c.Add(icon, qr)
	return c
}
