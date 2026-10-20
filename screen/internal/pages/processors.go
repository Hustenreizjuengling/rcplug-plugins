package pages

import (
	"image"
	"math"
	"strconv"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// A processor page: a title, an on/off toggle and a set of knobs (and the
// odd band toggle) bound to the channel's processing properties. The desk
// stores each processing parameter as a double 0..1; the format functions
// turn that into the device's units for display. The scales follow the
// catalog (approximate where the taper is not published).

type procKind int

const (
	knobParam procKind = iota
	toggleParam
)

type procParam struct {
	key  string
	prop string
	kind procKind
	fmt  func(float64) string
}

type processorPage struct {
	strip    int
	titleKey string
	onProp   string
	params   []procParam
}

func (p *processorPage) Title() string { return "" }

func (p *processorPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	strip := p.strip

	// Header: the processor name + its on/off toggle.
	head := image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+56)
	card := ui.NewPanelView(th.Panel)
	card.SetBounds(head)
	name := ui.NewLabel(env.T(p.titleKey), th.Fonts.Title, th.Text, font.Left)
	name.SetBounds(image.Rect(head.Min.X+16, head.Min.Y, head.Max.X-100, head.Max.Y))
	c.Add(card, name)
	if p.onProp != "" {
		on := p.onProp
		tog := ui.NewToggle(func() bool { return env.Desk.Bool(desk.ChannelNode, strip, on) },
			func(v bool) { env.Desk.SetBool(desk.ChannelNode, strip, on, v) })
		tog.SetBounds(image.Rect(head.Max.X-88, head.Min.Y+10, head.Max.X-12, head.Max.Y-10))
		c.Add(tog)
	}

	// Split params into toggles and knobs.
	var toggles, knobs []procParam
	for _, pr := range p.params {
		if pr.kind == toggleParam {
			toggles = append(toggles, pr)
		} else {
			knobs = append(knobs, pr)
		}
	}

	y := head.Max.Y + 10
	if len(toggles) > 0 {
		row := image.Rect(r.Min.X, y, r.Max.X, y+48)
		cols := columns(row, len(toggles), 8)
		for i, pr := range toggles {
			prop := pr.prop
			cell := cols[i]
			bg := ui.NewPanelView(th.Panel)
			bg.SetBounds(cell)
			lbl := ui.NewLabel(env.T(pr.key), th.Fonts.Small, th.Text, font.Left)
			lbl.SetBounds(image.Rect(cell.Min.X+12, cell.Min.Y, cell.Max.X-70, cell.Max.Y))
			tog := ui.NewToggle(func() bool { return env.Desk.Bool(desk.ChannelNode, strip, prop) },
				func(v bool) { env.Desk.SetBool(desk.ChannelNode, strip, prop, v) })
			tog.SetBounds(image.Rect(cell.Max.X-64, cell.Min.Y+8, cell.Max.X-8, cell.Max.Y-8))
			c.Add(bg, lbl, tog)
		}
		y = row.Max.Y + 10
	}

	// Knob grid.
	if len(knobs) > 0 {
		area := image.Rect(r.Min.X, y, r.Max.X, r.Max.Y)
		cols := 4
		if len(knobs) < 4 {
			cols = len(knobs)
		}
		rows := (len(knobs) + cols - 1) / cols
		cells := grid(area, cols, rows, 10)
		for i, pr := range knobs {
			if i >= len(cells) {
				break
			}
			prop, fmtFn := pr.prop, pr.fmt
			k := ui.NewKnob(func() float64 { return env.Desk.Num(desk.ChannelNode, strip, prop) },
				func(v float64) { env.Desk.SetFloat(desk.ChannelNode, strip, prop, v) })
			k.Label = env.T(pr.key)
			if fmtFn != nil {
				k.Text = fmtFn
			}
			k.SetBounds(cells[i])
			c.Add(k)
		}
	}
	return c
}

// --- scale helpers: fraction 0..1 → the device's display unit ---

func lin(minV, maxV float64) func(float64) string {
	return func(v float64) string { return dbText(minV + v*(maxV-minV)) }
}

// hz shows a frequency on an approximate log scale min..max.
func hz(minV, maxV float64) func(float64) string {
	return func(v float64) string {
		f := minV * math.Pow(maxV/minV, v)
		if f >= 1000 {
			return strconv.FormatFloat(f/1000, 'f', 1, 64) + " kHz"
		}
		return strconv.Itoa(int(f+0.5)) + " Hz"
	}
}

func ms(minV, maxV float64) func(float64) string {
	return func(v float64) string {
		t := minV * math.Pow(maxV/minV, v)
		if t >= 1000 {
			return strconv.FormatFloat(t/1000, 'f', 2, 64) + " s"
		}
		return strconv.FormatFloat(t, 'f', 1, 64) + " ms"
	}
}

func ratio(minV, maxV float64) func(float64) string {
	return func(v float64) string {
		return strconv.FormatFloat(minV+v*(maxV-minV), 'f', 1, 64) + ":1"
	}
}

// bipolar maps 0..1 to −span..+span dB (0.5 = 0 dB).
func bipolar(span float64) func(float64) string {
	return func(v float64) string { return dbText((v - 0.5) * 2 * span) }
}

// --- the processors, from the catalog (CHANNEL[strip]) ---

func newHPF(strip int) Page {
	return &processorPage{strip: strip, titleKey: "HIGH-PASS FILTER", onProp: "hpfOn", params: []procParam{
		{key: "FREQUENCY", prop: "hpfFrequency", fmt: hz(20, 200)},
		{key: "SLOPE", prop: "hpfSlope", fmt: func(v float64) string { return strconv.Itoa(6+int(v*3+0.5)*6) + " dB/oct" }},
	}}
}

