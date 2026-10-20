package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// ChannelPage is the set-up of one strip: its source, input (gain, phantom,
// phase), the simple processing (Depth/Sparkle/Punch) and the way to the
// processors, panning and the fader.
type ChannelPage struct{ strip int }

// NewChannel makes a strip's channel page.
func NewChannel(strip int) Page { return &ChannelPage{strip: strip} }

func (p *ChannelPage) Title() string { return "" } // set from the source below

func (p *ChannelPage) titleText(env *Env) string {
	n := env.Desk.SourceName(env.Desk.SourceOf(p.strip))
	if n.Label == "" {
		return env.T("NOT ASSIGNED")
	}
	return n.Label
}

func (p *ChannelPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	strip := p.strip
	src := env.Desk.SourceOf(strip)
	input := env.Desk.InputIndex(strip)

	// Top caption from the source (the shell shows a back button).
	cap := ui.NewLabel(p.titleText(env), th.Fonts.Title, th.Text, font.Left)
	cap.SetBounds(image.Rect(r.Min.X, r.Min.Y, r.Max.X-200, r.Min.Y+44))
	c.Add(cap)

	assign := ui.NewButton(env.T("CHANNEL ASSIGNMENT"), func() { env.Nav.Push(NewAssign(strip)) })
	assign.Icon = "IconRouting"
	assign.SetBounds(image.Rect(r.Max.X-260, r.Min.Y, r.Max.X, r.Min.Y+48))
	c.Add(assign)

	body := image.Rect(r.Min.X, r.Min.Y+56, r.Max.X, r.Max.Y)
	left, right := columns(body, 2, 16)[0], columns(body, 2, 16)[1]

	// Left column: input controls.
	if src < 0 {
		note := ui.NewLabel(env.T("NOT ASSIGNED"), th.Fonts.Body, th.TextDim, font.Center)
		note.SetBounds(left)
		c.Add(note)
	} else {
		c.Add(p.inputControls(env, input, left)...)
	}

	// Right column: processing.
	c.Add(p.processing(env, strip, right)...)
	return c
}

// inputControls builds gain, phantom and phase for a source.
func (p *ChannelPage) inputControls(env *Env, input int, area image.Rectangle) []ui.View {
	th := env.Ctx.Theme
	var out []ui.View
	rows := rowRects(area, 72, 10)
	if len(rows) == 0 {
		return out
	}
	lbl := ui.NewLabel(env.T("GAIN"), th.Fonts.Label, th.TextDim, font.Left)
	lbl.SetBounds(image.Rect(area.Min.X, area.Min.Y-4, area.Max.X, area.Min.Y+20))
	out = append(out, lbl)

	// Which gain property this source uses.
	gainProp := "inputMicrophoneGain"
	if !env.Desk.Has(desk.InputNode, input, gainProp) {
		if env.Desk.Has(desk.InputNode, input, "inputDigitalGain") {
			gainProp = "inputDigitalGain"
		} else if env.Desk.Has(desk.InputNode, input, "inputInstrumentGain") {
			gainProp = "inputInstrumentGain"
		}
	}
	gp := gainProp
	gain := ui.NewStepper(func() float64 { return float64(env.Desk.Int(desk.InputNode, input, gp)) },
		func(d int) {
			v := env.Desk.Int(desk.InputNode, input, gp) + int64(d)
			env.Desk.SetInt(desk.InputNode, input, gp, v)
		})
	gain.Label = env.T("GAIN")
	gain.Text = func(v float64) string { return dbText(v) }
	gain.SetBounds(rows[0])
	out = append(out, gain)

	ri := 1
	// Phantom power (only on combo inputs: inputPower not −1).
	if env.Desk.Has(desk.InputNode, input, "inputPower") && env.Desk.Int(desk.InputNode, input, "inputPower") >= 0 && ri < len(rows) {
		row := rows[ri]
		ri++
		card := ui.NewPanelView(th.Panel)
		card.SetBounds(row)
		out = append(out, card)
		t := ui.NewLabel("P48", th.Fonts.Body, th.Text, font.Left)
		t.SetBounds(image.Rect(row.Min.X+16, row.Min.Y, row.Max.X-90, row.Max.Y))
		tog := ui.NewToggle(func() bool { return env.Desk.Int(desk.InputNode, input, "inputPower") == 1 },
			func(on bool) {
				v := int64(0)
				if on {
					v = 1
				}
				env.Desk.SetInt(desk.InputNode, input, "inputPower", v)
			})
		tog.SetBounds(image.Rect(row.Max.X-80, row.Min.Y, row.Max.X-8, row.Max.Y))
		out = append(out, card, t, tog)
	}
	// Phase flip.
	if env.Desk.Has(desk.InputNode, input, "inputPhaseFlip") && ri < len(rows) {
		row := rows[ri]
		ri++
		card := ui.NewPanelView(th.Panel)
		card.SetBounds(row)
		t := ui.NewLabel("Ø", th.Fonts.Body, th.Text, font.Left)
		t.SetBounds(image.Rect(row.Min.X+16, row.Min.Y, row.Max.X-90, row.Max.Y))
		tog := ui.NewToggle(func() bool { return env.Desk.Bool(desk.InputNode, input, "inputPhaseFlip") },
			func(on bool) { env.Desk.SetBool(desk.InputNode, input, "inputPhaseFlip", on) })
		tog.SetBounds(image.Rect(row.Max.X-80, row.Min.Y, row.Max.X-8, row.Max.Y))
		out = append(out, card, t, tog)
	}
	return out
}

