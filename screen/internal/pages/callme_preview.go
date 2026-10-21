package pages

import "github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"

// CallMePage returns a CallMe, network or Bluetooth page by a short name
// (previews and tests); unknown names give the CallMe menu.
func CallMePage(name string) Page {
	switch name {
	case "hosting":
		return newCallMeHosting()
	case "activate":
		return newCallMeActivation()
	case "advanced":
		return newCallMeAdvanced()
	case "dial":
		return newCallMeDialHub(0)
	case "quickdial":
		return newQuickDialPage(0, phoneBookEntry{Index: -1, CallType: callTypeSIP, Transport: "udp", Codec: "Opus", Bitrate: 128000}, false, false)
	case "quickdial-saved":
		return newQuickDialPage(0, phoneBookEntry{Index: 1, Idx: 1, Name: "Weekly Zoom", Address: "1234.5678@zoomcrc.com", CallType: callTypeSIP, Transport: "tls", Codec: "G722", Bitrate: 64000}, true, false)
	case "sipdial":
		return newQuickDialPage(0, phoneBookEntry{CallType: callTypeSIP, Transport: "udp", Codec: "Opus", Bitrate: 128000}, false, true)
	case "fader":
		return newCallMeFaderAssign(0, nil)
	case "assign":
		return newCallMeAssignIncoming(incomingCall{ID: 7, Name: "Studio B"})
	case "network-join":
		return newNetworkJoin()
	case "network-advanced":
		return newNetworkAdvanced()
	case "network-dns":
		return newNetworkDNS()
	case "network-proxy":
		return newNetworkProxy()
	}
	return newCallMeConfig()
}

// CallMePopup returns a CallMe popup by name (previews and tests).
func CallMePopup(env *Env, name string) ui.View {
	switch name {
	case "incoming":
		return NewCallMeIncoming(env, incomingCall{ID: 7, Name: "Studio B"})
	case "conflict":
		return newCallerConflict(env, 1, nil)
	case "recording":
		return newRecordingWarning(env)
	case "welcome":
		return NewCallMeWelcome(env)
	case "rating":
		return newCallRating(env, "uuid-1", true, nil)
	case "timelimit":
		return newTimeLimitWarning(env, 1)
	}
	return nil
}
