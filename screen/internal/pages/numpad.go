package pages

import (
	"image"
	"strconv"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// NumPadPage is RØDE's NumPadMinMaxPage: a prompt, the number being typed
// in a black field with its allowed range, a 3 × 4 keypad (digits, clear,
// backspace), and the cross and tick in the header; the knob counts the
// value up and down (the EncoderLeft/EncoderRight pictures) and its press
// applies. The value is clamped to min…max on apply. Pages that take a
// number (a port, a jitter buffer, a MIDI number) push one with their
// prompt and get the number back through done.
type NumPadPage struct {
	Prompt   string
	Min, Max int
	text     string
	done     func(int)
}

// NewNumPad makes a number entry page for min…max starting at initial.
func NewNumPad(prompt string, initial, min, max int, done func(int)) *NumPadPage {
	return &NumPadPage{Prompt: prompt, Min: min, Max: max, text: strconv.Itoa(initial), done: done}
}

func (p *NumPadPage) Title() string { return p.Prompt }

// value is the typed number clamped to the range (min when empty).
func (p *NumPadPage) value() int {
	n, err := strconv.Atoi(p.text)
	if err != nil {
		n = p.Min
	}
	if n < p.Min {
		n = p.Min
	}
	if n > p.Max {
		n = p.Max
	}
	return n
}

func (p *NumPadPage) apply(env *Env) {
	if p.done != nil {
		p.done(p.value())
	}
	env.Nav.Pop()
}

func (p *NumPadPage) Header(env *Env, bar image.Rectangle) ([]ui.View, bool) {
	th := env.Ctx.Theme
	x := ui.NewButton("", func() { env.Nav.Pop() })
	x.Icon, x.Style, x.Fill = "Red_Cross", ui.StyleRound, gfx.Hex(0xb81e1e)
	x.SetBounds(image.Rect(bar.Min.X+30, bar.Min.Y+22, bar.Min.X+102, bar.Min.Y+94))
	tick := ui.NewButton("", func() { p.apply(env) })
	tick.Icon, tick.Style, tick.Fill = "Green_Tick", ui.StyleRound, th.Accent
	tick.SetBounds(image.Rect(bar.Max.X-102, bar.Min.Y+22, bar.Max.X-30, bar.Min.Y+94))
	title := ui.NewLabel(p.Prompt, th.Fonts.Title, th.Text, font.Center)
	title.SetBounds(image.Rect(bar.Min.X+120, bar.Min.Y, bar.Max.X-120, bar.Max.Y))
	return []ui.View{x, title, tick}, true
}

// Encoder: turning steps the number, a press applies.
func (p *NumPadPage) Encoder(env *Env, ev desk.EncoderEvent) bool {
	switch {
	case ev.Delta != 0:
		p.text = strconv.Itoa(p.value() + ev.Delta)
		p.text = strconv.Itoa(p.value())
		env.Nav.Refresh()
		return true
	case ev.Release && ev.Held == 0:
		p.apply(env)
		return true
	}
	return false
}

func (p *NumPadPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	// The value field on the left with the range under it; the knob hints
	// beside it.
	left := image.Rect(r.Min.X, r.Min.Y, r.Min.X+520, r.Max.Y)
	field := ui.NewPanelView(th.Black)
	field.SetBounds(image.Rect(left.Min.X, left.Min.Y+40, left.Max.X, left.Min.Y+170))
	val := ui.NewLabel(p.text, th.Fonts.Huge, th.Text, font.Center)
	val.SetBounds(field.Bounds())
	rng := ui.NewLabel(strconv.Itoa(p.Min)+" – "+strconv.Itoa(p.Max), th.Fonts.Small, th.TextDim, font.Center)
	rng.SetBounds(image.Rect(left.Min.X, left.Min.Y+180, left.Max.X, left.Min.Y+220))
	c.Add(field, val, rng)
	hints := []struct{ icon, text string }{
		{"EncoderLeft", env.TR("", "Drehen: Wert ändern", "Turn: change the value")},
		{"EncoderApply", env.TR("", "Drücken: übernehmen", "Press: apply")},
	}
	for i, h := range hints {
		y := left.Min.Y + 260 + i*70
		ic := ui.NewIcon(h.icon, 0) // RØDE's encoder pictures in their own colours
		ic.SetBounds(image.Rect(left.Min.X+20, y, left.Min.X+70, y+50))
		l := ui.NewLabel(h.text, th.Fonts.Small, th.TextDim, font.Left)
		l.SetBounds(image.Rect(left.Min.X+90, y, left.Max.X, y+50))
		c.Add(ic, l)
	}
	// The keypad on the right: 1–9, clear, 0, backspace.
	pad := image.Rect(r.Max.X-600, r.Min.Y, r.Max.X, r.Max.Y)
	keys := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "C", "0", "⌫"}
	cells := grid(pad, 3, 4, th.Gap)
	for i, k := range keys {
		k := k
		b := ui.NewButton(k, func() {
			switch k {
			case "C":
				p.text = ""
			case "⌫":
				if len(p.text) > 0 {
					p.text = p.text[:len(p.text)-1]
				}
			default:
				if len(p.text) < 9 {
					p.text += k
				}
			}
			val.Set(p.text)
		})
		b.Font = th.Fonts.Title
		if k == "⌫" {
			b.Label, b.Icon = "", "BackspaceKey"
		}
		b.SetBounds(cells[i])
		c.Add(b)
	}
	return c
}