// processing builds the simple processing (Depth/Sparkle/Punch) and the row
// of processors.
func (p *ChannelPage) processing(env *Env, strip int, area image.Rectangle) []ui.View {
	th := env.Ctx.Theme
	var out []ui.View

	head := image.Rect(area.Min.X, area.Min.Y, area.Max.X, area.Min.Y+48)
	card := ui.NewPanelView(th.Panel)
	card.SetBounds(head)
	t := ui.NewLabel(env.T("PROCESSING"), th.Fonts.Label, th.Text, font.Left)
	t.SetBounds(image.Rect(head.Min.X+16, head.Min.Y, head.Max.X-90, head.Max.Y))
	// channelBypassProcessing is inverted: true = off.
	tog := ui.NewToggle(func() bool { return !env.Desk.Bool(desk.ChannelNode, strip, "channelBypassProcessing") },
		func(on bool) { env.Desk.SetBool(desk.ChannelNode, strip, "channelBypassProcessing", !on) })
	tog.SetBounds(image.Rect(head.Max.X-80, head.Min.Y, head.Max.X-8, head.Max.Y))
	out = append(out, card, t, tog)

	// Depth / Sparkle / Punch.
	knobRow := image.Rect(area.Min.X, head.Max.Y+10, area.Max.X, head.Max.Y+10+150)
	kc := columns(knobRow, 3, 10)
	for i, spec := range []struct{ key, prop string }{{"DEPTH", "channelDepth"}, {"SPARKLE", "channelSparkle"}, {"PUNCH", "channelPunch"}} {
		prop := spec.prop
		k := ui.NewKnob(func() float64 { return env.Desk.Num(desk.ChannelNode, strip, prop) },
			func(v float64) { env.Desk.SetFloat(desk.ChannelNode, strip, prop, v) })
		k.Label = env.T(spec.key)
		k.Text = func(v float64) string { return pct(v) }
		k.SetBounds(kc[i])
		out = append(out, k)
	}

	// Processors grid.
	procArea := image.Rect(area.Min.X, knobRow.Max.Y+12, area.Max.X, area.Max.Y-60)
	procs := []struct {
		key  string
		icon string
		make func() Page
	}{
		{"HIGH-PASS FILTER", "procHighPass", func() Page { return newHPF(strip) }},
		{"NOISE GATE", "procNoiseGate", func() Page { return newGate(strip) }},
		{"COMPRESSOR", "procCompressor", func() Page { return newComp(strip) }},
		{"DE-ESSER", "procDeesser", func() Page { return newDeEsser(strip) }},
		{"EQUALIZER", "procEq", func() Page { return newEQ(strip) }},
		{"EXCITER", "procAuralExciter", func() Page { return newAphex(strip) }},
		{"PANNING", "procLR", func() Page { return newPanning(strip) }},
	}
	cells := grid(procArea, 4, 2, 8)
	for i, pr := range procs {
		if i >= len(cells) {
			break
		}
		mk := pr.make
		b := ui.NewButton(env.T(pr.key), func() { env.Nav.Push(mk()) })
		b.Icon = pr.icon
		b.TileLayout = true
		b.SetBounds(cells[i])
		out = append(out, b)
	}

	// Fader at the very bottom (virtual-strip convenience).
	foot := image.Rect(area.Min.X, area.Max.Y-52, area.Max.X, area.Max.Y)
	fl := ui.NewSlider(func() float64 { return env.Desk.FaderFrac(strip) },
		func(v float64) { env.Desk.SetFaderFrac(strip, v) })
	fl.Label = env.T("FADER")
	fl.Unity = env.Desk.UnityFrac()
	fl.Text = func(v float64) string { return dbText(env.Desk.FaderDbFrac(v)) }
	fl.SetBounds(foot)
	out = append(out, fl)
	return out
}
