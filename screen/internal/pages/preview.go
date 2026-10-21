package pages

// Detail returns a settings detail page by a short name (previews and
// tests); unknown names give the metering page.
func Detail(name string) Page {
	switch name {
	case "listen":
		return newListenPage()
	case "faders":
		return newFadersPage()
	case "language":
		return newLanguagePage()
	case "datetime":
		return newDateTimePage()
	case "brightness":
		return newBrightnessPage()
	case "haptics":
		return newHapticsPage()
	case "monitor":
		return newMonitorPage()
	case "multitrack":
		return newMultitrackPage()
	case "bluetooth":
		return newBluetoothOutputPage()
	case "headphones":
		return newHeadphonesPage()
	case "recbutton":
		return newRecButtonPage()
	case "compression":
		return newCompressionPage()
	case "midi":
		return newMidiPage()
	case "info":
		return newInfoMainPage()
	case "info-device":
		return newInfoDevicePage()
	case "regulatory":
		return newRegulatoryPage()
	case "analytics":
		return newAnalyticsPage()
	case "firmware":
		return newFirmwareUpdatePage()
	case "opmode":
		return newOperatingModePage()
	case "storage":
		return newStoragePage()
	case "timezone":
		return newTimeZonePage(&dateTimePage{init: true, zone: 49})
	case "cloud":
		return newCloudPage()
	case "numpad":
		return NewNumPad("Port", 1080, 1, 65535, nil)
	}
	return newMeteringPage()
}

// PadPage returns a pad's options, new-pad or colour page by name (previews
// and tests).
func PadPage(idx int, name string) Page {
	switch name {
	case "new":
		return newPadNew(idx)
	case "colour":
		return newPadColour(idx)
	}
	return newPadOptions(idx)
}

// RoutingMix returns the mix page of an output (previews and tests).
func RoutingMix(output int) Page { return newRoutingMix(output) }
