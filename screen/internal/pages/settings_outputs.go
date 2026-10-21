package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// MultitrackPage (Outputs > Multitrack), as RØDE's screen builds it: three
// tabs. Recording: Off / Pre-fader / Post-fader for the microSD multitrack
// (OUTPUT.recordingMultitrackMode) with Bypass Processing
// (recordingProcessingBypass). USB 1 Output: the same three for the USB
// multitrack with its bypass (outputMultiBypass). USB 1 Input: Standard /
// Expanded. The USB settings share OUTPUT.outputMultiMode (see desk.USBMode);
// changing them interrupts USB audio, so the screen asks first, and leaving
// Expanded also unassigns the extended inputs, which it says too.
type multitrackPage struct {
	tab int
}

func newMultitrackPage() Page { return &multitrackPage{} }

func (p *multitrackPage) Title() string { return "MULTITRACK" }

func (p *multitrackPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	tabs := ui.NewSegmented([]string{env.T("RECORDING"), "USB 1 " + env.TR("", "Output", "Output"), "USB 1 " + env.TR("", "Input", "Input")},
		func() int { return p.tab }, func(i int) { p.tab = i; env.Nav.Refresh() })
	tabs.SetBounds(image.Rect(r.Min.X+160, r.Min.Y, r.Max.X-160, r.Min.Y+72))
	c.Add(tabs)
	body := image.Rect(r.Min.X, r.Min.Y+100, r.Max.X, r.Max.Y)
	switch p.tab {
	case 0:
		p.busTab(env, c, body, "recordingMultitrackMode", "recordingProcessingBypass", nil)
	case 1:
		p.busTab(env, c, body, "outputMultiMode", "outputMultiBypass", func(mode int64, apply func()) {
			env.Nav.Confirm("", env.T("USB AUDIO WILL BE INTERRUPTED TO APPLY THIS CHANGE.")+"\n"+env.T("APPLY THIS SETTING NOW?"), env.T("APPLY"), false, apply)
		})
	default:
		p.inputTab(env, c, body)
	}
	return c
}

// busTab lays the Off / Pre-fader / Post-fader cards and the bypass switch
// (shown with pre-fader, where processing can be left out).
func (p *multitrackPage) busTab(env *Env, c *ui.Container, r image.Rectangle, modeProp, bypassProp string, confirm func(mode int64, apply func())) {
	th := env.Ctx.Theme
	d := env.Desk
	current := func() int64 {
		v := d.Int(desk.OutputNode, 0, modeProp)
		if modeProp == "outputMultiMode" {
			return desk.USBOutputMode(v)
		}
		return v
	}
	write := func(mode int64) {
		if modeProp == "outputMultiMode" {
			mode = desk.USBMode(mode, desk.USBInputExpanded(d.Int(desk.OutputNode, 0, modeProp)))
		}
		d.SetInt(desk.OutputNode, 0, modeProp, mode)
	}
	cl := ui.NewChoiceList([]ui.Choice{
		{Title: env.T("OFF")}, {Title: env.T("PRE-FADER"), Icon: "IconListenPreFader"}, {Title: env.T("POST-FADER"), Icon: "IconListenAfterFader"},
	}, func() int { return int(current()) }, func(i int) {
		mode := int64(i)
		if mode == current() {
			return
		}
		if confirm != nil {
			confirm(mode, func() { write(mode) })
			return
		}
		write(mode)
	})
	cl.Columns = 3
	cl.RowH = 190
	cl.SetBounds(image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+190))
	c.Add(cl)
	// What the choice means, in RØDE's words.
	note := func() string {
		switch current() {
		case desk.MultitrackPre:
			if d.Bool(desk.OutputNode, 0, bypassProp) {
				return env.T("ALL PROCESSING, EFFECTS AND FADER POSITIONS WILL BE ") + env.T("EXCLUDED ") + env.T("FROM MULTITRACK.")
			}
			return env.T("FADER POSITIONS WILL BE ") + env.T("EXCLUDED ") + env.T("FROM MULTITRACK.")
		case desk.MultitrackPost:
			return env.T("PROCESSING, EFFECTS AND FADER POSITIONS WILL BE ") + env.T("INCLUDED ") + env.T("IN MULTITRACK.") + "\n" + env.T("WARNING! FADER POSITION WILL DETERMINE RECORDING LEVEL.")
		}
		if modeProp == "outputMultiMode" {
			return env.T("MULTITRACK FOR USB IS DISABLED.")
		}
		return env.T("MULTITRACK RECORDING IS DISABLED.")
	}
	para := ui.NewParagraph(note(), th.Fonts.Small, th.TextDim)
	para.Align = font.Center
	para.SetBounds(image.Rect(r.Min.X, r.Min.Y+206, r.Max.X, r.Min.Y+286))
	c.Add(para, &updater{get: func() {
		if s := note(); s != para.Text {
			para.Text = s
			para.MarkDirty()
		}
	}})
	if current() == desk.MultitrackPre {
		rr := image.Rect(r.Min.X+200, r.Min.Y+296, r.Max.X-200, r.Min.Y+296+th.Row)
		card := ui.NewCard()
		card.SetBounds(rr)
		tog := ui.NewToggle(func() bool { return d.Bool(desk.OutputNode, 0, bypassProp) },
			func(v bool) { d.SetBool(desk.OutputNode, 0, bypassProp, v) })
		tog.Label = env.T("BYPASS PROCESSING")
		tog.SetBounds(ui.Inset(rr, 14))
		c.Add(card, tog)
	}
}

