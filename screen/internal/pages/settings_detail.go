package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// --- Display ---

func newBrightnessPage() Page {
	return newSlidersPage("BRIGHTNESS", []sliderSpec{
		{key: "DISPLAY", node: desk.GUINode, prop: "screenBrightness", min: 0, max: 255},
		{key: "BUTTONS", node: desk.GUINode, prop: "activeButtonsBrightness", min: 0, max: 255},
		{key: "BUTTONS INACTIVE", node: desk.GUINode, prop: "inactiveButtonsBrightness", min: 0, max: 255},
	})
}

func newMeteringPage() Page {
	return newRadioPage("METERING", desk.GUINode, 0, "metering", []radioSpec{
		{title: "DEFAULT", value: 0},
		{title: "BROADCAST", value: 1},
	})
}

func newHapticsPage() Page {
	return newRadioPage("HAPTICS", desk.SystemNode, 0, "systemHapticSetting", []radioSpec{
		{title: "HAPTIC FEEDBACK ALWAYS ON", value: 0},
		{title: "HAPTIC FEEDBACK ON HOLD", value: 1},
		{title: "NO HAPTIC FEEDBACK", value: 2},
	})
}

func newRecButtonPage() Page {
	return newRadioPage("REC BUTTON", desk.SystemNode, 0, "systemRecButtonSetting", []radioSpec{
		{title: "PRESS TO PAUSE RECORDING", value: 0},
		{title: "PRESS TO STOP RECORDING", value: 1},
	})
}

// --- Outputs ---

func newMonitorPage() Page {
	return newTogglesPage("MONITOR", []toggleRow{
		{key: "AUTO-MUTE MONITOR OUTPUT", node: desk.OutputNode, prop: "outputMonAutoMute"},
		{key: "AUTO-MUTE BLUETOOTH OUTPUT", node: desk.OutputNode, prop: "outputBTAutoMute"},
		{key: "FIXED OUTPUT LEVEL", node: desk.OutputNode, prop: "outputMonFixed"},
	})
}

func newListenPage() Page {
	return newRadioPage("LISTEN", desk.OutputNode, 0, "outputPrefader", []radioSpec{
		{title: "AFTER-FADER LISTEN", value: 0},
		{title: "PRE-FADER LISTEN", value: 1},
	})
}

func newMultitrackPage() Page {
	return &funcPage{title: "MULTITRACK", build: func(env *Env, r image.Rectangle) ui.View {
		c := &ui.Container{}
		c.SetBounds(r)
		th := env.Ctx.Theme
		lbl := ui.NewLabel(env.T("RECORDING"), th.Fonts.Label, th.TextDim, font.Left)
		lbl.SetBounds(image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+28))
		c.Add(lbl)
		rl := ui.NewRadioList([]ui.RadioOption{
			{Title: env.T("OFF")}, {Title: env.T("PRE-FADER")}, {Title: env.T("POST-FADER")},
		}, func() int { return int(env.Desk.Int(desk.OutputNode, 0, "recordingMultitrackMode")) },
			func(i int) { env.Desk.SetInt(desk.OutputNode, 0, "recordingMultitrackMode", int64(i)) })
		rl.SetBounds(image.Rect(r.Min.X, r.Min.Y+32, r.Max.X, r.Min.Y+32+3*76))
		c.Add(rl)
		// Bypass processing toggle.
		br := image.Rect(r.Min.X, r.Min.Y+32+3*76+12, r.Max.X, r.Min.Y+32+3*76+12+64)
		card := ui.NewPanelView(th.Panel)
		card.SetBounds(br)
		t := ui.NewLabel(env.T("BYPASS PROCESSING"), th.Fonts.Body, th.Text, font.Left)
		t.SetBounds(image.Rect(br.Min.X+16, br.Min.Y, br.Max.X-90, br.Max.Y))
		tog := ui.NewToggle(func() bool { return env.Desk.Bool(desk.OutputNode, 0, "recordingProcessingBypass") },
			func(v bool) { env.Desk.SetBool(desk.OutputNode, 0, "recordingProcessingBypass", v) })
		tog.SetBounds(image.Rect(br.Max.X-84, br.Min.Y+12, br.Max.X-12, br.Max.Y-12))
		c.Add(card, t, tog)
		return c
	}}
}

