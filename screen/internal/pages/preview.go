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

// RoutingMix returns the mix page of an output (previews and tests).
func RoutingMix(output int) Page { return newRoutingMix(output) }
