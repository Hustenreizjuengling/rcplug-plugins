// Package preview runs the interface without a device: a browser server and
// a PNG renderer over an offline desk model filled with a representative
// RØDECaster Duo state. The icons and text come from a local folder of the
// device's resources passed at runtime (never shipped), so this package
// carries no RØDE assets.
package preview

import (
	"encoding/json"
	"math"
	"strconv"
)

// node is one tree node in the console's {t,p,c} shape.
type node struct {
	T string                 `json:"t"`
	P map[string]interface{} `json:"p"`
	C []node                 `json:"c,omitempty"`
}

func leaf(t string, p map[string]interface{}) node { return node{T: t, P: p} }

var fixtureSources = []map[string]string{
	{"id": "combo1", "label": "Mic 1"}, {"id": "combo2", "label": "Mic 2"},
	{"id": "combo3", "label": "Mic 3"}, {"id": "combo4", "label": "Mic 4"},
	{"id": "combo1_2", "label": "Mic 1+2"}, {"id": "combo2_3", "label": "Mic 2+3"},
	{"id": "combo3_4", "label": "Mic 3+4"}, {"id": "usb1", "label": "USB 1"},
	{"id": "chat", "label": "Chat"}, {"id": "usb2", "label": "USB 2"},
	{"id": "bluetooth", "label": "Bluetooth"}, {"id": "soundpad", "label": "SMART Pads"},
	{"id": "game", "label": "Game"}, {"id": "music", "label": "Music"},
	{"id": "virtuala", "label": "Virtual A"}, {"id": "virtualb", "label": "Virtual B"},
	{"id": "callme1", "label": "Call 1"}, {"id": "callme2", "label": "Call 2"},
	{"id": "callme3", "label": "Call 3"},
}

var fixtureOutputs = []map[string]string{
	{"id": "headphone1", "label": "Headphones 1"}, {"id": "headphone2", "label": "Headphones 2"},
	{"id": "headphone3", "label": "Headphones 3"}, {"id": "headphone4", "label": "Headphones 4"},
	{"id": "speaker", "label": "Speakers"}, {"id": "recording", "label": "Recording"},
	{"id": "bluetooth", "label": "Bluetooth"}, {"id": "usb1", "label": "USB 1"},
	{"id": "chat", "label": "Chat"}, {"id": "usb2", "label": "USB 2"},
	{"id": "callme1", "label": "Call 1"}, {"id": "callme2", "label": "Call 2"},
	{"id": "callme3", "label": "Call 3"},
}

// fixture faders: 4 physical + 5 virtual.
var fixtureFaders = []map[string]string{
	{"id": "physical1", "label": "Fader 1"}, {"id": "physical2", "label": "Fader 2"},
	{"id": "physical3", "label": "Fader 3"}, {"id": "physical4", "label": "Fader 4"},
	{"id": "virtual1", "label": "Virtual 1"}, {"id": "virtual2", "label": "Virtual 2"},
	{"id": "virtual3", "label": "Virtual 3"}, {"id": "virtual4", "label": "Virtual 4"},
	{"id": "virtual5", "label": "Virtual 5"},
}

// sources assigned to the 10 channels in the fixture (−1 = not assigned).
var fixtureChannelSources = []int64{0, 1, 7, 10, 11, 17, -1, -1, -1, -1}
var fixtureColours = []string{"ff7ad43a", "ff33a3ff", "ffe5322d", "fff0a830", "ffb060ff", "ff30d0c0"}
var fixtureFaderLevels = []int64{96, 90, 104, 80, 90, 64, 40, 90, 90}
var fixtureMeterL = []float64{0.35, 0.12, 0.06, 0.02, 0.2, 0.0, 0.0, 0.0, 0.0}

