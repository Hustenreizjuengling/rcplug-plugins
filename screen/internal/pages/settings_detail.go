package pages

import (
	"image"
	"time"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// --- Display ---

func newBrightnessPage() Page {
	sliders := newSlidersPage("BRIGHTNESS", []sliderSpec{
		{key: "DISPLAY", node: desk.GUINode, prop: "screenBrightness", min: 0, max: 255, text: pct255},
		{key: "BUTTONS", node: desk.GUINode, prop: "activeButtonsBrightness", min: 0, max: 255, text: pct255},
		{key: "BUTTONS INACTIVE", de: "Tasten inaktiv", en: "Buttons inactive", node: desk.GUINode, prop: "inactiveButtonsBrightness", min: 0, max: 255, text: pct255},
	})
	return &funcPage{title: "BRIGHTNESS", build: func(env *Env, r image.Rectangle) ui.View {
		c := &ui.Container{}
		c.SetBounds(r)
		th := env.Ctx.Theme
		c.Add(sliders.Build(env, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+3*(124+th.Gap))))
		// Auto brightness, as the device's switch (a bool since 1.7.6).
		if env.Desk.Has(desk.GUINode, 0, "autoBrightness") {
			rr := image.Rect(r.Min.X, r.Min.Y+3*(124+th.Gap), r.Max.X, r.Min.Y+3*(124+th.Gap)+th.Row)
			card := ui.NewCard()
			card.SetBounds(rr)
			tog := ui.NewToggle(func() bool { return env.Desk.Bool(desk.GUINode, 0, "autoBrightness") },
				func(on bool) { env.Desk.SetBool(desk.GUINode, 0, "autoBrightness", on) })
			tog.Label = env.T("AUTO")
			tog.SetBounds(ui.Inset(rr, 14))
			c.Add(card, tog)
		}
		return c
	}}
}

func pct255(v float64) string { return pct(v / 255) }

func newMeteringPage() Page {
	return newChoicePage("METERING", 2, []radioSpec{
		{title: "DEFAULT", icon: "MeteringDefault", value: 0,
			desc: "LEVEL METERS WILL INCLUDE MARKERS AT REGULAR INTERVALS AND WILL INDICATE UNITY LEVEL.",
			de:   "Die Pegelanzeigen zeigen Markierungen in regelmäßigen Abständen und den Einheitspegel.",
			en:   "Level meters will include markers at regular intervals and will indicate unity level."},
		{title: "BROADCAST", icon: "MeteringBroadcast", value: 1,
			desc: "LEVEL METERS WILL INCLUDE DBFS MARKERS ON EACH CHANNEL. GAIN REDUCTION METER WILL BE VISIBLE FOR EACH CHANNEL.",
			de:   "Die Pegelanzeigen zeigen dBFS-Marken an jedem Kanal. Die Pegelreduktion wird für jeden Kanal angezeigt.",
			en:   "Level meters will include dBFS markers on each channel. Gain reduction will be visible for each channel."},
	}, func(env *Env) int64 { return env.Desk.Int(desk.GUINode, 0, "metering") },
		func(env *Env, v int64) { env.Desk.SetInt(desk.GUINode, 0, "metering", v) })
}

func newHapticsPage() Page {
	return newRadioPage("HAPTICS", desk.SystemNode, 0, "systemHapticSetting", []radioSpec{
		{title: "HAPTIC FEEDBACK ALWAYS ON", icon: "IconHapticsAlways", value: 0},
		{title: "HAPTIC FEEDBACK ON HOLD", icon: "IconHapticsHold", value: 1},
		{title: "NO HAPTIC FEEDBACK", icon: "IconHapticsOff", value: 2},
	})
}

func newRecButtonPage() Page {
	return newRadioPage("REC BUTTON", desk.SystemNode, 0, "systemRecButtonSetting", []radioSpec{
		{title: "PRESS TO PAUSE RECORDING", desc: "HOLD REC TO STOP RECORDING", icon: "IconRecPause", value: 0},
		{title: "PRESS TO STOP RECORDING", desc: "", icon: "IconRecStop", value: 1},
	})
}

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

func newMultitrackPage() Page {
	return &funcPage{title: "MULTITRACK", build: func(env *Env, r image.Rectangle) ui.View {
		c := &ui.Container{}
		c.SetBounds(r)
		th := env.Ctx.Theme
		cl := ui.NewChoiceList([]ui.Choice{
			{Title: env.T("OFF")}, {Title: env.T("PRE-FADER"), Icon: "IconListenPreFader"}, {Title: env.T("POST-FADER"), Icon: "IconListenAfterFader"},
		}, func() int { return int(env.Desk.Int(desk.OutputNode, 0, "recordingMultitrackMode")) },
			func(i int) { env.Desk.SetInt(desk.OutputNode, 0, "recordingMultitrackMode", int64(i)) })
		cl.Columns = 3
		cl.RowH = 200
		cl.SetBounds(image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+200))
		c.Add(cl)
		rr := image.Rect(r.Min.X, r.Min.Y+230, r.Max.X, r.Min.Y+230+th.Row)
		card := ui.NewCard()
		card.SetBounds(rr)
		tog := ui.NewToggle(func() bool { return env.Desk.Bool(desk.OutputNode, 0, "recordingProcessingBypass") },
			func(v bool) { env.Desk.SetBool(desk.OutputNode, 0, "recordingProcessingBypass", v) })
		tog.Label = env.T("BYPASS PROCESSING")
		tog.SetBounds(ui.Inset(rr, 14))
		c.Add(card, tog)
		return c
	}}
}

