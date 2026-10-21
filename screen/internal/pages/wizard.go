package pages

import (
	"image"
	"strings"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// The setup assistant (RØDE's SetupAssistantPage, 1.7.6), reached after the
// welcome flow and from a new show. It fills the screen over the wizard
// background; the cross at the top left leaves it, the blue arrow at the
// bottom right goes on (there is no way back, as on the device). The steps
// and their texts follow the screen program's step function
// (rc_gui_juce FUN_0082bfd0): intro, the hearing warning, faders to unity,
// headphones (three Getting Started pictures), what the combo inputs take,
// the input selection, then per physical input the microphone, a wireless
// transmitter, the gain and a preset, then Bluetooth, the USB inputs, the
// channel assignment, the SMART pads, shows, the show's name, ready.

// Page ids of the first-run flow.
const (
	IDWelcome = "welcome"
	IDWizard  = "wizard"
)

func init() {
	register(IDWizard, func() Page { return NewSetupAssistant(nil) })
}

// wizardStep is one screen of the assistant, numbered as the device's modes.
type wizardStep int

const (
	stepIntro       wizardStep = 0
	stepWarning     wizardStep = 1
	stepUnity       wizardStep = 2
	stepPhonesMin   wizardStep = 3
	stepPhonesPlug  wizardStep = 4
	stepPhonesLevel wizardStep = 5
	stepCombo       wizardStep = 6
	stepInputs      wizardStep = 7
	stepPerInput    wizardStep = 8 // four steps per input: mic, wireless, gain, preset
	stepBluetooth   wizardStep = 0x18
	stepUSB         wizardStep = 0x19
	stepChannels    wizardStep = 0x1a
	stepPads        wizardStep = 0x1b
	stepShows       wizardStep = 0x1c
	stepName        wizardStep = 0x1d
	stepDone        wizardStep = 0x1e
)

// The per-input sub-steps.
const (
	subMic = iota
	subWireless
	subGain
	subPreset
)

// SetupAssistantPage runs the assistant. done runs when the user finishes
// or leaves (nil: back to the mixer).
type SetupAssistantPage struct {
	step wizardStep
	done func()

	// The input selection's choices per physical strip: the source to carry
	// (written when the step is left with the arrow).
	choice map[int]int64
}

// NewSetupAssistant makes the assistant at its first step.
func NewSetupAssistant(done func()) Page {
	return &SetupAssistantPage{done: done, choice: map[int]int64{}}
}

// Step is the current step, numbered as the device's modes (for tests).
func (p *SetupAssistantPage) Step() int { return int(p.step) }

// SetStep jumps to a step (previews and tests).
func (p *SetupAssistantPage) SetStep(n int) {
	if n >= int(stepIntro) && n <= int(stepDone) {
		p.step = wizardStep(n)
	}
}

func (p *SetupAssistantPage) Title() string   { return "SETUP WIZARD" }
func (p *SetupAssistantPage) FullBleed() bool { return true }

// finish leaves the assistant.
func (p *SetupAssistantPage) finish(env *Env) {
	p.discoverable(env, false)
	if p.done != nil {
		p.done()
		return
	}
	env.Nav.Home()
}

// inputOf is the physical input (0…3) and sub-step of a per-input step.
func (p *SetupAssistantPage) inputOf() (input, sub int) {
	n := int(p.step - stepPerInput)
	return n / 4, n % 4
}

// inputSteps is how many physical inputs the assistant walks through: the
// desk's combo inputs (2 on a Duo, 4 on a Pro II).
func inputSteps(env *Env) int {
	if n := env.Desk.MicInputs(); n > 0 {
		return min(n, 4)
	}
	return min(env.Desk.PhysicalFaders(), 4)
}

// wants reports whether a per-input step applies: the microphone choice
// only to a combo input with a microphone type, the wireless pairing only
// to a wireless channel, gain and preset to anything but a wireless
// channel.
func (p *SetupAssistantPage) wants(env *Env, input, sub int) bool {
	if input >= inputSteps(env) {
		return false
	}
	src := env.Desk.SourceOf(input)
	wireless := src == 19 || src == 20
	headset := src == 2 && isDuo(env)
	switch sub {
	case subMic:
		return src >= 0 && src <= 6 && !headset
	case subWireless:
		return wireless
	default:
		return src >= 0 && !wireless
	}
}

// next moves on, skipping the per-input steps that do not apply.
func (p *SetupAssistantPage) next(env *Env) {
	if p.step == stepInputs {
		p.applyInputs(env)
	}
	if p.step >= stepDone {
		p.finish(env)
		return
	}
	p.step++
	for p.step >= stepPerInput && p.step < stepBluetooth {
		in, sub := p.inputOf()
		if p.wants(env, in, sub) {
			break
		}
		p.step++
	}
	if p.step == stepBluetooth {
		// Entering the step makes the desk discoverable, as the pairing
		// page does; finish turns it off again.
		p.discoverable(env, true)
	}
	env.Nav.Refresh()
}

// discoverable switches the desk's Bluetooth visibility (NETWORK.btVisible).
func (p *SetupAssistantPage) discoverable(env *Env, on bool) {
	d := env.Desk
	if d.Has(desk.NetworkNode, 0, "btVisible") && d.Bool(desk.NetworkNode, 0, "btVisible") != on {
		d.SetBool(desk.NetworkNode, 0, "btVisible", on)
	}
}

func (p *SetupAssistantPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	if p.step == stepWarning {
		// The hearing warning: a red screen.
		red := &gradientView{top: gfx.Hex(0x5f0000), bottom: gfx.Hex(0xc70000)}
		red.SetBounds(r)
		c.Add(red)
	} else {
		bg := &pictureView{name: "WizardBackground"}
		bg.SetBounds(r)
		c.Add(bg)
	}
	body := image.Rect(r.Min.X+40, r.Min.Y+36, r.Max.X-40, r.Max.Y-130)
	switch {
	case p.step == stepIntro:
		c.Add(p.intro(env, r)...)
	case p.step == stepWarning:
		c.Add(p.warning(env, r)...)
	case p.step == stepUnity:
		c.Add(p.gettingStarted(env, body, "FadersAtUnity"+deviceSuffix(env), 560, 424,
			env.T("SET ALL FADERS TO 'UNITY' INDICATED BY THE BOLD LINE"))...)
	case p.step == stepPhonesMin:
		c.Add(p.gettingStarted(env, body, "GettingStarted"+deviceSuffix(env)+"A", 560, 165,
			env.T("ENSURE THAT ALL HEADPHONES ARE SET TO THE MINIMUM LEVEL."))...)
	case p.step == stepPhonesPlug:
		c.Add(p.gettingStarted(env, body, "GettingStartedB", 342, 238, env.T("CONNECT YOUR HEADPHONES AND TAP NEXT"))...)
	case p.step == stepPhonesLevel:
		text := strings.ReplaceAll(env.T("SLOWLY INCREASE THE LEVEL UNTIL YOU CAN HEAR THE [SOUND] AT A COMFORTABLE LEVEL."), "[SOUND]", strings.ToLower(env.T("SOUND"))) +
			"\n" + env.T("TAP NEXT WHEN READY.")
		c.Add(p.gettingStarted(env, body, "GettingStartedC", 350, 197, text)...)
	case p.step == stepCombo:
		c.Add(p.comboInputs(env, body)...)
	case p.step == stepInputs:
		c.Add(p.inputSelection(env, body)...)
	case p.step >= stepPerInput && p.step < stepBluetooth:
		in, sub := p.inputOf()
		switch sub {
		case subMic:
			c.Add(p.micStep(env, body, in)...)
		case subWireless:
			c.Add(p.wirelessStep(env, body, in)...)
		case subGain:
			c.Add(p.gainStep(env, body, in)...)
		default:
			c.Add(p.presetStep(env, body, in)...)
		}
	case p.step == stepBluetooth:
		c.Add(p.bluetoothStep(env, body)...)
	case p.step == stepUSB:
		c.Add(p.usbStep(env, body)...)
	case p.step == stepChannels:
		c.Add(wizardTitle(env, env.T("CHANNEL ASSIGNMENT"), body),
			Detail("faders").Build(env, image.Rect(body.Min.X, body.Min.Y+80, body.Max.X, body.Max.Y)))
	case p.step == stepPads:
		c.Add(p.padsStep(env, body)...)
	case p.step == stepShows:
		c.Add(p.showsStep(env, body)...)
	case p.step == stepName:
		c.Add(p.nameStep(env, body)...)
	default:
		c.Add(p.doneStep(env, r)...)
	}
	c.Add(p.navigation(env, r)...)
	return c
}

// navigation: the cross and the arrow.
func (p *SetupAssistantPage) navigation(env *Env, r image.Rectangle) []ui.View {
	th := env.Ctx.Theme
	x := &pictureButton{name: "IconSetupAssistantClose", onTap: func() { p.finish(env) }}
	x.SetBounds(image.Rect(r.Min.X+20, r.Min.Y+20, r.Min.X+90, r.Min.Y+90))
	n := ui.NewButton("", func() { p.next(env) })
	n.Icon, n.Style, n.Fill = "arrow_rounded_right", ui.StyleRound, th.Blue
	n.SetBounds(image.Rect(r.Max.X-140, r.Max.Y-140, r.Max.X-30, r.Max.Y-30))
	return []ui.View{x, n}
}

// deviceSuffix picks the device's variant of a picture (…DUO / …RCP2).
func deviceSuffix(env *Env) string {
	if isDuo(env) {
		return "DUO"
	}
	return "RCP2"
}

// deviceLogo is the device's name as a picture.
func deviceLogo(env *Env) string {
	if isDuo(env) {
		return "IconRodecasterDuoLogo"
	}
	return "IconRodecasterProLogo"
}

// intro: the wizard hat and the two sentences.
func (p *SetupAssistantPage) intro(env *Env, r image.Rectangle) []ui.View {
	th := env.Ctx.Theme
	hat := ui.NewIcon("IconWizard", th.Text)
	hat.SetBounds(centred(image.Rect(r.Min.X, r.Min.Y+100, r.Max.X, r.Min.Y+270), 120, 150))
	text := strings.ReplaceAll(env.T("THE SETUP WIZARD WILL PROVIDE STEP-BY-STEP\n INSTRUCTIONS TO CONFIGURE YOUR DEVICE."), "\n ", " ") +
		"\n\n" + env.T("THE SETUP WIZARD CAN BE ACCESSED AGAIN THROUGH THE HELP MENU, OR WHEN CREATING A NEW SHOW.")
	para := ui.NewParagraph(text, th.Fonts.Body, th.Text)
	para.Align = font.Center
	para.SetBounds(image.Rect(r.Min.X+160, r.Min.Y+300, r.Max.X-160, r.Max.Y-130))
	return []ui.View{hat, para}
}

// warning: the hearing-safety notice.
func (p *SetupAssistantPage) warning(env *Env, r image.Rectangle) []ui.View {
	th := env.Ctx.Theme
	icon := ui.NewIcon("IconListeningWarning", th.Text)
	icon.SetBounds(centred(image.Rect(r.Min.X, r.Min.Y+70, r.Max.X, r.Min.Y+250), 150, 150))
	title := ui.NewLabel(env.T("WARNING!"), th.Fonts.Title, th.Text, font.Center)
	title.SetBounds(image.Rect(r.Min.X, r.Min.Y+260, r.Max.X, r.Min.Y+330))
	para := ui.NewParagraph(env.T("THIS EQUIPMENT FEATURES HEADPHONE OUTPUTS THAT CAN PRODUCE VERY HIGH VOLUME LEVELS. LISTENING AT LOUD VOLUMES MAY DAMAGE YOUR HEARING."), th.Fonts.Body, th.Text)
	para.Align = font.Center
	para.SetBounds(image.Rect(r.Min.X+150, r.Min.Y+350, r.Max.X-150, r.Max.Y-130))
	return []ui.View{icon, title, para}
}

// gettingStarted: the "Getting Started" caption, a picture and a sentence.
func (p *SetupAssistantPage) gettingStarted(env *Env, body image.Rectangle, picture string, pw, ph int, text string) []ui.View {
	th := env.Ctx.Theme
	title := wizardTitle(env, env.T("GETTING STARTED"), body)
	// The picture keeps its proportions in a 560 × 300 field.
	field := image.Rect(body.Min.X, body.Min.Y+70, body.Max.X, body.Min.Y+370)
	w, h := pw, ph
	if h > 300 {
		w, h = pw*300/ph, 300
	}
	pic := &pictureView{name: picture}
	pic.SetBounds(centred(field, w, h))
	para := ui.NewParagraph(text, th.Fonts.Body, th.Text)
	para.Align = font.Center
	para.SetBounds(image.Rect(body.Min.X+120, body.Min.Y+390, body.Max.X-120, body.Max.Y+40))
	return []ui.View{title, pic, para}
}

// comboInputs: what the combo inputs (and on a Duo the headset jack) take.
func (p *SetupAssistantPage) comboInputs(env *Env, body image.Rectangle) []ui.View {
	th := env.Ctx.Theme
	var out []ui.View
	out = append(out, wizardTitle(env, env.T("COMBO INPUTS"), body))
	row := func(y int, icons []string, text string) {
		x := body.Min.X + 120
		for _, ic := range icons {
			v := ui.NewIcon(ic, th.Text)
			v.SetBounds(image.Rect(x, y, x+70, y+90))
			out = append(out, v)
			x += 80
		}
		para := ui.NewParagraph(text, th.Fonts.Body, th.Text)
		para.SetBounds(image.Rect(x+20, y, body.Max.X-100, y+100))
		out = append(out, para)
	}
	row(body.Min.Y+90, []string{"combo_jack", "inputMic", "IconInstrument", "inputLineIn"},
		env.T("THE REAR")+" "+env.T("COMBO")+" "+env.T("INPUT CAN BE USED TO CONNECT A MICROPHONE, INSTRUMENT, OR LINE-LEVEL SOURCE."))
	if isDuo(env) {
		row(body.Min.Y+230, []string{"combo_headset", "headset"},
			env.T("THE FRONT")+" "+env.T("TRRS")+" "+env.T("JACK CAN BE USED TO CONNECT A WIRED HEADSET."))
	}
	note := ui.NewLabel(env.T("THE NEXT STEP WILL ALLOW YOU TO ASSIGN EACH INPUT TYPE"), th.Fonts.Body, th.TextDim, font.Center)
	note.SetBounds(image.Rect(body.Min.X, body.Max.Y-20, body.Max.X, body.Max.Y+40))
	out = append(out, note)
	return out
}

// inputChoice is one option of the input selection.
type inputChoice struct {
	source int64
	key    string // COMBO, HEADSET, WIRELESS
	where  string // THE REAR / THE FRONT, "" when it goes without saying
	icon   string
}

// inputChoices lists what physical strip i can carry on this desk: its
// combo input, on a Duo's second strip the front headset, and the wireless
// channel of the same number (sources 19, 20).
func inputChoices(env *Env, strip int) []inputChoice {
	out := []inputChoice{{source: int64(strip), key: "COMBO", icon: "combo" + intText(int64(strip+1))}}
	if isDuo(env) && strip == 1 {
		out[0].where = "THE REAR"
		out = append(out, inputChoice{source: 2, key: "HEADSET", where: "THE FRONT", icon: "combo_headset"})
	}
	if strip < 2 && env.Desk.Count(desk.InputNode) > 20 {
		out = append(out, inputChoice{source: int64(19 + strip), key: "WIRELESS", icon: "InputIcon_Wireless"})
	}
	return out
}

// applyInputs writes the selection's choices that differ from the desk.
func (p *SetupAssistantPage) applyInputs(env *Env) {
	for strip, src := range p.choice {
		if env.Desk.SourceOf(strip) != src {
			env.Desk.SetSource(strip, src)
		}
	}
}

// inputSelection (SetupAssistantInputAssigner): a column per physical
// input with what it can carry.
func (p *SetupAssistantPage) inputSelection(env *Env, body image.Rectangle) []ui.View {
	th := env.Ctx.Theme
	var out []ui.View
	out = append(out, wizardTitle(env, env.T("INPUT SELECTION"), body))
	n := inputSteps(env)
	colW := min(300, (body.Dx()-200)/n)
	x0 := body.Min.X + (body.Dx()-n*colW-(n-1)*40)/2
	for strip := 0; strip < n; strip++ {
		strip := strip
		if _, ok := p.choice[strip]; !ok {
			p.choice[strip] = env.Desk.SourceOf(strip)
		}
		col := image.Rect(x0+strip*(colW+40), body.Min.Y+80, x0+strip*(colW+40)+colW, body.Max.Y+40)
		title := ui.NewLabel(env.T("INPUT "+intText(int64(strip+1))), th.Fonts.Label, th.Text, font.Center)
		title.SetBounds(image.Rect(col.Min.X, col.Min.Y, col.Max.X, col.Min.Y+40))
		out = append(out, title)
		choices := inputChoices(env, strip)
		cells := columns(image.Rect(col.Min.X, col.Min.Y+50, col.Max.X, col.Min.Y+50+150), len(choices), 12)
		for i, ch := range choices {
			ch := ch
			label := env.T(ch.key)
			if ch.where != "" {
				label = env.T(ch.where)
			}
			b := ui.NewTile(label, ch.icon, func() {
				p.choice[strip] = ch.source
				env.Nav.Refresh()
			})
			b.Toggle = true
			b.On = p.choice[strip] == ch.source
			b.SetBounds(cells[i])
			out = append(out, b)
		}
	}
	return out
}

// micStep: the microphone choice of an input.
func (p *SetupAssistantPage) micStep(env *Env, body image.Rectangle, strip int) []ui.View {
	th := env.Ctx.Theme
	input := env.Desk.InputIndex(strip)
	var out []ui.View
	out = append(out, wizardTitle(env, env.T("INPUT "+intText(int64(strip+1))), body))
	sel := ui.NewLabel(env.T("SELECT MICROPHONE"), th.Fonts.Body, th.Text, font.Center)
	sel.SetBounds(image.Rect(body.Min.X, body.Min.Y+70, body.Max.X, body.Min.Y+110))
	out = append(out, sel)
	if input >= 0 && env.Desk.HasMicType(input) {
		car := newMicCarousel(env, strip, input)
		car.SetBounds(image.Rect(body.Min.X, body.Min.Y+120, body.Max.X, body.Min.Y+370))
		out = append(out, car)
	}
	hint := ui.NewParagraph(env.T("IF YOU CAN'T FIND YOUR MICROPHONE IN THE LIST, SELECT ONE OF THE GENERIC OPTIONS THAT MATCHES YOUR MICROPHONE TYPE."), th.Fonts.Small, th.TextDim)
	hint.Align = font.Center
	hint.SetBounds(image.Rect(body.Min.X+120, body.Max.Y-40, body.Max.X-120, body.Max.Y+40))
	out = append(out, hint)
	return out
}

// wirelessStep: pairing a transmitter to a wireless channel. Pairing is
// done with the transmitter itself (and the wireless page); this step
// shows the way to RØDE's information, as the device does.
func (p *SetupAssistantPage) wirelessStep(env *Env, body image.Rectangle, strip int) []ui.View {
	th := env.Ctx.Theme
	title := wizardTitle(env, env.T("INPUT")+" "+intText(int64(strip+1))+" · "+env.T("PAIR WIRELESS TRANSMITTER"), body)
	icon := ui.NewIcon("InputIcon_Wireless", th.Text)
	icon.SetBounds(centred(image.Rect(body.Min.X, body.Min.Y+80, body.Max.X, body.Min.Y+280), 180, 180))
	qr := &pictureView{name: "QRWirelessPairing"}
	qr.SetBounds(image.Rect(body.Max.X-200, body.Max.Y-120, body.Max.X-80, body.Max.Y))
	note := ui.NewParagraph(env.T("SCAN FOR MORE INFORMATION ABOUT WIRELESS INTEGRATION"), th.Fonts.Small, th.TextDim)
	note.SetBounds(image.Rect(body.Max.X-460, body.Max.Y-100, body.Max.X-210, body.Max.Y))
	return []ui.View{title, icon, qr, note}
}

// gainStep: the input's gain with the level meter and its target zone.
func (p *SetupAssistantPage) gainStep(env *Env, body image.Rectangle, strip int) []ui.View {
	th := env.Ctx.Theme
	d := env.Desk
	input := d.InputIndex(strip)
	headset := d.SourceOf(strip) == 2 && isDuo(env)
	var out []ui.View
	title := env.T("SET INPUT") + " " + env.T("GAIN")
	text := env.T("SPEAK INTO MICROPHONE") + " " + env.T("AT THE CORRECT DISTANCE AT A NATURAL VOLUME AND ADJUST THE GAIN BY TAPPING + AND - UNTIL THE METER READS WITHIN THE GREEN TARGET AREA.")
	if headset {
		title = env.T("SET HEADSET GAIN")
		text = env.T("SPEAK INTO HEADSET MICROPHONE AT A NATURAL VOLUME AND ADJUST THE GAIN BY TAPPING + AND -UNTIL THE METER READS WITHIN THE GREEN TARGET AREA.")
	}
	out = append(out, wizardTitle(env, title, body))
	// The input's picture at the left: the chosen microphone, or the
	// instrument / line pictures, or the headset.
	picture := "ImageHeadset"
	if !headset && input >= 0 {
		switch t := d.Int(desk.InputNode, input, "inputMicrophoneType"); t {
		case 0:
			picture = "mic_linein"
		case 1:
			picture = "mic_instrument"
		default:
			picture = desk.MicByType(t).Picture
		}
	}
	pic := &pictureView{name: picture}
	pic.SetBounds(image.Rect(body.Min.X+60, body.Min.Y+80, body.Min.X+300, body.Min.Y+400))
	out = append(out, pic)
	para := ui.NewParagraph(text, th.Fonts.Body, th.Text)
	para.SetBounds(image.Rect(body.Min.X+340, body.Min.Y+70, body.Max.X-60, body.Min.Y+200))
	out = append(out, para)
	if input >= 0 {
		prop := "inputMicrophoneGain"
		if d.Int(desk.InputNode, input, "inputMicrophoneType") == 1 && d.Has(desk.InputNode, input, "inputInstrumentGain") {
			prop = "inputInstrumentGain"
		}
		if d.Has(desk.InputNode, input, prop) {
			st := ui.NewStepper(func() float64 { return float64(d.Int(desk.InputNode, input, prop)) },
				func(delta int) {
					d.SetInt(desk.InputNode, input, prop, d.Int(desk.InputNode, input, prop)+int64(delta))
				})
			st.Label = env.T("GAIN")
			st.Unit = "dB"
			st.Text = func(v float64) string { return dbNum(v) }
			st.SetBounds(image.Rect(body.Min.X+360, body.Min.Y+210, body.Max.X-320, body.Min.Y+340))
			lm := &levelBar{env: env, strip: strip}
			lm.SetBounds(image.Rect(body.Min.X+380, body.Min.Y+350, body.Max.X-340, body.Min.Y+398))
			out = append(out, st, lm)
		}
	}
	qr := &pictureView{name: "QRMicGain"}
	qr.SetBounds(image.Rect(body.Max.X-200, body.Min.Y+230, body.Max.X-60, body.Min.Y+370))
	note := ui.NewParagraph(env.T("SCAN TO LEARN MORE ABOUT MICROPHONE TECHNIQUE"), th.Fonts.Tiny, th.TextDim)
	note.Align = font.Center
	note.SetBounds(image.Rect(body.Max.X-240, body.Min.Y+375, body.Max.X-20, body.Min.Y+440))
	out = append(out, qr, note)
	return out
}

// presetStep: the three factory presets of an input (the device offers the
// first three: Neutral, Podcast, Broadcast).
func (p *SetupAssistantPage) presetStep(env *Env, body image.Rectangle, strip int) []ui.View {
	th := env.Ctx.Theme
	d := env.Desk
	headset := d.SourceOf(strip) == 2 && isDuo(env)
	title := env.T("SET MICROPHONE") + " " + env.T("PRESET")
	if headset {
		title = env.T("SET HEADSET PRESET")
	}
	var out []ui.View
	out = append(out, wizardTitle(env, title, body))
	ps := d.Presets()
	if len(ps) > 3 {
		ps = ps[:3]
	}
	if len(ps) > 0 {
		cells := columns(image.Rect(body.Min.X+140, body.Min.Y+110, body.Max.X-140, body.Min.Y+210), len(ps), 20)
		for i, pr := range ps {
			pr := pr
			b := ui.NewButton(pr.Name, func() {
				if !d.ApplyPreset(strip, pr) {
					env.Nav.Toast(env.TR("", "Dieses Preset ist nicht lesbar.", "This preset cannot be read."))
				}
			})
			b.Toggle = true
			b.On = d.CurrentPreset(strip) == pr.Idx
			b.SetBounds(cells[i])
			out = append(out, b, &updater{get: func() { b.SetOn(d.CurrentPreset(strip) == pr.Idx) }})
		}
	}
	para := ui.NewParagraph(env.T("TAP A PRESET BELOW TO AUDITION THEM, AND TAP NEXT WHEN YOU ARE HAPPY WITH THE SOUND."), th.Fonts.Body, th.Text)
	para.Align = font.Center
	para.SetBounds(image.Rect(body.Min.X+100, body.Min.Y+250, body.Max.X-100, body.Min.Y+340))
	later := ui.NewLabel(env.T("THESE SETTINGS CAN BE ADJUSTED LATER."), th.Fonts.Small, th.TextDim, font.Center)
	later.SetBounds(image.Rect(body.Min.X, body.Max.Y-20, body.Max.X, body.Max.Y+30))
	out = append(out, para, later)
	return out
}

// bluetoothStep: the desk is discoverable; pair a phone or skip it.
func (p *SetupAssistantPage) bluetoothStep(env *Env, body image.Rectangle) []ui.View {
	th := env.Ctx.Theme
	d := env.Desk
	title := wizardTitle(env, env.T("BLUETOOTH"), body)
	icon := ui.NewIcon("BluetoothPairing", th.Text)
	icon.SetBounds(centred(image.Rect(body.Min.X, body.Min.Y+70, body.Max.X, body.Min.Y+230), 160, 160))
	name := d.Str(desk.SystemNode, 0, "systemName")
	if name == "" {
		name = d.Info().ModelName
	}
	disc := ui.NewLabel(name+" "+env.T("IS NOW DISCOVERABLE"), th.Fonts.Body, th.Accent, font.Center)
	disc.SetBounds(image.Rect(body.Min.X, body.Min.Y+240, body.Max.X, body.Min.Y+290))
	open := "OPEN THE BLUETOOTH SETTINGS IN THE DEVICE YOU WANT TO CONNECT AND SELECT 'RODECASTER DUO'"
	if !isDuo(env) {
		open = "OPEN THE BLUETOOTH SETTINGS IN THE DEVICE YOU WANT TO CONNECT AND SELECT 'RODECASTER PRO II'"
	}
	para := ui.NewParagraph(env.T(open)+"\n"+env.T("THE BLUETOOTH CHANNEL IS IDEAL FOR CONNECTING DEVICES SUCH AS SMARTPHONES, COMPUTERS AND TABLETS WIRELESSLY."), th.Fonts.Small, th.TextDim)
	para.Align = font.Center
	para.SetBounds(image.Rect(body.Min.X+120, body.Min.Y+300, body.Max.X-120, body.Max.Y+20))
	later := ui.NewButton(env.T("LATER"), func() { p.next(env) })
	later.SetBounds(image.Rect(body.Min.X+body.Dx()/2-150, body.Max.Y+40, body.Min.X+body.Dx()/2+150, body.Max.Y+110))
	return []ui.View{title, icon, disc, para, later}
}

// usbStep: what the three USB sources are for.
func (p *SetupAssistantPage) usbStep(env *Env, body image.Rectangle) []ui.View {
	th := env.Ctx.Theme
	var out []ui.View
	out = append(out, wizardTitle(env, env.T("USB INPUTS"), body))
	rows := []struct{ icon, name, text string }{
		{"IconStripUsb1Main", "USB 1", "USB 1 CAN BE USED AS A STEREO INPUT AND MULTICHANNEL OUTPUT DEVICE"},
		{"IconStripUsb1Comms", "USB 1 CHAT", "USB 1 CHAT FEATURES MIX-MINUS FOR USE WITH COMMUNICATION APPS"},
		{"IconStripUsb2Main", "USB 2", "USB 2 CAN BE USED TO CONNECT AN ADDITIONAL DEVICE OR A COMPATIBLE RODE USB MICROPHONE"},
	}
	for i, row := range rows {
		y := body.Min.Y + 90 + i*140
		icon := ui.NewIcon(row.icon, th.Text)
		icon.SetBounds(image.Rect(body.Min.X+140, y, body.Min.X+230, y+90))
		name := ui.NewLabel(env.T(row.name), th.Fonts.Label, th.Text, font.Left)
		name.SetBounds(image.Rect(body.Min.X+270, y, body.Max.X-100, y+40))
		text := ui.NewParagraph(env.T(row.text), th.Fonts.Small, th.TextDim)
		text.SetBounds(image.Rect(body.Min.X+270, y+42, body.Max.X-100, y+120))
		out = append(out, icon, name, text)
	}
	return out
}

// padsStep: the pad types.
func (p *SetupAssistantPage) padsStep(env *Env, body image.Rectangle) []ui.View {
	th := env.Ctx.Theme
	var out []ui.View
	out = append(out, wizardTitle(env, env.T("SMART PADS"), body))
	pic := &pictureView{name: "SetupAssistantPads" + deviceSuffix(env)}
	pic.SetBounds(image.Rect(body.Min.X+120, body.Min.Y+80, body.Min.X+360, body.Max.Y+40))
	out = append(out, pic)
	intro := ui.NewParagraph(env.T("SMART PADS HAVE VARIOUS TYPES AND BEHAVIOURS."), th.Fonts.Body, th.Text)
	intro.SetBounds(image.Rect(body.Min.X+420, body.Min.Y+80, body.Max.X-60, body.Min.Y+130))
	out = append(out, intro)
	kinds := []struct{ icon, key string }{
		{"pads_sound", "SOUNDS"}, {"pads_effect", "EFFECTS"}, {"pads_mixer", "MIXER ACTIONS"}, {"pads_action", "MIDI"},
	}
	for i, k := range kinds {
		y := body.Min.Y + 150 + i*70
		icon := ui.NewIcon(k.icon, th.Text)
		icon.SetBounds(image.Rect(body.Min.X+420, y, body.Min.X+470, y+50))
		lbl := ui.NewLabel(env.T(k.key), th.Fonts.Body, th.Text, font.Left)
		lbl.SetBounds(image.Rect(body.Min.X+490, y, body.Max.X-60, y+50))
		out = append(out, icon, lbl)
	}
	note := ui.NewParagraph(env.T("THE ICON SHOWN INSIDE THE PAD DENOTES THE MAIN TYPE."), th.Fonts.Small, th.TextDim)
	note.SetBounds(image.Rect(body.Min.X+420, body.Min.Y+440, body.Max.X-60, body.Min.Y+500))
	out = append(out, note)
	return out
}

// showsStep: what a show is.
func (p *SetupAssistantPage) showsStep(env *Env, body image.Rectangle) []ui.View {
	th := env.Ctx.Theme
	title := wizardTitle(env, env.T("SHOWS"), body)
	icon := ui.NewIcon("IconShows", th.Text)
	icon.SetBounds(centred(image.Rect(body.Min.X, body.Min.Y+70, body.Max.X, body.Min.Y+230), 140, 140))
	text := env.T("ALL DEVICE SETTINGS AND PREFERENCES ARE SAVED AS A") + " " + env.T("SHOW.") + "\n\n" +
		env.T("SHOWS") + " " + env.T("CAN BE SAVED, LOADED, AND SHARED AT ANY TIME THROUGH THE") + " " + env.T("SHOWS MENU.")
	para := ui.NewParagraph(text, th.Fonts.Body, th.Text)
	para.Align = font.Center
	para.SetBounds(image.Rect(body.Min.X+120, body.Min.Y+260, body.Max.X-120, body.Max.Y+40))
	return []ui.View{title, icon, para}
}

// nameStep: the show's name, typed on the keyboard.
func (p *SetupAssistantPage) nameStep(env *Env, body image.Rectangle) []ui.View {
	th := env.Ctx.Theme
	d := env.Desk
	title := wizardTitle(env, env.T("NAME YOUR SHOW"), body)
	name := ui.NewButton(d.Str("CURRENTSHOW", 0, "currentShowName"), func() {
		env.Nav.Keyboard(env.T("NAME YOUR SHOW"), d.Str("CURRENTSHOW", 0, "currentShowName"), func(s string) {
			if s != "" && s != d.Str("CURRENTSHOW", 0, "currentShowName") {
				d.SetStr("CURRENTSHOW", 0, "currentShowName", s)
			}
			env.Nav.Refresh()
		})
	})
	name.Font = th.Fonts.Title
	name.SetBounds(image.Rect(body.Min.X+body.Dx()/2-360, body.Min.Y+170, body.Min.X+body.Dx()/2+360, body.Min.Y+290))
	return []ui.View{title, name}
}

// doneStep: ready to go.
func (p *SetupAssistantPage) doneStep(env *Env, r image.Rectangle) []ui.View {
	th := env.Ctx.Theme
	logo := &pictureView{name: deviceLogo(env)}
	logo.SetBounds(centred(image.Rect(r.Min.X, r.Min.Y+120, r.Max.X, r.Min.Y+240), 700, 60))
	big := ui.NewLabel(env.T("YOU'RE READY TO GO!"), th.Fonts.Title, th.Text, font.Center)
	big.SetBounds(image.Rect(r.Min.X, r.Min.Y+300, r.Max.X, r.Min.Y+380))
	return []ui.View{logo, big}
}

// wizardTitle is a step's caption at the top of its body.
func wizardTitle(env *Env, text string, body image.Rectangle) ui.View {
	th := env.Ctx.Theme
	f := th.Fonts.Title
	if ui.TextWidth(f, text) > body.Dx()-200 {
		f = th.Fonts.Body
	}
	l := ui.NewLabel(text, f, th.Text, font.Center)
	l.SetBounds(image.Rect(body.Min.X+60, body.Min.Y, body.Max.X-60, body.Min.Y+60))
	return l
}

// pictureView draws one of RØDE's pictures (a PNG or SVG) fitted into its
// bounds, untinted.
type pictureView struct {
	ui.Base
	name string
}

func (v *pictureView) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	c.IconIn(dst, v.name, v.Bounds(), 0, 255, clip)
}

// pictureButton is a tappable untinted picture (the assistant's cross).
type pictureButton struct {
	ui.Base
	name    string
	onTap   func()
	pressed bool
}

func (b *pictureButton) CanFocus() bool { return true }
func (b *pictureButton) Activate() bool {
	if b.onTap != nil {
		b.onTap()
	}
	return false
}

func (b *pictureButton) Touch(phase ui.TouchPhase, p image.Point) bool {
	switch phase {
	case ui.Down:
		b.pressed = true
	case ui.Up:
		if b.pressed && b.onTap != nil {
			b.onTap()
		}
		b.pressed = false
	}
	return true
}

func (b *pictureButton) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	c.IconIn(dst, b.name, b.Bounds(), 0, 255, clip)
	if b.Focused() {
		ui.Focus(dst, b.Bounds(), 10, false, c.Theme, clip)
	}
}

// gradientView fills its bounds with a vertical gradient.
type gradientView struct {
	ui.Base
	top, bottom gfx.Color
}

func (g *gradientView) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	ui.Gradient(dst, g.Bounds(), 0, g.top, g.bottom, clip)
}
