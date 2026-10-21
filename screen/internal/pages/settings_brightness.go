package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// BrightnessPage (Display > Brightness), as RØDE's screen builds it: the
// Display slider (GUI.screenBrightness), the Buttons control with its two
// values Active and Inactive (activeButtonsBrightness,
// inactiveButtonsBrightness), the Auto switch (autoBrightness) and Dim After
// with its six buttons: 30 sec, 1 min, 5 min, 15 min, 30 min, Never
// (GUI.screenDimAfterSeconds, in seconds; 0 is Never).

// dimAfterOptions are the screen's buttons, in its order, in seconds.
var dimAfterOptions = []int64{30, 60, 300, 900, 1800, 0}

// dimAfterText is a button's caption: "Never", "30 sec" or "5 min".
func dimAfterText(env *Env, secs int64) string {
	switch {
	case secs <= 0:
		return env.T("NEVER")
	case secs < 60:
		return intText(secs) + " " + env.TR("SEC", "Sek.", "sec")
	}
	return intText(secs/60) + " " + env.TR("MIN", "Min.", "min")
}

func newBrightnessPage() Page {
	return &funcPage{title: "BRIGHTNESS", build: func(env *Env, r image.Rectangle) ui.View {
		c := &ui.Container{}
		c.SetBounds(r)
		th := env.Ctx.Theme
		d := env.Desk
		slider := func(rr image.Rectangle, label, icon, prop string) {
			card := ui.NewCard()
			card.SetBounds(rr)
			ic := ui.NewIcon(icon, th.Text)
			ic.SetBounds(image.Rect(rr.Min.X+20, rr.Min.Y+(rr.Dy()-64)/2, rr.Min.X+84, rr.Min.Y+(rr.Dy()-64)/2+64))
			s := ui.NewSlider(func() float64 { return d.Num(desk.GUINode, 0, prop) / 255 },
				func(v float64) { d.SetInt(desk.GUINode, 0, prop, int64(v*255+0.5)) })
			s.Label = label
			s.Text = func(v float64) string { return pct(v) }
			s.Step = 5.0 / 255
			s.SetBounds(image.Rect(rr.Min.X+100, rr.Min.Y+12, rr.Max.X-20, rr.Max.Y-12))
			c.Add(card, ic, s)
		}
		h := 118
		slider(image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+h), env.T("DISPLAY"), "IconBrightnessDisplay", "screenBrightness")
		// Buttons: active and inactive side by side.
		y := r.Min.Y + h + th.Gap
		half := (r.Dx() - th.Gap) / 2
		slider(image.Rect(r.Min.X, y, r.Min.X+half, y+h), env.T("BUTTONS")+" · "+env.T("ACTIVE"), "IconBrightnessActive", "activeButtonsBrightness")
		slider(image.Rect(r.Max.X-half, y, r.Max.X, y+h), env.T("BUTTONS")+" · "+env.T("INACTIVE"), "IconBrightnessInactive", "inactiveButtonsBrightness")
		// Auto, as the screen's switch.
		y += h + th.Gap
		if d.Has(desk.GUINode, 0, "autoBrightness") {
			rr := image.Rect(r.Min.X, y, r.Min.X+half, y+th.Row)
			card := ui.NewCard()
			card.SetBounds(rr)
			tog := ui.NewToggle(func() bool { return d.Bool(desk.GUINode, 0, "autoBrightness") },
				func(on bool) { d.SetBool(desk.GUINode, 0, "autoBrightness", on) })
			tog.Label = env.T("AUTO")
			tog.SetBounds(ui.Inset(rr, 14))
			c.Add(card, tog)
		}
		// Dim After: the six value buttons, the current one light.
		dl := ui.NewLabel(env.T("DIM AFTER"), th.Fonts.Small, th.TextDim, font.Left)
		dl.SetBounds(image.Rect(r.Max.X-half+8, y-2, r.Max.X, y+30))
		c.Add(dl)
		labels := make([]string, len(dimAfterOptions))
		for i, s := range dimAfterOptions {
			labels[i] = dimAfterText(env, s)
		}
		seg := ui.NewSegmented(labels, func() int {
			cur := dimAfterCurrent(d)
			for i, s := range dimAfterOptions {
				if s == cur {
					return i
				}
			}
			return -1
		}, func(i int) { dimAfterWrite(d, dimAfterOptions[i]) })
		seg.SetBounds(image.Rect(r.Max.X-half, y+32, r.Max.X, y+32+72))
		c.Add(seg)
		return c
	}}
}

// dimAfterCurrent reads GUI.screenDimAfterSeconds. The screen program holds
// it as a number of seconds; a desk that reports it as a flag counts true
// as the first option.
func dimAfterCurrent(d *desk.Model) int64 {
	v, ok := d.Get(desk.GUINode, 0, "screenDimAfterSeconds")
	if !ok {
		return 0
	}
	if v.Kind == 'b' {
		if v.Bool {
			return dimAfterOptions[0]
		}
		return 0
	}
	return v.Int()
}

// dimAfterWrite writes the seconds in the property's own type.
func dimAfterWrite(d *desk.Model, secs int64) {
	if v, ok := d.Get(desk.GUINode, 0, "screenDimAfterSeconds"); ok && v.Kind == 'b' {
		d.SetBool(desk.GUINode, 0, "screenDimAfterSeconds", secs != 0)
		return
	}
	d.SetInt(desk.GUINode, 0, "screenDimAfterSeconds", secs)
}
