package pages

import "testing"

// The strip icon table of RØDE's screen program (rc_gui_juce 1.7.6,
// FUN_004ffb40): the icon is picked by source, microphone type and, for the
// RØDECaster Video sources, the audio source type.
func TestStripIconChoice(t *testing.T) {
	cases := []struct {
		source, mic, rcv int64
		duo              bool
		want             string
	}{
		{0, 8, -1, true, "IconStripMic1"},  // NT1-A on input 1
		{1, 2, -1, true, "IconStripMic2"},  // dynamic
		{0, 0, -1, true, "IconStripLine1"}, // line-in
		{1, 1, -1, true, "IconStripInstrument2"},
		{0, 11, -1, true, "IconStripWireless1"}, // wireless microphone type
		{3, 10, -1, false, "IconStripMic4"},
		{4, 4, -1, true, "IconStripMics12"},
		{5, 0, -1, false, "IconStripLines23"},
		{6, 1, -1, false, "IconStripInstruments34"},
		{6, 11, -1, false, "IconStripWireless34"},
		{2, 4, -1, true, "headset"},        // the Duo's front headset
		{2, 4, -1, false, "IconStripMic3"}, // the Pro II's third combo
		{0, 12, -1, true, ""},              // unknown microphone type: nothing
		{7, 0, -1, true, "IconStripUsb1Main"},
		{8, 0, -1, true, "IconStripUsb1Comms"},
		{9, 0, -1, true, "IconStripUsb2Main"},
		{10, 0, -1, true, "bluetooth"},
		{11, 0, -1, true, "pads"},
		{12, 0, -1, true, "VirtualInput_Game"},
		{13, 0, -1, true, "VirtualInput_Music"},
		{14, 0, -1, true, "Letter_A"},
		{15, 0, -1, true, "Letter_B"},
		{16, 0, -1, true, "CallMeFaderIcon1"},
		{18, 0, -1, true, "CallMeFaderIcon3"},
		{19, 11, -1, true, "IconStripWireless1"},
		{20, 11, -1, true, "IconStripWireless2"},
		{21, 0, 0, true, "IconStripMic1"},
		{21, 0, 7, true, "InputIcon_hdmi1L"},
		{22, 0, 12, true, "InputIcon_hdmi2"},
		{23, 0, 17, true, "InputIcon_hdmi4R"},
		{24, 0, 19, true, "bluetooth"},
		{25, 0, 23, true, "IconStripUsb4"},
		{26, 0, 28, true, "IconStripUsb5"},
		{27, 0, 31, true, "InputIcon_Network3"},
		{29, 0, 99, true, "Oslash"},
		{-1, 0, -1, true, "plus"}, // an unassigned strip
		{30, 0, -1, true, ""},
		{-2, 0, -1, true, ""},
	}
	for _, c := range cases {
		if got := stripIconChoice(c.source, c.mic, c.rcv, c.duo); got != c.want {
			t.Errorf("stripIconChoice(%d, %d, %d, %v) = %q, want %q", c.source, c.mic, c.rcv, c.duo, got, c.want)
		}
	}
}
