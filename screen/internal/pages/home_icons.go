package pages

import "github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"

// The strip icons of the mixer, chosen as RØDE's screen chooses them: by the
// source number, and for the combo inputs by the microphone type (line,
// instrument, microphone or wireless), for the RØDECaster Video sources by
// their audio source type. An unassigned strip shows a grey "+"; on a Duo
// the third combo source is the front headset.

// stripIconChoice is the icon of a source. source is channelInputSource,
// micType the source's inputMicrophoneType, rcvType its
// inputRcvAudioSourceType, duo whether the desk is a RØDECaster Duo. It
// returns "" when the source is out of range (nothing is drawn).
func stripIconChoice(source, micType, rcvType int64, duo bool) string {
	if source < -1 || source > 29 {
		return ""
	}
	if source >= 0 && source <= 6 && micType >= 12 {
		return "" // a combo input with an unknown microphone type draws nothing
	}
	switch {
	case source < 0:
		return "plus"
	case source <= 6:
		if source == 2 && duo {
			return "headset"
		}
		return comboStripIcon(int(source), micType)
	}
	switch source {
	case 7:
		return "IconStripUsb1Main"
	case 8:
		return "IconStripUsb1Comms"
	case 9:
		return "IconStripUsb2Main"
	case 10:
		return "bluetooth"
	case 11:
		return "pads"
	case 12:
		return "VirtualInput_Game"
	case 13:
		return "VirtualInput_Music"
	case 14:
		return "Letter_A"
	case 15:
		return "Letter_B"
	case 16:
		return "CallMeFaderIcon1"
	case 17:
		return "CallMeFaderIcon2"
	case 18:
		return "CallMeFaderIcon3"
	case 19:
		return "IconStripWireless1"
	case 20:
		return "IconStripWireless2"
	}
	return rcvStripIcon(rcvType)
}

// comboStripSuffixes are the number parts of the combo icons: inputs 1…4,
// then the linked pairs 1+2, 2+3, 3+4 (sources 4, 5, 6).
var comboStripSuffixes = []string{"1", "2", "3", "4", "12", "23", "34"}

// comboStripIcon is the icon of combo source 0…6 for a microphone type:
// 0 line, 1 instrument, 2…10 a microphone, anything else wireless.
func comboStripIcon(source int, micType int64) string {
	suffix := comboStripSuffixes[source]
	pair := source >= 4
	kind := "Wireless"
	switch {
	case micType >= 2 && micType <= 10:
		kind = "Mic"
	case micType == 1:
		kind = "Instrument"
	case micType == 0:
		kind = "Line"
	}
	if pair && kind != "Wireless" {
		kind += "s" // IconStripMics12, IconStripLines12, IconStripInstruments12
	}
	return "IconStrip" + kind + suffix
}

// rcvStripIcon is the icon of a RØDECaster Video source (sources 21…29) by
// its audio source type.
func rcvStripIcon(t int64) string {
	switch t {
	case 0:
		return "IconStripMic1"
	case 1:
		return "IconStripMic2"
	case 2:
		return "IconStripMics12"
	case 3:
		return "IconStripWireless1"
	case 4:
		return "IconStripWireless2"
	case 5:
		return "InputIcon_RCVVideoClips"
	case 6:
		return "InputIcon_RCVSounds"
	case 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18:
		// HDMI 1…4, each as left, right, both.
		n := (t-7)/3 + 1
		side := []string{"L", "R", ""}[(t-7)%3]
		return "InputIcon_hdmi" + string(rune('0'+n)) + side
	case 19:
		return "bluetooth"
	case 20:
		return "IconStripUsb1Main"
	case 21:
		return "IconStripUsb1Comms"
	case 22:
		return "IconStripUsb2Main"
	case 23, 25, 26:
		return "IconStripUsb4"
	case 24, 27, 28:
		return "IconStripUsb5"
	case 29, 30, 31, 32:
		return "InputIcon_Network" + string(rune('0'+t-28))
	}
	return "Oslash"
}

// isDuo reports whether the desk is a RØDECaster Duo (SYSTEM.boardType 1,
// as the screen program tests it; the console's model name as a fallback).
func isDuo(env *Env) bool {
	if env.Desk.Has(desk.SystemNode, 0, "boardType") {
		return env.Desk.Int(desk.SystemNode, 0, "boardType") == 1
	}
	return env.Desk.Info().Model == "duo"
}

// stripIcon is the mixer icon of strip i's source.
func stripIcon(env *Env, strip int) string {
	d := env.Desk
	src := d.SourceOf(strip)
	var micType, rcvType int64 = -1, -1
	if in := d.InputIndex(strip); in >= 0 {
		micType = d.Int(desk.InputNode, in, "inputMicrophoneType")
		if d.Has(desk.InputNode, in, "inputRcvAudioSourceType") {
			rcvType = d.Int(desk.InputNode, in, "inputRcvAudioSourceType")
		}
	}
	return stripIconChoice(src, micType, rcvType, isDuo(env))
}

// plusIcon is the "+" of an unassigned strip; RØDE draws it light grey.
const plusIcon = "plus"
