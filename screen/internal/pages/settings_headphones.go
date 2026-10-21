package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// HeadphonesPage (Outputs > Headphones), as RØDE's screen has it: the port
// is picked in the header with ‹ › around its number, and three cards
// choose the gain structure of that port, HEADPHONE[p].headphoneType:
// RØDE NTH-100 (0), High Sensitivity (1), Low Sensitivity (2). Leaving High
// Sensitivity raises the output, so the screen asks first ("This will
// increase the headphone output. Please turn down your headphones first.")
// and writes only on Ok.
type headphonesPage struct {
	port int
}

func newHeadphonesPage() Page { return &headphonesPage{} }

func (p *headphonesPage) Title() string { return "HEADPHONES" }

// Header: ‹ (n) › around the title, like the routing mix page's output
// switcher.
func (p *headphonesPage) Header(env *Env, bar image.Rectangle) ([]ui.View, bool) {
	th := env.Ctx.Theme
	n := env.Desk.Headphones()
	cx := (bar.Min.X + bar.Max.X) / 2
	cy := (bar.Min.Y + bar.Max.Y) / 2
	title := ui.NewLabel(env.T(p.Title()), th.Fonts.Title, th.Text, font.Right)
	title.SetBounds(image.Rect(cx-460, bar.Min.Y, cx-110, bar.Max.Y))
	num := ui.NewIcon(portIcon(p.port), th.Text)
	num.SetBounds(image.Rect(cx-30, cy-30, cx+30, cy+30))
	prev := chevron(env, false, n > 1, func() { p.port = (p.port + n - 1) % n; env.Nav.Refresh() })
	prev.SetBounds(image.Rect(cx-110, cy-50, cx-30, cy+50))
	next := chevron(env, true, n > 1, func() { p.port = (p.port + 1) % n; env.Nav.Refresh() })
	next.SetBounds(image.Rect(cx+30, cy-50, cx+110, cy+50))
	return []ui.View{title, num, prev, next}, false
}

// portIcon is the circled number of a headphone port (RØDE's circled1…4).
func portIcon(port int) string {
	return "circled" + intText(int64(port+1))
}

func (p *headphonesPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	port := p.port
	// The big headphone picture on the left, the three cards on the right
	// (RØDE's cards sit at x 483, 715 wide).
	pic := ui.NewIcon("IconHeadphones", th.TextDim)
	pic.SetBounds(centred(image.Rect(r.Min.X, r.Min.Y, r.Min.X+420, r.Max.Y), 240, 240))
	c.Add(pic)
	opts := []radioSpec{
		{title: "RØDE NTH-100", icon: "HeadphonesNTH100", value: desk.HeadphonesNTH100},
		{title: "HIGH SENSITIVITY", icon: "HeadphonesHighSensitivity", value: desk.HeadphonesHighSensitivity},
		{title: "LOW SENSITIVITY", icon: "HeadphonesLowSensitivity", value: desk.HeadphonesLowSensitivity},
	}
	choices := make([]ui.Choice, len(opts))
	for i, o := range opts {
		choices[i] = ui.Choice{Title: env.T(o.title), Icon: o.icon}
	}
	current := func() int64 { return env.Desk.Int("HEADPHONE", port, "headphoneType") }
	write := func(v int64) { env.Desk.SetInt("HEADPHONE", port, "headphoneType", v) }
	cl := ui.NewChoiceList(choices, func() int {
		cur := current()
		for i, o := range opts {
			if o.value == cur {
				return i
			}
		}
		return -1
	}, func(i int) {
		v := opts[i].value
		if v == current() {
			return
		}
		if current() == desk.HeadphonesHighSensitivity {
			// Both other settings are louder: RØDE's screen asks first.
			env.Nav.Confirm("", env.T("THIS WILL INCREASE THE HEADPHONE OUTPUT.")+"\n"+env.T("PLEASE TURN DOWN YOUR HEADPHONES FIRST."),
				env.T("OK"), false, func() { write(v) })
			return
		}
		write(v)
	})
	cl.RowH = 130
	cl.SetBounds(image.Rect(r.Max.X-715, r.Min.Y+20, r.Max.X, r.Min.Y+20+3*130+2*16))
	c.Add(cl)
	return c
}
