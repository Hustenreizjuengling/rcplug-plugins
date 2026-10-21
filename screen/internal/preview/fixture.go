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
var fixtureChannelSources = []int64{0, 1, 7, 10, 11, 17, 19, 2, 20, -1}
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
			"channelDepth":              0.5, "channelSparkle": 0.35, "channelPunch": 0.4, "channelCurrentFxPreset": int64(1),
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
		"systemDateTime24h": true, "systemDateTimeOnHome": true, "systemChannelSelected": -1, "systemDateTimeDaylightSavings": false, "systemDateTimezone": int64(49), "transferModeType": int64(0), "assignableMeterSource": 0,
		"osUpdateAvailable": false, "appUpdateAvailable": false, "usb1Connected": true,
		"disableAllHeadphoneOutputs": false, "disableAllLineoutOutputs": false,
		"usbHostOnUsb2": int64(0), "podMicOnUSB2": false, "usbHostUnspportedFirmware": false,
		// Settings and requests of the system pages (values as on a real Duo).
		"systemBetaMode": false, "shareAnom": true, "shareResource": true, "boardType": int64(1),
		"updateCheckRequested": false, "updateChecking": false, "updateNoInternet": false, "updateVersion": "",
		"downloadInitiateRequested": false, "downloadCancelRequested": false, "updateDownloadProgress": int64(0),
		"updateDownloaded": false, "updateInitiateRequested": false, "updateInstalledProgress": int64(0), "updateComplete": false,
		"updateRebootRequested": false, "updateResetDeviceRequested": false, "updateViaUSB": false, "powerOffRequest": false,
		"rcSyncModeRequest": false, "rcSyncState": int64(0), "rcSyncDeviceModel": int64(-1), "rcSyncCaptureMode": false,
		"systemMainMode": int64(0), "engineMode": int64(0), "systemMixSelected": int64(13),
	}))
	// Headphone outputs (four in the tree, two on a Duo): port 1 set to
	// Low Sensitivity as on the reference device, the ring colours as hex.
	for i, typ := range []int64{2, 0, 0, 0} {
		kids = append(kids, leaf("HEADPHONE", map[string]interface{}{"headphoneType": typ, "headphoneColour": []string{"ff3bd33b", "ff1e62e6", "ffffffff", "ffffffff"}[i]}))
	}
	kids = append(kids, leaf("BUILD", map[string]interface{}{
		"buildMixerVersion": "1.7.6+", "buildMixerModulesGitSha": "0000000", "buildGuiVersion": "1.7.6", "buildGuiModulesGitSha": "0000000", "buildCallMeVersion": "1.7.6",
	}))
	kids = append(kids, leaf("OAUTH", map[string]interface{}{
		"requestSignIn": false, "requestSignOut": false, "responsePending": false, "hasValidToken": false,
		"verificationURL": "", "verificationURLBare": "", "userCode": "", "userName": "", "errorMessage": "",
		"cloudUploadRequestUID": int64(0), "cloudUploadCancelUID": int64(0),
	}))
	kids = append(kids, leaf("RECORDER", map[string]interface{}{
		"recordState": 0, "recordTimeMs": 0, "requestDropMarker": 0, "requestRecordState": 255, "recordBytesPerSecond": 384000,
	}))
	kids = append(kids, leaf("MASTERCHANNEL", map[string]interface{}{
		"masterCompellorOn": true, "masterCompellorThreshold": 0.5, "masterCompellorAttack": 0.4,
		"masterCompellorRelease": 0.5, "masterCompellorGain": 0.3, "masterDelayOn": false, "masterDelaySeconds": 0.0,
	}))
	kids = append(kids, leaf("DUCKER", map[string]interface{}{"duckerDepth": -9.0}))
	// The processing presets (the first two as a Duo with 1.7.6 holds them).
	var presets []node
	for i, c := range []string{
		"Neutral|0.3|0.35|0.2|0.3|0.3|0.2|0|1|0.811844|0.315|0.345|0.27|1|0.713333|0.408248|0.422549|0.38903|0.0666667|1|0.755929|0.566667|0.471405|0.422549|0.38903|0|1|0.283333|0.61928|0|0.375804|0.161245|0.2|1|1|0.666667|0.020202|0.5375|1|0.333333|0.040404|0.48125|1|0.894427|0.020202|0.5375|1|0.527046|0.333333|1|0.5|0|1",
		"Podcast Studio|0.75|0.8|0.7|0.7|0.75|0.7|0|1|0.811844|0.72|0.345|0.675|1|0.58|0.408248|0.422549|0.38903|0.233333|1|0.755929|0.525|0.471405|0.422549|0.38903|0|1|0.366667|0.61928|0|0.375804|0.301662|0.2|1|1|0.666667|0.020202|0.59375|1|0.333333|0.040404|0.453125|1|0.894427|0.020202|0.59375|1|0.527046|0.333333|1|0.5|0|1",
		"Broadcast|0.6|0.6|0.6|0.6|0.6|0.6|0|1|0.811844|0.5|0.345|0.5|1|0.65|0.408248|0.422549|0.38903|0.15|1|0.755929|0.55|0.471405|0.422549|0.38903|0|1|0.33|0.61928|0|0.375804|0.25|0.2|1|1|0.666667|0.020202|0.55|1|0.333333|0.040404|0.47|1|0.894427|0.020202|0.55|1|0.527046|0.333333|1|0.5|0|1",
		"Voice Over|0.5|0.5|0.5|0.5|0.5|0.5|0|1|0.811844|0.4|0.345|0.4|1|0.6|0.408248|0.422549|0.38903|0.1|1|0.755929|0.5|0.471405|0.422549|0.38903|0|1|0.3|0.61928|0|0.375804|0.2|0.2|1|1|0.666667|0.020202|0.5|1|0.333333|0.040404|0.5|1|0.894427|0.020202|0.5|1|0.527046|0.333333|1|0.5|0|1",
	} {
		presets = append(presets, leaf("FXPRESET", map[string]interface{}{"fxPresetContents": c, "fxPresetIdx": int64(i)}))
	}
	kids = append(kids, node{T: "FXPRESETS", P: map[string]interface{}{}, C: presets})
	// The headphone knobs and the current show.
	for _, lv := range []int64{46, 8} {
		kids = append(kids, leaf("POT", map[string]interface{}{"potMin": int64(0), "potMax": int64(127), "potLevel": lv}))
	}
	kids = append(kids, leaf("CURRENTSHOW", map[string]interface{}{"currentShowName": "Meine Show", "currentShowIcon": int64(0), "currentShowUUID": ""}))
	// The main knob (idle: int 0 + true) and the recording storage.
	kids = append(kids, leaf("ENCODER", map[string]interface{}{"encoderSignal": "0105010000000001" + "0102", "encoderPressed": false, "encoderColour": 7}))
	// The three volumes: microSD, USB storage, the pads' internal storage.
	kids = append(kids, leaf("STORAGEVOLUME", map[string]interface{}{
		"storageVolumeInserted": true, "storageVolumeMounted": true, "storageVolumeFormatted": true,
		"storageVolumeCapacity": 64e9, "storageVolumeFree": 41e9, "storageVolumeRecDestination": true,
		"storageVolumeEject": false, "storageVolumeErase": false, "storageVolumeTransfer": false, "storageVolumeName": "", "storageVolumeState": "64000000000|41000000000|0|1|1",
	}))
	kids = append(kids, leaf("STORAGEVOLUME", map[string]interface{}{
		"storageVolumeInserted": false, "storageVolumeMounted": false, "storageVolumeFormatted": false,
		"storageVolumeCapacity": 0, "storageVolumeFree": 0, "storageVolumeRecDestination": true,
		"storageVolumeEject": false, "storageVolumeErase": false, "storageVolumeTransfer": false, "storageVolumeName": "", "storageVolumeState": "0|0|0|0|0",
	}))
	kids = append(kids, leaf("STORAGEVOLUME", map[string]interface{}{
		"storageVolumeInserted": true, "storageVolumeMounted": true, "storageVolumeFormatted": true,
		"storageVolumeCapacity": 5277638656, "storageVolumeFree": 5064294400, "storageVolumeRecDestination": false,
		"storageVolumeEject": false, "storageVolumeErase": false, "storageVolumeName": "", "storageVolumeState": "5277638656|5064294400|0|1|1",
	}))
	// 30 input sources with gain/power/colour.
	for i := 0; i < 30; i++ {
		p := map[string]interface{}{"inputId": i, "inputType": 0}
		switch {
		case i < 4: // combo mic inputs
			p["inputMicrophoneGain"] = int64(38 + i*2)
			p["inputMicrophoneType"] = []int64{4, 8, 10, 0}[i] // PodMic, NT1-A, SM7B, line-in
			p["inputPower"] = 1
			p["inputPhaseFlip"] = false
			p["inputInstrumentGain"] = 0
		case i == 19 || i == 20: // the internal wireless receiver's channels (1.7.3+)
			p["inputType"] = int64(3)
			p["inputMicrophoneGain"] = int64(50)
			p["inputMicrophoneType"] = int64(11)
			p["inputPower"] = 0
			p["inputPhaseFlip"] = false
			p["inputInstrumentGain"] = 0
			p["inputDigitalGain"] = 0
			p["inputWirelessSN"] = ""
			p["inputColour"] = "ffad00ff"
		default:
			p["inputDigitalGain"] = 0
			p["inputPower"] = -1
			if i >= 21 {
				p["inputType"] = int64(-1)
			}
		}
		if i >= 16 && i <= 18 { // CallMe 1–3
			p["inputType"] = -1
			p["inputSipCallSlot"] = int64(i - 16)
		}
		if i < len(fixtureColours) {
			p["inputColour"] = fixtureColours[i]
		}
		kids = append(kids, leaf("INPUTSOURCE", p))
	}
	// The wireless receiver: slot 0 has a Wireless ME paired, slot 1 is free.
	kids = append(kids, node{T: "RADIO", P: map[string]interface{}{"radioPair": int64(-1), "radioPaired": int64(1), "radioUnpair": ""}, C: []node{
		leaf("RADIOTX", map[string]interface{}{
			"txConnectionId": int64(0), "txDeviceSN": "TX000001", "txDeviceType": int64(2), "txConnected": true,
			"txRssi": int64(-48), "txSignalQuality": int64(4), "txBatteryLevel": int64(72), "txChargingState": false,
			"txRecord": false, "txRemoteMute": false, "txGainAssist": int64(0), "txPad": false,
		}),
		leaf("RADIOTX", map[string]interface{}{
			"txConnectionId": int64(1), "txDeviceSN": "", "txDeviceType": int64(-1), "txConnected": false,
			"txRssi": int64(0), "txSignalQuality": int64(0), "txBatteryLevel": int64(0), "txChargingState": false,
			"txRecord": false, "txRemoteMute": false, "txGainAssist": int64(2), "txPad": false,
		}),
	}})
	kids = append(kids, leaf("RADIORX", map[string]interface{}{"rxRadioId": ""}))
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
		"Nachhall", "Roboter", "Zensur", "Ducking", "MIDI", "Ausblenden",
		"CallMe 13", "Scene A", "CallMe Icecast"}
	padTypes := []int64{1, 1, 1, 2, 3, 1, 2, 2, 3, 3, 4, 3, 5, 6, 7}
	padMixerModes := []int64{0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 0, 2, 0, 0, 0}
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
			"padFilePath": "", "padMixerCensorFilePath": "", "padIsInternal": false,
			"padRCVSyncPadType": int64(0), "padSIPCallSlot": int64(0), "padSIPFlashState": int64(0), "padSIPPhoneBookEntry": int64(0), "padSIPQdLock": false,
			"padProgressRequestSignal": "0109040000000000000000" + "0102",
		}
		if i < 6 || i == 2 {
			p["padFilePath"] = "/Application/emmc-data/pads/" + strconv.Itoa(i+1) + "/sound.wav"
		}
		if i == 13 {
			p["padRCVSyncPadType"] = int64(13)
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
	// The network: wired with an address, Wi-Fi idle with a scan's results,
	// a Bluetooth device paired, the proxy off.
	network := map[string]interface{}{
		"wiredConnected": true, "wifi": false, "ipAddress": "192.168.0.50", "subnetMask": "255.255.255.0", "gateway": "192.168.0.1",
		"primaryDns": "192.168.0.1", "secondaryDns": "", "staticIpSet": false, "wifiDHCP": true,
		"wifiSSID": "", "wifiPSK": "", "wifiIncorrectPSK": false, "wifiScan": false, "wifiIpAddress": "", "wifiSubnetMask": "", "wifiGateway": "",
		"cellUSBFound": false, "cellEnabled": false, "cellAPN": "", "cellIpAddress": "", "cellSubnetMask": "", "cellGateway": "",
		"btConnectedAddress": "00:11:22:33:44:55 - Studio Speaker", "btConnectedType": 0, "btVisible": false, "btDoScan": false,
		"btDoPair": "", "btDoConnect": "", "btDoDisconnect": "", "btDoUnPair": "", "btPairCode": "",
		"btPairedNumber1": "00:11:22:33:44:55 - Studio Speaker", "btPairedNumber2": "66:77:88:99:AA:BB - Phone",
	}
	for i := 1; i <= 10; i++ {
		network["wifiScan"+strconv.Itoa(i)] = ""
		network["btScan"+strconv.Itoa(i)] = ""
	}
	network["btScan1"] = "CC:DD:EE:FF:00:11 - Headphones"
	for i := 3; i <= 5; i++ {
		network["btPairedNumber"+strconv.Itoa(i)] = ""
	}
	var scan []node
	for i := 0; i < 15; i++ {
		ssid := ""
		switch i {
		case 0:
			ssid = "Studio----PSK"
		case 1:
			ssid = "Guest"
		case 2:
			ssid = "Old Router----WEP"
		}
		scan = append(scan, leaf("WIFISCANRESULT", map[string]interface{}{"wifiScanResultSSID": ssid}))
	}
	scan = append(scan, leaf("PROXY", map[string]interface{}{"proxyEnabled": false, "proxyAddress": "", "proxyPort": int64(1080), "proxyUsername": "", "proxyPassword": ""}))
	kids = append(kids, node{T: "NETWORK", P: network, C: scan})
	// CallMe: three slots (the second on a fader with a call up), the
	// service on the Lite tier with hosting on, two quick dials.
	for k := 0; k < 3; k++ {
		state := "Idle"
		addr := ""
		if k == 1 {
			state, addr = `{"state": "Active", "reason": ""}`, "Studio B"
		}
		kids = append(kids, leaf("SIPCALLSLOTS", map[string]interface{}{
			"sipCallSlotId": int64(k), "sipSlotCallState": state, "sipSlotCallUUID": "", "sipSlotCallAddress": addr,
			"sipSlotCallQuality": int64(0), "sipSlotCallJitter": 0.0, "sipSlotCallBitrate": 0.0, "sipSlotCallPacketLoss": 0.0,
			"sipSlotCallDisconnect": "010102010102", "sipSlotCallExtend": false, "sipSlotCallIsHost": k == 1, "sipSlotCallMode": int64(0),
		}))
	}
	kids = append(kids, node{T: "SIPCALLING", P: map[string]interface{}{
		"sipCallHostingEnabled": true, "sipCallHostingToggleEnabled": true, "sipLicenceMode": int64(0), "sipLicenceCheck": false,
		"sipRodeCode": "AB12CD", "sipIncomingCallAccept": "", "sipIncomingCallDetails": "", "sipOutgoingCallDetails": "",
		"sipSlotPendingCallSetup": int64(-1), "sipRemainingCalltime": int64(600), "sipRemainingWebCalltime": int64(300),
		"sipRenewalDate": "October 27, 2026", "sipLicenceRenewal": "", "sipCallRating": "",
	}, C: []node{
		leaf("SIPREGISTRATION", map[string]interface{}{"sipRegistrationIndex": int64(0), "sipRegistrationDetails": "", "sipRegistrationIsRegistered": true}),
		leaf("SIPREGISTRATION", map[string]interface{}{"sipRegistrationIndex": int64(1), "sipRegistrationDetails": "", "sipRegistrationIsRegistered": false}),
	}})
	kids = append(kids, leaf("SIPADVANCED", map[string]interface{}{
		"forceCodecChoice": true, "enableVideoStream": true, "enableRFCDuplication": false, "incomingNameFilter": true, "usbNumberPad": true,
		"nonHTTPSInterface": true, "autoReconnect": false, "incomingAudioRouting": "nextavailable", "dtmfMode": "auto",
		"receiveJitterBufferMin": int64(100), "receiveJitterBufferMax": int64(5000), "sipWebPassword": "", "sipAccountRegister": "disabled",
		"sipAccountUsername": "", "sipAccountPassword": "", "sipAccountDomain": "", "sipAccountProxyAddress": "", "sipAccountTransport": "auto",
		"sipNATTraversalMode": "none", "sipNATTraversalServer": "", "sipNATTraversalUsername": "", "sipNATTraversalPassword": "",
		"sipUnitName": "", "sipAccountAuthUsername": "", "rodeCallQuality": "good", "sipCallKeepalive": true,
		"icecastAddress": "", "icecastPort": int64(8000), "icecastMountpoint": "", "icecastUsername": "", "icecastPassword": "",
		"icecastTLS": false, "icecastBitrate": int64(128000), "icecastConnect": false, "icecastState": "",
	}))
	kids = append(kids, node{T: "SIPPHONEBOOKARRAY", P: map[string]interface{}{}, C: []node{
		leaf("SIPPHONEBOOK", map[string]interface{}{"phoneBookIdx": int64(0), "phoneBookCallAddress": "XY34ZW", "phoneBookCallType": int64(0), "entryName": "Studio B", "transportType": "udp", "codecType": "Opus", "bitRate": int64(128000)}),
		leaf("SIPPHONEBOOK", map[string]interface{}{"phoneBookIdx": int64(1), "phoneBookCallAddress": "1234.5678@zoomcrc.com", "phoneBookCallType": int64(1), "entryName": "Weekly Zoom", "transportType": "tls", "codecType": "G722", "bitRate": int64(64000)}),
	}})
	// The pad recorder, the recording player and the recordings and shows
	// the desk lists (values as a Duo 1.7.6 holds them).
	kids = append(kids, leaf("PADRECORDER", map[string]interface{}{
		"padRecordIdx": int64(0), "padRecordState": int64(0), "padRecordSeconds": 0.0,
		"padRecordStateRequest": "01050100000000010102", "padRecordClear": "01050100000000010103", "padRecordMemoryFull": "010103010102",
	}))
	kids = append(kids, leaf("PLAYER", map[string]interface{}{
		"playerState": int64(0), "playerSpeed": int64(0), "playerFilePath": "", "playerFileSize": 0.0,
		"playerCurrentPositionTime": 4.0, "playerProgress": 0.18, "playerJumpToSample": "01050100000000010103",
		"playerEnvStart": 0.0, "playerEnvFadeIn": 0.0, "playerEnvFadeOut": 1.0, "playerEnvStop": 1.0,
	}))
	// A recording's content is RØDE's fourteen "|" fields: name, hash, path,
	// show, time (ms), length (s), multitrack, two flags, the envelope and
	// the upload status (desk.ParseRecording).
	var recs []node
	for i, c := range []string{
		"Recording 24|3f2a9c|/media/sd/RODECaster/Recording 24.wav|Morgenrunde|1652324700000|22.5|1|0|0|0|0|1|1|0",
		"Recording 23|8b1d04|/media/sd/RODECaster/Recording 23.wav|Morgenrunde|1652238300000|903.2|0|0|0|0|0|1|1|0",
		"Recording 22|c07e51|/media/sd/RODECaster/Recording 22.wav|Interview|1652151900000|6275.8|1|0|0|0.05|0.1|0.9|0.95|0",
	} {
		recs = append(recs, leaf("RECORDING", map[string]interface{}{"recordingUID": int64(100 + i), "recordingContent": c}))
	}
	kids = append(kids, node{T: "RECORDINGS", P: map[string]interface{}{"recordingTotalCount": 3, "recordingTotalDuration": 7200, "requestDeleteUID": int64(0)}, C: recs})
	var shows []node
	for i, n := range []string{"Morgenrunde", "Interview"} {
		shows = append(shows, leaf("SHOW", map[string]interface{}{
			"showName": n, "showIcon": int64(i + 1), "showUID": int64(10 + i), "showUUID": "", "showStorageType": int64(0),
			"showSnapshotIndex": int64(i + 1), "showLastModification": int64(1652324700 - i*86400),
		}))
	}
	kids = append(kids, node{T: "SHOWS", P: map[string]interface{}{}, C: shows})
	kids = append(kids, leaf("SHOWCONTROL", map[string]interface{}{
		"showControlUpdating": false, "showControlExport": false, "showControlImport": int64(-1), "showControlDelete": int64(-1),
		"showControlExportImport": int64(-1), "showControlNewFromDefault": false, "showControlNewFromDefaultMuted": false,
		"showControlLockoutCentral": false, "showControlProgress": 0.0, "showControlLastError": int64(0),
	}))

	root := node{T: "Rodecaster", P: map[string]interface{}{}, C: kids}
	doc := map[string]interface{}{
		"epoch": 1, "seq": 1, "stale": false,
		"meta": map[string]interface{}{
			"layout": map[string]interface{}{
				"model": "duo", "modelName": "RØDECaster Duo", "firmware": "1.7.6",
				"physicalFaders": 4, "padsPerBank": 6, "padBanks": 8, "mixPerSource": 13, "sourceCount": 19, "micInputs": 2, "headphones": 2,
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
