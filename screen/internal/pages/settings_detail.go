package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// --- Display ---

// --- Outputs ---

func newMonitorPage() Page {
	return newTogglesPage("MONITOR", []toggleRow{
		{key: "AUTO-MUTE MONITOR OUTPUT", node: desk.OutputNode, prop: "outputMonAutoMute"},
		{key: "AUTO-MUTE BLUETOOTH OUTPUT", node: desk.OutputNode, prop: "outputBTAutoMute"},
		{key: "FIXED OUTPUT LEVEL", node: desk.OutputNode, prop: "outputMonFixed"},
		{key: "DISABLE HEADPHONE OUTPUTS", node: desk.SystemNode, prop: "disableAllHeadphoneOutputs"},
	})
}

func newListenPage() Page {
	return newChoicePage("LISTEN", 1, []radioSpec{
		{title: "AFTER-FADER LISTEN", icon: "IconListenAfterFader", value: 0,
			desc: "WHEN SOLO IS ENGAGED, THE HOST WILL HEAR THE CHANNEL AT THE FADER LEVEL.",
			de:   "Bei aktivem Solo hört der Moderator den Kanal mit dem Faderpegel.",
			en:   "When solo is engaged, the host will hear the channel at the fader level."},
		{title: "PRE-FADER LISTEN", icon: "IconListenPreFader", value: 1,
			desc: "WHEN SOLO IS ENGAGED, THE HOST WILL HEAR THE CHANNEL AT UNITY REGARDLESS OF THE FADER LEVEL.",
			de:   "Bei aktivem Solo hört der Moderator den Kanal unabhängig vom Faderpegel mit Einheitspegel.",
			en:   "When solo is engaged, the host will hear the channel at unity regardless of the fader level."},
	}, func(env *Env) int64 {
		if env.Desk.Bool(desk.OutputNode, 0, "outputPrefader") {
			return 1
		}
		return 0
	}, func(env *Env, v int64) { env.Desk.SetBool(desk.OutputNode, 0, "outputPrefader", v == 1) })
}

// --- Faders: a tile per strip, physical and virtual apart ---

func newFadersPage() Page {
	return &funcPage{title: "FADERS", build: func(env *Env, r image.Rectangle) ui.View {
		c := &ui.Container{}
		c.SetBounds(r)
		th := env.Ctx.Theme
		n := env.Desk.StripCount()
		physical := env.Desk.PhysicalFaders()
		if n == 0 {
			return c
		}
		tileW, tileH, gap := 140, 210, 16
		place := func(strips []int, y int, caption string) {
			lbl := ui.NewLabel(caption, th.Fonts.Body, th.TextDim, font.Center)
			lbl.SetBounds(image.Rect(r.Min.X, y-44, r.Min.X+120, y-4))
			c.Add(lbl)
			total := len(strips)*tileW + (len(strips)-1)*gap
			x := r.Min.X + 140 + max(0, (r.Dx()-140-total)/2)
			for _, s := range strips {
				t := &faderTile{env: env, strip: s}
				t.SetBounds(image.Rect(x, y, x+tileW, y+tileH))
				c.Add(t)
				x += tileW + gap
			}
		}
		var phys, virt []int
		for i := 0; i < n; i++ {
			if i < physical {
				phys = append(phys, i)
			} else {
				virt = append(virt, i)
			}
		}
		if len(virt) == 0 {
			place(phys, r.Min.Y+(r.Dy()-tileH)/2, env.TR("PHYSICAL", "Physisch", "Physical"))
			return c
		}
		place(phys, r.Min.Y+50, env.TR("PHYSICAL", "Physisch", "Physical"))
		place(virt, r.Min.Y+50+tileH+56, env.TR("VIRTUAL", "Virtuell", "Virtual"))
		return c
	}}
}

// faderTile is a strip on the faders page: its icon on a card with the
// source colour along the bottom; a tap opens the channel page.
type faderTile struct {
	ui.Base
	env     *Env
	strip   int
	pressed bool
	last    string
}

func (t *faderTile) CanFocus() bool { return true }
func (t *faderTile) Activate() bool {
	t.env.Nav.Push(NewChannel(t.strip))
	return false
}

func (t *faderTile) Touch(phase ui.TouchPhase, p image.Point) bool {
	switch phase {
	case ui.Down:
		t.pressed = true
		t.MarkDirty()
	case ui.Up:
		if t.pressed {
			t.Activate()
		}
		t.pressed = false
	}
	return true
}

func (t *faderTile) Dirty() []image.Rectangle {
	if s := intText(t.env.Desk.SourceOf(t.strip)) + t.env.Desk.InputColour(t.strip); s != t.last {
		t.last = s
		t.MarkDirty()
	}
	return t.Base.Dirty()
}

func (t *faderTile) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	b := t.Bounds()
	if t.pressed {
		ui.Panel(dst, b, th.Radius, th.PanelHi, clip)
	} else {
		ui.Card(dst, b, th, clip)
	}
	src := t.env.Desk.SourceOf(t.strip)
	name := t.env.Desk.SourceName(src)
	icon := sourceIconBig(name.ID)
	fg := th.Text
	if src < 0 || icon == "" {
		icon, fg = "IconNotAssigned", th.TextMuted
	}
	c.Icon(dst, icon, centred(image.Rect(b.Min.X, b.Min.Y, b.Max.X, b.Max.Y-24), 100, 100), fg)
	colour := th.Text
	if col := t.env.Desk.InputColour(t.strip); col != "" && src >= 0 {
		colour = parseColour(col, th.Text)
	} else if src < 0 {
		colour = th.Line
	}
	ui.Panel(dst, image.Rect(b.Min.X, b.Max.Y-18, b.Max.X, b.Max.Y), 6, colour, clip)
	ui.Fill(dst, image.Rect(b.Min.X, b.Max.Y-18, b.Max.X, b.Max.Y-10), colour, clip)
	if t.Focused() {
		ui.Focus(dst, b, th.Radius, false, th, clip)
	}
}

// --- System ---

// The language, MIDI, date & time, storage and information pages are in
// settings_system.go, settings_datetime.go and settings_info.go.