// BuildTree returns a representative Duo tree document for the offline model.
func BuildTree() []byte {
	var kids []node

	// 10 channels with processing defaults and a source.
	for i := 0; i < 10; i++ {
		src := int64(-1)
		if i < len(fixtureChannelSources) {
			src = fixtureChannelSources[i]
		}
		p := map[string]interface{}{
			"channelInputSource":        src,
			"channelOutputMute":         i == 3,
			"channelCueEnable":          false,
			"channelBypassProcessing":   i >= 4,
			"channelAdvancedProcessing": false,
			"channelDepth":              0.5, "channelSparkle": 0.35, "channelPunch": 0.4,
			"channelPan": 0.5, "channelPanOn": false, "channelPanL": 0.0, "channelPanR": 1.0,
			"hpfOn": i == 0, "hpfFrequency": 0.25, "hpfSlope": 0.33,
			"noiseGateOn": false, "noiseGateThreshold": 0.3, "noiseGateRange": 0.5,
			"noiseGateHysteresis": 0.2, "noiseGateAttack": 0.1, "noiseGateHold": 0.3, "noiseGateRelease": 0.4,
			"compressorOn": i == 0, "compressorThreshold": 0.45, "compressorRatio": 0.4,
			"compressorAttack": 0.3, "compressorRelease": 0.5, "compressorGain": 0.3,
			"deesserOn": false, "deesserFrequency": 0.6, "deesserThreshold": 0.4, "deesserRatio": 0.3,
			"deesserAttack": 0.3, "deesserRelease": 0.4, "deesserGain": 0.2,
			"eqOn": false, "eqLowOn": true, "eqMidOn": true, "eqHighOn": true,
			"eqLowGain": 0.5, "eqLowShelf": 0.3, "eqLowQ": 0.5,
			"eqMidGain": 0.5, "eqMidBell": 0.4, "eqMidQ": 0.5,
			"eqHighGain": 0.5, "eqHighBell": 0.6, "eqHighQ": 0.5,
			"aphexOn": false, "aphexAEMix": 0.3, "aphexAETune": 0.5, "aphexBBDrive": 0.3, "aphexBBTune": 0.4,
		}
		kids = append(kids, leaf("CHANNEL", p))
	}
	// 9 faders.
	for i := 0; i < 9; i++ {
		lvl := int64(90)
		if i < len(fixtureFaderLevels) {
			lvl = fixtureFaderLevels[i]
		}
		kids = append(kids, leaf("FADER", map[string]interface{}{"faderLevel": lvl, "faderMin": 0, "faderMax": 127}))
	}
	// 22 meters (strips 0..8, master 9, focus 10..12, GR 13..21).
	for i := 0; i < 22; i++ {
		l := 0.0
		if i < len(fixtureMeterL) {
			l = fixtureMeterL[i]
		}
		if i == 9 {
			l = 0.6
		}
		if i >= 13 {
			l = 1
			if i == 13 {
				l = 0.5 // 6 dB of gain reduction on strip 0
			}
		}
		stereo := i == 9 || i == 2 || i == 4
		kids = append(kids, leaf("METER", map[string]interface{}{
			"meterLevelL": l, "meterLevelR": l * 0.8, "meterPeakL": math.Min(1, l*1.3), "meterPeakR": l, "meterStereo": stereo,
		}))
	}
	// Output, GUI, System, Recorder, Master.
	kids = append(kids, leaf("OUTPUT", map[string]interface{}{
		"outputMonLevel": 0.7, "outputMonMute": false, "outputMonAutoMute": true, "outputMonFixed": false,
		"outputBTLevel": 0.5, "outputBTMute": false, "outputBTAutoMute": false,
		"outputPrefader": false, "recordingMultitrackMode": 0, "recordingProcessingBypass": false,
		"outputMultiMode": 0, "outputMultiBypass": false, "recordingCompressionQuality": 0, "headphone1Level": 90,
		"outputMonAutoMuteActive": false, "outputBTAutoMuteActive": false,
	}))
	kids = append(kids, leaf("GUI", map[string]interface{}{
		"selectedBank": 0, "metering": 0, "broadcastMeters": false, "lang": "de",
		"screenBrightness": 200, "activeButtonsBrightness": 180, "inactiveButtonsBrightness": 60,
		"autoBrightness": false, "screenDimAfterSeconds": false,
		"eqParamModeLow": 0, "eqParamModeMid": 0, "eqParamModeHigh": 0, "padActiveEdit": int64(-1), "screenTouched": false,
	}))
	kids = append(kids, leaf("SYSTEM", map[string]interface{}{
		"systemFirmwareVersion": "1.7.6", "systemSerialNumber": "", "systemName": "RODECaster",
		"systemHapticSetting": 1, "systemRecButtonSetting": 0, "systemMidiControl": false,
		"systemDateTime24h": true, "systemDateTimeOnHome": true, "systemChannelSelected": -1, "assignableMeterSource": 0,
		"osUpdateAvailable": false, "appUpdateAvailable": false, "usb1Connected": true,
		"disableAllHeadphoneOutputs": false,
	}))
	kids = append(kids, leaf("RECORDER", map[string]interface{}{
		"recordState": 0, "recordTimeMs": 0, "requestDropMarker": 0, "requestRecordState": 255, "recordBytesPerSecond": 384000,
	}))
	kids = append(kids, leaf("MASTERCHANNEL", map[string]interface{}{
		"masterCompellorOn": true, "masterCompellorThreshold": 0.5, "masterCompellorAttack": 0.4,
		"masterCompellorRelease": 0.5, "masterCompellorGain": 0.3, "masterDelayOn": false, "masterDelaySeconds": 0.0,
	}))
	kids = append(kids, leaf("DUCKER", map[string]interface{}{"duckerDepth": -9.0}))
	// The main knob (idle: int 0 + true) and the recording storage.
	kids = append(kids, leaf("ENCODER", map[string]interface{}{"encoderSignal": "0105010000000001" + "0102", "encoderPressed": false, "encoderColour": 7}))
	kids = append(kids, leaf("STORAGEVOLUME", map[string]interface{}{
		"storageVolumeInserted": true, "storageVolumeMounted": true, "storageVolumeFormatted": true,
		"storageVolumeCapacity": 64e9, "storageVolumeFree": 41e9, "storageVolumeRecDestination": true,
	}))
	kids = append(kids, leaf("STORAGEVOLUME", map[string]interface{}{
		"storageVolumeInserted": false, "storageVolumeMounted": false, "storageVolumeFormatted": false,
		"storageVolumeCapacity": 0, "storageVolumeFree": 0, "storageVolumeRecDestination": false,
	}))
	// 30 input sources with gain/power/colour.
	for i := 0; i < 30; i++ {
		p := map[string]interface{}{"inputId": i, "inputType": 0}
		switch {
		case i < 4: // combo mic inputs
			p["inputMicrophoneGain"] = int64(38 + i*2)
			p["inputMicrophoneType"] = 4
			p["inputPower"] = 1
			p["inputPhaseFlip"] = false
			p["inputInstrumentGain"] = 0
		default:
			p["inputDigitalGain"] = 0
			p["inputPower"] = -1
		}
		if i < len(fixtureColours) {
			p["inputColour"] = fixtureColours[i]
		}
		kids = append(kids, leaf("INPUTSOURCE", p))
	}
	// 11 effects slots, one per strip (as the Duo keeps them).
	for i := 0; i < 11; i++ {
		kids = append(kids, leaf("EFFECTS_PARAMETERS", map[string]interface{}{
			"effectsIdx": int64(i),
			"reverbOn":   i == 0, "reverbMix": 0.5, "reverbLowCut": 0.6661, "reverbHighCut": 0.3333, "reverbModel": 0.6,
			"echoOn": false, "echoMix": 0.5, "echoDelay": 0.165, "echoDecay": 0.5, "echoLowCut": 0.5, "echoHighCut": 0.5,
			"distortionOn": false, "distortionIntensity": 0.7, "robotOn": false, "robotLevel": 0.0, "robotMix": 0.0,
			"voiceDisguiseOn": false, "pitchShiftOn": false, "pitchShiftSemitones": 0.0,
		}))
	}
	// 13 mix-minus modes (one per output): Headphones 1 is a custom mix.
	for i := 0; i < 13; i++ {
		mode := 0
		if i == 0 {
			mode = 2
		}
		kids = append(kids, leaf("MIXMINUSES", map[string]interface{}{"outputMixMinus": mode}))
	}
	// The send matrix: 19 sources × 13 outputs; the custom Headphones 1 mix
	// has an unlinked, lower USB 1 and a muted chat.
	for s := 0; s < len(fixtureSources); s++ {
		for o := 0; o < 13; o++ {
			lv := 0.708661
			link, mute, dis := true, false, false
			if o == 0 && s == 7 {
				lv, link = 0.45, false
			}
			if o == 0 && s == 8 {
				mute = true
			}
			if (o == 7 && s == 7) || (o == 9 && s == 9) || (o == 6 && s == 10) {
				dis = true
			}
			kids = append(kids, leaf("MIX", map[string]interface{}{
				"mixLevelWithAnchor": fmtMix(lv) + "|" + fmtMix(0.708661), "mixLink": link,
				"mixLinkRequest": false, "mixUnlinkRequest": false, "mixMute": mute, "mixDisabled": dis,
			}))
		}
	}
	// 21 pads with every field the device keeps: bank 0 named, bank 1 the
	// factory effects and mixer actions, the rest empty.
	padNames := []string{"Intro", "Applaus", "Lacher", "Jingle", "Stinger", "Outro",
		"Nachhall", "Roboter", "Zensur", "Ducking", "MIDI", "Ausblenden"}
	padTypes := []int64{1, 1, 1, 2, 3, 1, 2, 2, 3, 3, 4, 3}
	padMixerModes := []int64{0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 0, 2}
	for i := 0; i < 21; i++ {
		p := map[string]interface{}{
			"padIdx": i, "padColourIndex": int64(i % 12), "padActive": false, "padName": "", "padType": int64(0),
			"padLoop": false, "padReplay": true, "padPlayMode": int64(0), "padProgress": 0.0, "padGain": -12.0,
			"padEnvStart": 0.0, "padEnvFadeIn": 0.0, "padEnvFadeOut": 1.0, "padEnvStop": 1.0,
			"padMixerMode": int64(0), "padMixerTriggerMode": int64(0), "padMixerCensorCustom": false,
			"padMixerFadeInSeconds": 2.0, "padMixerFadeOutSeconds": 5.0, "padMixerFadeExcludeHost": false,
			"padMixerBackChannelMic2": false, "padMixerBackChannelMic3": false, "padMixerBackChannelMic4": false,
			"padMixerBackChannelUsb1Comms": false, "padMixerBackChannelUsb2Main": false, "padMixerBackChannelBluetooth": false,
			"padMixerBackChannelCallMe1": false, "padMixerBackChannelCallMe2": false, "padMixerBackChannelCallMe3": false,
			"padEffectInput": int64(0), "padEffectTriggerMode": int64(1),
			"padTriggerMode": int64(1), "padTriggerSend": int64(2), "padTriggerType": int64(0), "padTriggerControl": int64(i),
			"padTriggerChannel": int64(1), "padTriggerOn": int64(127), "padTriggerOff": int64(0), "padTriggerCustom": false,
		}
		if i < len(padNames) {
			p["padName"] = padNames[i]
			p["padType"] = padTypes[i]
			p["padMixerMode"] = padMixerModes[i]
			if i == 2 {
				p["padEnvStart"], p["padEnvFadeIn"], p["padEnvFadeOut"], p["padEnvStop"] = 0.1, 0.25, 0.8, 0.95
				p["padPlayMode"], p["padLoop"] = int64(0), true
			}
		}
		kids = append(kids, leaf("PAD", p))
	}
	// The effects pads' settings (under PADEFFECTS on the device).
	var padFx []node
	for _, i := range []int64{3, 6, 7} {
		padFx = append(padFx, leaf("EFFECTS_PARAMETERS", map[string]interface{}{
			"effectsIdx": i, "reverbOn": i == 3 || i == 6, "reverbMix": 0.5, "reverbLowCut": 0.6661, "reverbHighCut": 0.3333, "reverbModel": 0.4,
			"echoOn": i == 3, "echoMix": 0.3, "echoDelay": 0.165, "echoDecay": 0.5, "echoLowCut": 0.5, "echoHighCut": 0.5,
			"distortionOn": false, "distortionIntensity": 0.7, "robotOn": i == 7, "robotLevel": 0.2, "robotMix": 0.6667,
			"voiceDisguiseOn": false, "pitchShiftOn": false, "pitchShiftSemitones": 7.0,
		}))
	}
	kids = append(kids, node{T: "PADEFFECTS", P: map[string]interface{}{}, C: padFx})
	// The pad buttons (one per slot, as the Duo's eight).
	for i := 0; i < 8; i++ {
		kids = append(kids, leaf("PADBUTTON", map[string]interface{}{"padButtonPressed": false, "padButtonPreview": false}))
	}
	kids = append(kids, leaf("NETWORK", map[string]interface{}{
		"wiredConnected": true, "wifi": false, "ipAddress": "192.168.0.50", "btConnectedAddress": "", "btConnectedType": 0,
	}))
	kids = append(kids, leaf("RECORDINGS", map[string]interface{}{"recordingTotalCount": 3, "recordingTotalDuration": 7200}))

	root := node{T: "Rodecaster", P: map[string]interface{}{}, C: kids}
	doc := map[string]interface{}{
		"epoch": 1, "seq": 1, "stale": false,
		"meta": map[string]interface{}{
			"layout": map[string]interface{}{
				"model": "duo", "modelName": "RØDECaster Duo", "firmware": "1.7.6",
				"physicalFaders": 4, "padsPerBank": 6, "padBanks": 8, "mixPerSource": 13, "sourceCount": 19, "micInputs": 2,
			},
			"faders": fixtureFaders, "sources": fixtureSources, "outputs": fixtureOutputs, "unityFader": 90,
		},
		"root": root,
	}
	b, _ := json.Marshal(doc)
	return b
}

// fmtMix formats one half of a send level as the desk does.
func fmtMix(v float64) string { return strconv.FormatFloat(v, 'f', 6, 64) }
