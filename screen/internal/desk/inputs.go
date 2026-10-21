package desk

import "strconv"

// The kinds of input source and the nodes behind the wireless receiver, as
// RØDE's screen program tells them apart (its page per source: combo,
// wireless, USB, Bluetooth, pads, CallMe, the Video sync sources, none).

// Node types of the internal wireless receiver.
const (
	RadioNode   = "RADIO"   // radioPair (int, −1 at rest), radioPaired (int), radioUnpair (string)
	RadioTxNode = "RADIOTX" // one per transmitter slot: txConnectionId 0/1, txDeviceType, txConnected …
	RadioRxNode = "RADIORX" // rxRadioId
)

// Source numbers RØDE's program hard-codes (channelInputSource values).
const (
	SourceComboPair = 4  // inputs 1 and 2 linked as one stereo source
	SourceUSB1      = 7  // USB 1 main
	SourceChat      = 8  // USB 1 chat
	SourceUSB2      = 9  // USB 2
	SourceBluetooth = 10 // Bluetooth
	SourcePads      = 11 // SMART pads
	SourceCallMe1   = 16 // CallMe 1…3 (16…18)
	SourceWireless1 = 19 // wireless channel 1 of the internal receiver
	SourceWireless2 = 20 // wireless channel 2
	SourceSync1     = 21 // RØDECaster Video sync sources 21…29
	SourceCount     = 30 // INPUTSOURCE nodes of firmware 1.7.6
)

// SourceKind is the family of a source, which decides the channel page.
type SourceKind int

const (
	KindNone      SourceKind = iota // no source
	KindCombo                       // one combo input (microphone, line, instrument)
	KindHeadset                     // the Duo's front TRRS headset (its third input source)
	KindComboPair                   // two linked combo inputs
	KindWireless                    // a wireless channel of the internal receiver
	KindUSB                         // USB 1, chat, USB 2
	KindVirtual                     // the virtual devices Game, Music, A, B
	KindBluetooth                   // Bluetooth
	KindPads                        // SMART pads
	KindCallMe                      // a CallMe call
	KindSync                        // a RØDECaster Video source in sync mode
)

// InputKinds are the values of INPUTSOURCE.inputType (InputComboType in
// RØDE's program): 0 a combo socket, 3 a wireless channel; −1 on the
// sources without a physical socket.
const (
	InputTypeCombo    = 0
	InputTypeWireless = 3
)

// SourceHeadset is the headset's source number on a Duo: its program names
// InputSourceType 2 "Headset" on the Duo and "Input 3" on the Pro II (its
// name switch FUN_00709660), so the front TRRS jack is the third combo
// source of the mixer.
const SourceHeadset = 2

// IsDuo reports whether the desk is a RØDECaster Duo.
func (m *Model) IsDuo() bool { return m.Info().Model == "duo" }

// IsHeadset reports whether source s is the Duo's headset.
func (m *Model) IsHeadset(s int64) bool { return s == SourceHeadset && m.IsDuo() }

// SourceKindOf tells the family of source s from its number and the
// source's inputType; the numbering is the mixer's.
func (m *Model) SourceKindOf(s int64) SourceKind {
	switch {
	case s < 0:
		return KindNone
	case m.IsHeadset(s):
		return KindHeadset
	case s <= 3:
		return KindCombo
	case s <= 6:
		return KindComboPair
	case s <= 9:
		return KindUSB
	case s == SourceBluetooth:
		return KindBluetooth
	case s == SourcePads:
		return KindPads
	case s <= 15:
		return KindVirtual
	case s <= 18:
		return KindCallMe
	case s <= SourceWireless2:
		return KindWireless
	}
	return KindSync
}

// SourceName is the label of source index s; the host names the first 19
// sources, the wireless and sync sources of 1.7.6 are named here.
func (m *Model) SourceInfo(s int64) Name {
	if m.IsHeadset(s) {
		return Name{ID: "headset", Label: "Headset"}
	}
	if n := m.SourceName(s); n.ID != "" {
		return n
	}
	if s < 0 || int(s) >= m.Count(tInput) {
		return Name{}
	}
	switch {
	case m.IsHeadset(s):
		return Name{ID: "headset", Label: "Headset"}
	case s == SourceWireless1 || s == SourceWireless2:
		n := strconv.FormatInt(s-SourceWireless1+1, 10)
		return Name{ID: "wireless" + n, Label: "Wireless " + n}
	case s >= SourceSync1:
		n := strconv.FormatInt(s-SourceSync1+1, 10)
		return Name{ID: "sync" + n, Label: "Sync " + n}
	}
	return Name{ID: "source" + strconv.FormatInt(s, 10), Label: "Source " + strconv.FormatInt(s, 10)}
}

// HasWireless reports whether the desk has the internal receiver's sources
// (firmware 1.7.3 and later: INPUTSOURCE 19 and 20 with inputType 3).
func (m *Model) HasWireless() bool {
	return m.Count(tInput) > SourceWireless2 && m.Int(tInput, SourceWireless1, "inputType") == InputTypeWireless
}

// WirelessTx is the RADIOTX node index of wireless source s (19 → the
// transmitter with txConnectionId 0, 20 → 1), or −1.
func (m *Model) WirelessTx(s int64) int {
	if s != SourceWireless1 && s != SourceWireless2 {
		return -1
	}
	want := s - SourceWireless1
	n := m.Count(RadioTxNode)
	for i := 0; i < n; i++ {
		if m.Int(RadioTxNode, i, "txConnectionId") == want {
			return i
		}
	}
	if int(want) < n {
		return int(want)
	}
	return -1
}

