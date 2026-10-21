package app_test

import (
	"strings"
	"testing"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/pages"
)

// TestBluetoothPageDiscoverable: the Bluetooth page makes the desk
// discoverable while it is open and withdraws that on leaving.
func TestBluetoothPageDiscoverable(t *testing.T) {
	a, model := newTestApp(t)
	a.Open(pages.IDBluetooth)
	a.RenderOnce()
	if !model.Bool(desk.NetworkNode, 0, "btVisible") {
		t.Fatal("opening the Bluetooth page must set NETWORK.btVisible")
	}
	a.Pop()
	a.RenderOnce()
	if model.Bool(desk.NetworkNode, 0, "btVisible") {
		t.Fatal("leaving the Bluetooth page must clear NETWORK.btVisible")
	}
	a.Open(pages.IDBluetooth)
	a.Home()
	if model.Bool(desk.NetworkNode, 0, "btVisible") {
		t.Fatal("Home must also clear NETWORK.btVisible")
	}
}

// TestWifiJoinScans: the network join page asks for a scan on entry.
func TestWifiJoinScans(t *testing.T) {
	a, model := newTestApp(t)
	a.Push(pages.CallMePage("network-join"))
	a.RenderOnce()
	if !model.Bool(desk.NetworkNode, 0, "wifiScan") {
		t.Fatal("the join page must request NETWORK.wifiScan")
	}
}

// TestHostingToggleWrites: Turn Off Hosting writes the hosting flag.
func TestHostingToggleWrites(t *testing.T) {
	a, model := newTestApp(t)
	a.Push(pages.CallMePage("hosting"))
	a.RenderOnce()
	tap(a, 640, 652) // the red button at the bottom
	a.RenderOnce()
	if model.Bool("SIPCALLING", 0, "sipCallHostingEnabled") {
		t.Fatal("Turn Off Hosting must clear SIPCALLING.sipCallHostingEnabled")
	}
	tap(a, 640, 652) // now Turn On Hosting
	a.RenderOnce()
	if !model.Bool("SIPCALLING", 0, "sipCallHostingEnabled") {
		t.Fatal("Turn On Hosting must set SIPCALLING.sipCallHostingEnabled")
	}
}

// TestAdvancedToggleWrites: a switch on the advanced page writes its
// SIPADVANCED property.
func TestAdvancedToggleWrites(t *testing.T) {
	a, model := newTestApp(t)
	a.Push(pages.CallMePage("advanced"))
	a.RenderOnce()
	if !model.Bool("SIPADVANCED", 0, "forceCodecChoice") {
		t.Fatal("fixture: forceCodecChoice starts true")
	}
	tap(a, 1165, 488) // the Force Codec Choice switch (fourth row)
	a.RenderOnce()
	if model.Bool("SIPADVANCED", 0, "forceCodecChoice") {
		t.Fatal("the switch must clear SIPADVANCED.forceCodecChoice")
	}
}

// TestIncomingCallPopup: an incoming call shows the popup while hosting is
// on; the red cross declines with sip_slot -1; the popup goes when the
// desk withdraws the call.
func TestIncomingCallPopup(t *testing.T) {
	a, model := newTestApp(t)
	a.Frame() // primes the watcher
	model.SetStr("SIPCALLING", 0, "sipIncomingCallDetails", `{"id": 42, "name": "Studio B"}`)
	a.Frame()
	kind, ok := pages.IsCallMePopup(a.Overlay())
	if !ok || kind != "incoming" {
		t.Fatalf("overlay %v %q, want the incoming popup", ok, kind)
	}
	tap(a, 460, 490) // the red cross
	a.RenderOnce()
	if got := model.Str("SIPCALLING", 0, "sipIncomingCallAccept"); got != `{"id":42,"sip_slot":-1}` {
		t.Fatalf("decline wrote %q", got)
	}
	if a.Overlay() != nil {
		t.Fatal("declining closes the popup")
	}
	// A second call, withdrawn by the desk.
	model.SetStr("SIPCALLING", 0, "sipIncomingCallDetails", `{"id": 43, "name": "Studio C"}`)
	a.Frame()
	if _, ok := pages.IsCallMePopup(a.Overlay()); !ok {
		t.Fatal("the second call shows the popup again")
	}
	model.SetStr("SIPCALLING", 0, "sipIncomingCallDetails", "")
	a.Frame()
	if a.Overlay() != nil {
		t.Fatal("a withdrawn call takes its popup away")
	}
}

// TestIncomingAcceptOnLite: on the Lite licence the tick answers into slot
// 1 when it is free and on a fader.
func TestIncomingAcceptOnLite(t *testing.T) {
	a, model := newTestApp(t)
	model.SetSource(6, 16) // CallMe 1 on strip 7
	a.Frame()
	model.SetStr("SIPCALLING", 0, "sipIncomingCallDetails", `{"id": 5, "name": "Guest"}`)
	a.Frame()
	tap(a, 820, 490) // the green tick
	if got := model.Str("SIPCALLING", 0, "sipIncomingCallAccept"); got != `{"id":5,"sip_slot":0}` {
		t.Fatalf("accept wrote %q", got)
	}
}

// TestTimeLimitAndRating: an expiring call brings the warning (Extend
// writes the slot's extend flag); a call that ends brings the rating.
func TestTimeLimitAndRating(t *testing.T) {
	a, model := newTestApp(t)
	a.Frame()
	model.SetStr("SIPCALLSLOTS", 1, "sipSlotCallState", `{"state": "Expiring", "reason": ""}`)
	a.Frame()
	if kind, ok := pages.IsCallMePopup(a.Overlay()); !ok || kind != "timelimit" {
		t.Fatalf("overlay %v %q, want the time-limit warning", ok, kind)
	}
	tap(a, 830, 470) // Extend
	if !model.Bool("SIPCALLSLOTS", 1, "sipSlotCallExtend") {
		t.Fatal("Extend must set SIPCALLSLOTS[1].sipSlotCallExtend")
	}
	model.SetStr("SIPCALLSLOTS", 1, "sipSlotCallState", `{"state": "Idle", "reason": "Remote drop"}`)
	a.Frame()
	if kind, ok := pages.IsCallMePopup(a.Overlay()); !ok || kind != "rating" {
		t.Fatalf("overlay %v %q, want the rating", ok, kind)
	}
	tap(a, 565, 300) // the fourth star
	got := model.Str("SIPCALLING", 0, "sipCallRating")
	if !strings.Contains(got, `"rating": 4`) || !strings.HasPrefix(got, `{"uuid": "`) {
		t.Fatalf("rating wrote %q", got)
	}
}

// TestCallMeChannelPage: the channel page of a CallMe strip shows the
// CallMe panels and leads to the CallMe menu.
func TestCallMeChannelPage(t *testing.T) {
	a, _ := newTestApp(t)
	a.Push(pages.NewChannel(5)) // the fixture's CallMe 2 strip
	a.RenderOnce()
	tap(a, 840, 644) // CallMe Configuration
	a.RenderOnce()
	if got := a.PageTitle(); got != "CALLME CONFIGURATION" {
		t.Fatalf("page %q, want the CallMe menu", got)
	}
}
