package desk

// The enumerations behind the settings pages, as RØDE's screen program
// (rc_gui_juce 1.7.6) numbers them. Each value is what the screen writes to
// the property named; the mixer applies the side effects.

// HEADPHONE[p].headphoneType (value type HeadphonesType): the gain structure
// of a headphone output. The screen's HeadphonesPage builds its three
// buttons in this order with these ids.
const (
	HeadphonesNTH100          int64 = 0 // "RØDE NTH-100"
	HeadphonesHighSensitivity int64 = 1 // earphones, IEMs, low impedance
	HeadphonesLowSensitivity  int64 = 2 // high-impedance headphones (more output)
)

// SYSTEM.systemHapticSetting (HapticMode): the touchscreen's vibration.
const (
	HapticOff    int64 = 0 // "No Haptic Feedback"
	HapticHold   int64 = 1 // "Haptic Feedback on Hold"
	HapticAlways int64 = 2 // "Haptic Feedback Always On"
)

// GUI.metering (MeteringType): how the home screen's meters are scaled.
const (
	MeteringDefault   int64 = 0
	MeteringBroadcast int64 = 1
)

// OUTPUT.recordingCompressionQuality (CompressionQualityType): the recording
// format. The screen's cards pair 0 with "Broadcast Quality" and 1 with
// "High Quality" (and their descriptions in that order).
const (
	CompressionBroadcast int64 = 0
	CompressionHigh      int64 = 1
)

// SYSTEM.systemRecButtonSetting (RecButtonMode): what a press of REC does
// while recording.
const (
	RecButtonPause int64 = 0 // a press pauses, a hold stops
	RecButtonStop  int64 = 1 // a press stops, a hold pauses
)

// OUTPUT.recordingMultitrackMode: the recording's multitrack bus (off,
// pre-fader, post-fader), in the order of the screen's buttons.
const (
	MultitrackOff  int64 = 0
	MultitrackPre  int64 = 1
	MultitrackPost int64 = 2
)

// OUTPUT.outputMultiMode (OutputMultiModeType) holds both USB 1 settings of
// the multitrack page in one number: the USB 1 Output bus (off, pre-fader,
// post-fader = 0, 1, 2) plus 3 when the USB 1 Input is Expanded (the four
// extra virtual inputs). The screen's Expanded button writes 3, 4 or 5 and
// its Standard button 0, 1 or 2, keeping the output bus.
const usbModeExpanded int64 = 3

// USBOutputMode is the USB 1 Output bus of an outputMultiMode value.
func USBOutputMode(v int64) int64 { return ((v % 3) + 3) % 3 }

// USBInputExpanded reports whether an outputMultiMode value has the USB 1
// Input expanded.
func USBInputExpanded(v int64) bool { return v >= usbModeExpanded }

// USBMode composes an outputMultiMode value.
func USBMode(output int64, expanded bool) int64 {
	v := USBOutputMode(output)
	if expanded {
		v += usbModeExpanded
	}
	return v
}

// Languages the screen offers, in the order of its language page, with the
// codes GUI.lang holds. The names are shown untranslated, as the screen
// shows them.
var Languages = []struct{ Code, Name string }{
	{"en", "English"}, {"es", "Español"}, {"fr", "Français"}, {"it", "Italiano"},
	{"pt", "Português"}, {"de", "Deutsch"}, {"ru", "Русский"}, {"ko", "한국어"},
	{"zh", "中文"}, {"ja", "日本語"},
}

// LanguageIndex is the position of a language code in Languages (−1 if the
// code is not offered).
func LanguageIndex(code string) int {
	for i, l := range Languages {
		if l.Code == code {
			return i
		}
	}
	return -1
}

// IsDuo tells a Duo from a Pro II (the regulatory page and a few texts
// differ between them).
func (m *Model) IsDuo() bool { return m.Info().Model != "pro2" }