func newMasterProcPage() Page {
	return &funcPage{title: "PROCESSING", build: func(env *Env, r image.Rectangle) ui.View {
		c := &ui.Container{}
		c.SetBounds(r)
		th := env.Ctx.Theme
		head := image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+56)
		card := ui.NewPanelView(th.Panel)
		card.SetBounds(head)
		name := ui.NewLabel(env.T("MASTER COMPELLOR"), th.Fonts.Title, th.Text, font.Left)
		name.SetBounds(image.Rect(head.Min.X+16, head.Min.Y, head.Max.X-100, head.Max.Y))
		tog := ui.NewToggle(func() bool { return env.Desk.Bool(desk.MasterNode, 0, "masterCompellorOn") },
			func(v bool) { env.Desk.SetBool(desk.MasterNode, 0, "masterCompellorOn", v) })
		tog.SetBounds(image.Rect(head.Max.X-88, head.Min.Y+10, head.Max.X-12, head.Max.Y-10))
		c.Add(card, name, tog)
		knobs := []struct{ key, prop string }{
			{"THRESHOLD", "masterCompellorThreshold"}, {"ATTACK", "masterCompellorAttack"},
			{"RELEASE", "masterCompellorRelease"}, {"GAIN", "masterCompellorGain"},
		}
		cells := grid(image.Rect(r.Min.X, head.Max.Y+12, r.Max.X, head.Max.Y+12+150), 4, 1, 10)
		for i, k := range knobs {
			prop := k.prop
			kb := ui.NewKnob(func() float64 { return env.Desk.Num(desk.MasterNode, 0, prop) },
				func(v float64) { env.Desk.SetFloat(desk.MasterNode, 0, prop, v) })
			kb.Label = env.T(k.key)
			kb.Text = pctFmt
			kb.SetBounds(cells[i])
			c.Add(kb)
		}
		// Output delay.
		dr := image.Rect(r.Min.X, head.Max.Y+12+160, r.Max.X, head.Max.Y+12+160+64)
		dcard := ui.NewPanelView(th.Panel)
		dcard.SetBounds(dr)
		dl := ui.NewLabel(env.T("OUTPUT DELAY"), th.Fonts.Body, th.Text, font.Left)
		dl.SetBounds(image.Rect(dr.Min.X+16, dr.Min.Y, dr.Max.X-90, dr.Max.Y))
		dtog := ui.NewToggle(func() bool { return env.Desk.Bool(desk.MasterNode, 0, "masterDelayOn") },
			func(v bool) { env.Desk.SetBool(desk.MasterNode, 0, "masterDelayOn", v) })
		dtog.SetBounds(image.Rect(dr.Max.X-84, dr.Min.Y+12, dr.Max.X-12, dr.Max.Y-12))
		c.Add(dcard, dl, dtog)
		return c
	}}
}

func newFadersPage() Page {
	return &funcPage{title: "FADERS", build: func(env *Env, r image.Rectangle) ui.View {
		c := &ui.Container{}
		c.SetBounds(r)
		th := env.Ctx.Theme
		n := env.Desk.StripCount()
		if n == 0 {
			return c
		}
		cells := grid(r, 3, (n+2)/3, 10)
		for i := 0; i < n; i++ {
			if i >= len(cells) {
				break
			}
			strip := i
			s := env.Desk.Strip(i)
			src := env.Desk.SourceName(env.Desk.SourceOf(i))
			label := s.Label
			if src.Label != "" {
				label = src.Label
			}
			b := ui.NewButton(label, func() { env.Nav.Push(NewChannel(strip)) })
			b.TileLayout = true
			b.Icon = sourceIcon(src.ID)
			b.SetBounds(cells[i])
			c.Add(b)
		}
		_ = th
		return c
	}}
}

// --- System ---

func newLanguagePage() Page {
	langs := []struct{ code, label string }{
		{"en", "English"}, {"de", "Deutsch"}, {"es", "Español"}, {"fr", "Français"},
		{"it", "Italiano"}, {"pt", "Português"}, {"ru", "Русский"}, {"ja", "日本語"}, {"zh", "中文"},
	}
	return &funcPage{title: "LANGUAGE", build: func(env *Env, r image.Rectangle) ui.View {
		c := &ui.Container{}
		c.SetBounds(r)
		opts := make([]ui.RadioOption, len(langs))
		for i, l := range langs {
			opts[i] = ui.RadioOption{Title: l.label}
		}
		rl := ui.NewRadioList(opts, func() int {
			cur := env.Desk.Str(desk.GUINode, 0, "lang")
			for i, l := range langs {
				if l.code == cur {
					return i
				}
			}
			return -1
		}, func(i int) { env.Desk.SetStr(desk.GUINode, 0, "lang", langs[i].code) })
		rl.SetBounds(r)
		c.Add(rl)
		return c
	}}
}

func newMidiPage() Page {
	return newTogglesPage("MIDI", []toggleRow{
		{key: "MIDI CONTROL", node: desk.SystemNode, prop: "systemMidiControl"},
	})
}