// --- Faders: a tile per strip, physical and virtual apart ---

func newFadersPage() Page { return newFadersDragPage() }

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

var languages = []struct{ code, label string }{
	{"en", "English"}, {"es", "Español"}, {"fr", "Français"}, {"it", "Italiano"},
	{"pt", "Português"}, {"de", "Deutsch"}, {"ru", "Русский"}, {"ko", "Korean"}, {"ja", "Japanese"}, {"zh", "Chinese"},
}

func newLanguagePage() Page {
	opts := make([]radioSpec, len(languages))
	for i, l := range languages {
		opts[i] = radioSpec{title: l.label, value: int64(i)}
	}
	return newChoicePage("SELECT LANGUAGE", 2, opts, func(env *Env) int64 {
		cur := env.Desk.Str(desk.GUINode, 0, "lang")
		for i, l := range languages {
			if l.code == cur {
				return int64(i)
			}
		}
		return -1
	}, func(env *Env, v int64) {
		env.Desk.SetStr(desk.GUINode, 0, "lang", languages[v].code)
		env.Nav.Toast(env.TR("", "Die Sprache gilt nach einem Neustart der Oberfläche.", "The language applies after the interface restarts."))
	})
}

func newMidiPage() Page {
	return newTogglesPage("MIDI", []toggleRow{
		{key: "MIDI CONTROL", node: desk.SystemNode, prop: "systemMidiControl"},
	})
}

// newDateTimePage shows the clock settings as the device has them; the
// clock itself is the device's and is set on RØDE's screen.
func newDateTimePage() Page {
	return &funcPage{title: "DATE & TIME", build: func(env *Env, r image.Rectangle) ui.View {
		c := &ui.Container{}
		c.SetBounds(r)
		th := env.Ctx.Theme
		now := time.Now()
		fields := []struct{ key, val string }{
			{"YEAR", now.Format("2006")}, {"MONTH", now.Format("01")}, {"DAY", now.Format("02")},
			{"HOURS", now.Format("15")}, {"MINUTES", now.Format("04")}, {"SECONDS", now.Format("05")},
		}
		cells := columns(image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+160), 6, 16)
		for i, f := range fields {
			cell := cells[i]
			card := ui.NewPanelView(th.Black)
			card.SetBounds(image.Rect(cell.Min.X, cell.Min.Y, cell.Max.X, cell.Min.Y+110))
			v := ui.NewLabel(f.val, th.Fonts.Big, th.Text, font.Center)
			v.SetBounds(card.Bounds())
			l := ui.NewLabel(env.T(f.key), th.Fonts.Label, th.Text, font.Center)
			l.SetBounds(image.Rect(cell.Min.X, cell.Min.Y+114, cell.Max.X, cell.Max.Y))
			c.Add(card, v, l)
		}
		rows := []toggleRow{
			{key: "24H", node: desk.SystemNode, prop: "systemDateTime24h"},
			{key: "SHOW DATE AND TIME ON HOME SCREEN", node: desk.SystemNode, prop: "systemDateTimeOnHome"},
			{key: "DAYLIGHT SAVINGS", node: desk.SystemNode, prop: "systemDateTimeDaylightSavings"},
		}
		for i, row := range rows {
			row := row
			rr := image.Rect(r.Min.X, r.Min.Y+176+i*(th.Row+th.Gap), r.Max.X, r.Min.Y+176+i*(th.Row+th.Gap)+th.Row)
			card := ui.NewCard()
			card.SetBounds(rr)
			tog := ui.NewToggle(func() bool { return env.Desk.Bool(row.node, 0, row.prop) },
				func(on bool) { env.Desk.SetBool(row.node, 0, row.prop, on) })
			tog.Label = env.TR(row.key, "Sommerzeit", "Daylight saving time")
			tog.SetBounds(ui.Inset(rr, 14))
			c.Add(card, tog)
		}
		hint := ui.NewLabel(env.TR("", "Uhrzeit und Zeitzone werden auf RØDEs eigenem Bildschirm gestellt.", "The time and the time zone are set on RØDE's own screen."),
			th.Fonts.Small, th.TextDim, font.Center)
		hint.SetBounds(image.Rect(r.Min.X, r.Max.Y-44, r.Max.X, r.Max.Y))
		c.Add(hint)
		return c
	}}
}