// inputTab: Standard / Expanded for the USB 1 input, with RØDE's note and
// confirmations.
func (p *multitrackPage) inputTab(env *Env, c *ui.Container, r image.Rectangle) {
	th := env.Ctx.Theme
	d := env.Desk
	expanded := func() bool { return desk.USBInputExpanded(d.Int(desk.OutputNode, 0, "outputMultiMode")) }
	write := func(exp bool) {
		d.SetInt(desk.OutputNode, 0, "outputMultiMode", desk.USBMode(d.Int(desk.OutputNode, 0, "outputMultiMode"), exp))
	}
	cl := ui.NewChoiceList([]ui.Choice{{Title: env.T("STANDARD")}, {Title: env.T("EXPANDED")}},
		func() int {
			if expanded() {
				return 1
			}
			return 0
		}, func(i int) {
			exp := i == 1
			if exp == expanded() {
				return
			}
			if exp {
				env.Nav.Confirm("", env.T("USB AUDIO WILL BE INTERRUPTED TO APPLY THIS CHANGE.")+"\n"+env.T("APPLY THIS SETTING NOW?"), env.T("APPLY"), false, func() { write(true) })
			} else {
				env.Nav.Confirm("", env.T("ALL EXTENDED INPUTS WILL BE UNASSIGNED.")+"\n"+env.T("USB AUDIO WILL BE INTERRUPTED TO APPLY THIS CHANGE."), env.T("APPLY"), false, func() { write(false) })
			}
		})
	cl.Columns = 2
	cl.RowH = 190
	cl.SetBounds(image.Rect(r.Min.X+100, r.Min.Y, r.Max.X-100, r.Min.Y+190))
	c.Add(cl)
	note := ui.NewParagraph(env.TR("", "Im erweiterten Modus bietet USB 1 mit dem RØDECaster-Treiber für macOS/Windows vier zusätzliche Kanäle neben System- und Chat-Eingang.",
		"USB 1 will feature four additional channels alongside System and Chat inputs when used with the RØDECaster Virtual Device Driver for macOS/Windows."), th.Fonts.Small, th.TextDim)
	note.Align = font.Center
	note.SetBounds(image.Rect(r.Min.X+100, r.Min.Y+210, r.Max.X-100, r.Min.Y+300))
	c.Add(note)
}