// WirelessTypeName names a transmitter type (RADIOTX.txDeviceType), as
// RØDE's program does (its name switch, FUN_00614860): −1 and 7 are shown
// as plain "Wireless".
func WirelessTypeName(t int64) string {
	switch t {
	case 0:
		return "Wireless GO II"
	case 1:
		return "Wireless PRO"
	case 2:
		return "Wireless ME"
	case 3:
		return "Interview PRO"
	case 4:
		return "Wireless GO"
	case 5:
		return "RODELink II"
	case 6:
		return "EM-89 GO"
	}
	return "Wireless"
}

// WirelessTypePicture is the picture RØDE's program has for a transmitter
// type: the Wireless ME transmitter for its own kind, the GO II picture for
// everything else (the program ships only these two).
func WirelessTypePicture(t int64) string {
	if t == 2 {
		return "mic_wiMEtx"
	}
	return "mic_wgo2"
}

// GainAssist modes of a transmitter (RADIOTX.txGainAssist), from the
// program's button text: 0 Auto, 1 Dynamic, 2 and above Off. The button
// is shown for the types 1…4 (Wireless PRO, ME, Interview PRO, Wireless
// GO); the Wireless GO II (type 0) shows its pad instead (txPad).
const (
	GainAssistAuto    = 0
	GainAssistDynamic = 1
	GainAssistOff     = 2
)

// GainAssistKey is the text key of a GainAssist mode.
func GainAssistKey(v int64) string {
	switch v {
	case GainAssistAuto:
		return "AUTO"
	case GainAssistDynamic:
		return "DYNAMIC"
	}
	return "OFF"
}

// HasGainAssist reports whether a transmitter type offers GainAssist.
func HasGainAssist(t int64) bool { return t >= 1 && t <= 4 }

// HasTxPad reports whether a transmitter type offers the pad switch.
func HasTxPad(t int64) bool { return t == 0 }

// Pairing reports whether the receiver is searching for a transmitter
// (RADIO.radioPair is the slot being paired, −1 at rest).
func (m *Model) Pairing() bool {
	return m.Has(RadioNode, 0, "radioPair") && m.Int(RadioNode, 0, "radioPair") != -1
}

// StartPairing asks the receiver to pair the transmitter of a wireless
// source (the slot's number, as RØDE's "Pair New Device" button writes).
func (m *Model) StartPairing(s int64) {
	m.SetInt(RadioNode, 0, "radioPair", s-SourceWireless1)
}

// Unpair asks the receiver to forget the transmitter of a wireless source,
// by its serial (RADIO.radioUnpair takes the TX serial).
func (m *Model) Unpair(s int64) {
	tx := m.WirelessTx(s)
	if tx < 0 {
		return
	}
	sn := m.Str(RadioTxNode, tx, "txDeviceSN")
	if sn == "" {
		sn = m.Str(tInput, int(s), "inputWirelessSN")
	}
	if sn == "" {
		return
	}
	m.SetStr(RadioNode, 0, "radioUnpair", sn)
}

// GainProp is the gain property that counts for an input node: the digital
// level of the sources without a preamp, the instrument gain while the
// microphone choice is "Instrument", else the microphone gain (RØDE's
// gain panel binds the three the same way; line-in uses the microphone
// gain).
func (m *Model) GainProp(input int) string {
	if input < 0 {
		return ""
	}
	switch {
	case m.Int(tInput, input, "inputType") < 0:
		if m.Has(tInput, input, "inputDigitalGain") {
			return "inputDigitalGain"
		}
	case m.MicType(input) == MicInstrument && m.Has(tInput, input, "inputInstrumentGain"):
		return "inputInstrumentGain"
	}
	if m.Has(tInput, input, "inputMicrophoneGain") {
		return "inputMicrophoneGain"
	}
	if m.Has(tInput, input, "inputDigitalGain") {
		return "inputDigitalGain"
	}
	return ""
}

// MicInstrument and MicLineIn are the carousel entries that make a combo
// input an instrument or line input.
const (
	MicLineIn     = 0
	MicInstrument = 1
)

// USB microphone hosting on USB 2 (SYSTEM): the mixer reports the hosted
// device's kind in usbHostOnUsb2 (0 none) and podMicOnUSB2.
func (m *Model) USBMicHosted() bool {
	return m.Int(tSystem, 0, "usbHostOnUsb2") != 0 || m.Bool(tSystem, 0, "podMicOnUSB2")
}

// HostedChannel is the CHANNEL node of the USB microphone hosted on USB 2
// (RØDE's Usb2HostedChannel): the desk keeps it after the fader channels,
// with the USB 2 source and its own processing, whose high-pass filter is
// the microphone's own (hpfOn 60 Hz, hpfLowerOn 75 Hz, hpfHigherOn 150 Hz).
// −1 when the desk has no such node.
func (m *Model) HostedChannel() int {
	n := m.Count(ChannelNode)
	for i := m.StripCount(); i < n; i++ {
		if m.Int(ChannelNode, i, "channelInputSource") == SourceUSB2 {
			return i
		}
	}
	return -1
}