func newGate(strip int) Page {
	return &processorPage{strip: strip, titleKey: "NOISE GATE", onProp: "noiseGateOn", params: []procParam{
		{key: "THRESHOLD", prop: "noiseGateThreshold", fmt: lin(-60, 0)},
		{key: "RANGE", prop: "noiseGateRange", fmt: lin(0, 100)},
		{key: "HYSTERESIS", prop: "noiseGateHysteresis", fmt: lin(0, 30)},
		{key: "ATTACK", prop: "noiseGateAttack", fmt: ms(0.1, 1000)},
		{key: "HOLD", prop: "noiseGateHold", fmt: ms(50, 2000)},
		{key: "RELEASE", prop: "noiseGateRelease", fmt: ms(50, 2000)},
	}}
}

func newComp(strip int) Page {
	return &processorPage{strip: strip, titleKey: "COMPRESSOR", onProp: "compressorOn", params: []procParam{
		{key: "THRESHOLD", prop: "compressorThreshold", fmt: lin(-60, 0)},
		{key: "RATIO", prop: "compressorRatio", fmt: ratio(1.5, 4.5)},
		{key: "ATTACK", prop: "compressorAttack", fmt: ms(0.1, 10)},
		{key: "RELEASE", prop: "compressorRelease", fmt: ms(5, 200)},
		{key: "GAIN", prop: "compressorGain", fmt: lin(0, 9)},
	}}
}

func newDeEsser(strip int) Page {
	return &processorPage{strip: strip, titleKey: "DE-ESSER", onProp: "deesserOn", params: []procParam{
		{key: "FREQUENCY", prop: "deesserFrequency", fmt: hz(1000, 8000)},
		{key: "THRESHOLD", prop: "deesserThreshold", fmt: lin(-60, 0)},
		{key: "RATIO", prop: "deesserRatio", fmt: ratio(1, 10)},
		{key: "ATTACK", prop: "deesserAttack", fmt: ms(0.1, 10)},
		{key: "RELEASE", prop: "deesserRelease", fmt: ms(50, 2000)},
		{key: "GAIN", prop: "deesserGain", fmt: lin(0, 9)},
	}}
}

func newEQ(strip int) Page {
	return &processorPage{strip: strip, titleKey: "EQUALIZER", onProp: "eqOn", params: []procParam{
		{key: "L", prop: "eqLowOn", kind: toggleParam},
		{key: "M", prop: "eqMidOn", kind: toggleParam},
		{key: "H", prop: "eqHighOn", kind: toggleParam},
		{key: "LOW GAIN", prop: "eqLowGain", fmt: bipolar(12)},
		{key: "LOW BELL", prop: "eqLowShelf", fmt: hz(20, 200)},
		{key: "MID GAIN", prop: "eqMidGain", fmt: bipolar(12)},
		{key: "MID BELL", prop: "eqMidBell", fmt: hz(200, 2000)},
		{key: "HIGH GAIN", prop: "eqHighGain", fmt: bipolar(12)},
		{key: "HIGH BELL", prop: "eqHighBell", fmt: hz(2000, 12000)},
	}}
}

func newAphex(strip int) Page {
	return &processorPage{strip: strip, titleKey: "EXCITER", onProp: "aphexOn", params: []procParam{
		{key: "HARMONICS", prop: "aphexAEMix", fmt: pctFmt},
		{key: "TUNE", prop: "aphexAETune", fmt: hz(600, 5000)},
		{key: "DRIVE", prop: "aphexBBDrive", fmt: pctFmt},
		{key: "BB TUNE", prop: "aphexBBTune", fmt: hz(60, 312)},
	}}
}

func pctFmt(v float64) string { return pct(v) }

// PanningPage is a slider between left and right (and a stereo pair).
type PanningPage struct{ strip int }

func newPanning(strip int) Page { return &PanningPage{strip: strip} }

func (p *PanningPage) Title() string { return "" }

func (p *PanningPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	strip := p.strip

	head := image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+56)
	card := ui.NewPanelView(th.Panel)
	card.SetBounds(head)
	name := ui.NewLabel(env.T("PANNING"), th.Fonts.Title, th.Text, font.Left)
	name.SetBounds(image.Rect(head.Min.X+16, head.Min.Y, head.Max.X-100, head.Max.Y))
	tog := ui.NewToggle(func() bool { return env.Desk.Bool(desk.ChannelNode, strip, "channelPanOn") },
		func(v bool) { env.Desk.SetBool(desk.ChannelNode, strip, "channelPanOn", v) })
	tog.SetBounds(image.Rect(head.Max.X-88, head.Min.Y+10, head.Max.X-12, head.Max.Y-10))
	c.Add(card, name, tog)

	s := ui.NewSlider(func() float64 { return env.Desk.Num(desk.ChannelNode, strip, "channelPan") },
		func(v float64) { env.Desk.SetFloat(desk.ChannelNode, strip, "channelPan", v) })
	s.Label = env.T("PAN")
	s.Unity = 0.5
	s.Text = panText
	s.SetBounds(image.Rect(r.Min.X+20, head.Max.Y+40, r.Max.X-20, head.Max.Y+110))
	c.Add(s)
	return c
}

func panText(v float64) string {
	d := int((v - 0.5) * 200)
	switch {
	case d == 0:
		return "C"
	case d < 0:
		return "L" + strconv.Itoa(-d)
	default:
		return "R" + strconv.Itoa(d)
	}
}
