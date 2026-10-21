package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// The Display menu's pages and the recording-format page. RØDE's TextRadio
// pages put a picture on the left and the cards on the right (x 483, 715
// wide; three cards 88 high at y 177/352/527, two cards 134 high at
// y 191/454, in the screen's 1280 × 720 frame); the card order and the
// value each card writes come from the screen program's paint functions.

// cardPage lays a picture left and the option cards right, as those pages
// do. Each card writes its value to get/set.
func cardPage(title, picture string, opts []radioSpec, get func(*Env) int64, set func(*Env, int64)) Page {
	return &funcPage{title: title, build: func(env *Env, r image.Rectangle) ui.View {
		c := &ui.Container{}
		c.SetBounds(r)
		th := env.Ctx.Theme
		if picture != "" {
			pic := ui.NewIcon(picture, th.TextDim)
			pic.SetBounds(centred(image.Rect(r.Min.X, r.Min.Y, r.Max.X-740, r.Max.Y), 220, 220))
			c.Add(pic)
		}
		choices := make([]ui.Choice, len(opts))
		for i, o := range opts {
			choices[i] = ui.Choice{Title: env.T(o.title), Icon: o.icon}
			if o.desc != "" {
				choices[i].Desc = env.TR(o.desc, o.de, o.en)
			}
		}
		cl := ui.NewChoiceList(choices, func() int {
			cur := get(env)
			for i, o := range opts {
				if o.value == cur {
					return i
				}
			}
			return -1
		}, func(i int) { set(env, opts[i].value) })
		n := len(opts)
		rowH := 130
		if n <= 2 {
			rowH = 200
		}
		if total := n*rowH + (n-1)*16; total > r.Dy() {
			rowH = (r.Dy() - (n-1)*16) / n
		}
		top := r.Min.Y + max(0, (r.Dy()-(n*rowH+(n-1)*16))/2)
		cl.RowH = rowH
		cl.SetBounds(image.Rect(r.Max.X-715, top, r.Max.X, top+n*rowH+(n-1)*16))
		c.Add(cl)
		return c
	}}
}

// intProp binds a card page to an integer property.
func intProp(node string, idx int, prop string) (func(*Env) int64, func(*Env, int64)) {
	return func(env *Env) int64 { return env.Desk.Int(node, idx, prop) },
		func(env *Env, v int64) { env.Desk.SetInt(node, idx, prop, v) }
}

// MeteringPage: GUI.metering, Default (0) or Broadcast (1), each card with
// its meter picture.
func newMeteringPage() Page {
	get, set := intProp(desk.GUINode, 0, "metering")
	return cardPage("METERING", "IconMetering", []radioSpec{
		{title: "DEFAULT", icon: "MeteringDefault", value: desk.MeteringDefault,
			desc: "LEVEL METERS WILL INCLUDE MARKERS AT REGULAR INTERVALS AND WILL INDICATE UNITY LEVEL."},
		{title: "BROADCAST", icon: "MeteringBroadcast", value: desk.MeteringBroadcast,
			desc: "LEVEL METERS WILL INCLUDE DBFS MARKERS ON EACH CHANNEL. GAIN REDUCTION METER WILL BE VISIBLE FOR EACH CHANNEL."},
	}, get, set)
}

// HapticsPage: SYSTEM.systemHapticSetting, off (0), on hold (1), always (2),
// in the order the screen draws them.
func newHapticsPage() Page {
	get, set := intProp(desk.SystemNode, 0, "systemHapticSetting")
	return cardPage("HAPTICS", "IconHaptics", []radioSpec{
		{title: "NO HAPTIC FEEDBACK", icon: "IconHapticsOff", value: desk.HapticOff},
		{title: "HAPTIC FEEDBACK ON HOLD", icon: "IconHapticsHold", value: desk.HapticHold},
		{title: "HAPTIC FEEDBACK ALWAYS ON", icon: "IconHapticsAlways", value: desk.HapticAlways},
	}, get, set)
}

// RecButtonPage: SYSTEM.systemRecButtonSetting, press to pause (0) or press
// to stop (1).
func newRecButtonPage() Page {
	get, set := intProp(desk.SystemNode, 0, "systemRecButtonSetting")
	return cardPage("REC BUTTON", "IconRecordingSettings", []radioSpec{
		{title: "PRESS TO PAUSE RECORDING", icon: "IconRecPause", value: desk.RecButtonPause,
			desc: "A SINGLE PRESS WILL PAUSE THE RECORDING. \nPRESSING AND HOLDING WILL STOP THE RECORDING."},
		{title: "PRESS TO STOP RECORDING", icon: "IconRecStop", value: desk.RecButtonStop,
			desc: "A SINGLE PRESS WILL STOP THE RECORDING. \nPRESSING AND HOLDING WILL PAUSE THE RECORDING."},
	}, get, set)
}

// CompressionQualityPage: OUTPUT.recordingCompressionQuality. The screen
// pairs the cards and texts exactly so: Broadcast Quality (0) with "a higher
// level of sound quality", High Quality (1) with "a smaller file size".
func newCompressionPage() Page {
	get, set := intProp(desk.OutputNode, 0, "recordingCompressionQuality")
	return cardPage("RECORDING", "IconRecordingSettings", []radioSpec{
		{title: "BROADCAST QUALITY", icon: "IconCompressionWAV", value: desk.CompressionBroadcast,
			desc: "PROVIDES A HIGHER LEVEL OF SOUND QUALITY WITH LARGER STORAGE REQUIREMENTS."},
		{title: "HIGH QUALITY", icon: "IconCompressionMP3", value: desk.CompressionHigh,
			desc: "PROVIDES A SMALLER FILE SIZE FOR EASIER STORAGE AND DISTRIBUTION."},
	}, get, set)
}
