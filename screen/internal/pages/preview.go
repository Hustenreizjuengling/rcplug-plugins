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
