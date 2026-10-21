package pages

import "testing"

func TestParseCallState(t *testing.T) {
	cases := []struct {
		in     string
		state  callState
		reason string
	}{
		{"Idle", callIdle, ""},
		{"", callIdle, ""},
		{"Connecting", callConnecting, ""},
		{"Active", callActive, ""},
		{"Expiring", callExpiring, ""},
		{`{"state": "Active", "reason": ""}`, callActive, ""},
		{`{"state":"Idle","reason":"486 Rejected"}`, callIdle, "486 Rejected"},
		{`{"state":"Connecting"}`, callConnecting, ""},
	}
	for _, c := range cases {
		st, reason := parseCallState(c.in)
		if st != c.state || reason != c.reason {
			t.Errorf("parseCallState(%q) = %v %q, want %v %q", c.in, st, reason, c.state, c.reason)
		}
	}
}

func TestSlotView(t *testing.T) {
	s := callSlot{Strip: -1, State: callActive}
	if s.view() != slotNotAssigned {
		t.Fatal("a slot without a fader is not assigned, whatever its state")
	}
	s = callSlot{Strip: 2, State: callIdle, Reason: "404 Not Found"}
	if s.view() != slotFailedAddress {
		t.Fatal("404 shows 'Check Address'")
	}
	s.Reason = "486 Rejected"
	if s.view() != slotFailedRejected {
		t.Fatal("486 shows 'Host rejected'")
	}
	s.Reason = "408 Request Timeout"
	if s.view() != slotFailedOffline {
		t.Fatal("408 shows 'Host offline?'")
	}
	s.Reason = "Local drop"
	if s.view() != slotReady {
		t.Fatal("a dropped call leaves the slot ready")
	}
	s.State = callExpiring
	if s.view() != slotConnected {
		t.Fatal("an expiring call is still connected")
	}
}

func TestLicenceSlots(t *testing.T) {
	for mode, want := range map[int64]int{-1: 1, 0: 1, 1: 2, 2: 3, 7: 3} {
		if got := licenceSlots(mode); got != want {
			t.Errorf("licenceSlots(%d) = %d, want %d", mode, got, want)
		}
	}
}

func TestCallHours(t *testing.T) {
	for in, want := range map[int64]string{600: "10:00", 300: "5:00", 65: "1:05", 0: "0:00", -1: "0:00", 9: "0:09"} {
		if got := callHours(in); got != want {
			t.Errorf("callHours(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestJSONShapes(t *testing.T) {
	if got := dialJSON("ABC123", 1); got != `{"dest":"ABC123","sip_slot":1}` {
		t.Fatalf("dialJSON: %s", got)
	}
	if got := dialSIPJSON("123.456@zoomcrc.com", "Opus", "udp", 128000, 0); got != `{"dest":"123.456@zoomcrc.com","codec":"Opus","transport":"udp","bitrate":128000,"sip_slot":0}` {
		t.Fatalf("dialSIPJSON: %s", got)
	}
	if got := acceptJSON(42, -1); got != `{"id":42,"sip_slot":-1}` {
		t.Fatalf("acceptJSON: %s", got)
	}
	if got := ratingJSON("u-1", true, 4); got != `{"uuid": "u-1", "outgoing": true, "rating": 4}` {
		t.Fatalf("ratingJSON: %s", got)
	}
	if !isStreamAddress("icecasts://host/mount") || isStreamAddress("sip:user@host") {
		t.Fatal("stream address detection")
	}
}

func TestParseIncoming(t *testing.T) {
	if _, ok := parseIncoming(""); ok {
		t.Fatal("empty details are no call")
	}
	c, ok := parseIncoming(`{"id": 7, "name": "Studio B"}`)
	if !ok || c.ID != 7 || c.Name != "Studio B" {
		t.Fatalf("parsed %+v", c)
	}
	c, ok = parseIncoming("plain text")
	if !ok || c.Name != "plain text" {
		t.Fatalf("plain details keep the text: %+v", c)
	}
}

func TestEnumTables(t *testing.T) {
	if labelOf(incomingRoutingOpts, "nextavailable") != "First Available" || labelOf(dtmfOpts, "rtpevent") != "RTP (RFC 4733)" {
		t.Fatal("advanced option captions")
	}
	if labelOf(registerOpts, "enabledNoKeepAlive") != "Enabled (No Keepalive)" || labelOf(natOpts, "stun") != "STUN" {
		t.Fatal("register / NAT captions")
	}
	if indexOf(transportOpts, "tls") != 3 || indexOf(transportOpts, "x") != -1 {
		t.Fatal("transport index")
	}
	if bitrateLabel(510000) != "510kbp/s" || defaultBitrateFor("Opus") != 128000 || defaultBitrateFor("G722") != 64000 {
		t.Fatal("bitrates")
	}
	if len(codecOpts) != 5 || codecOpts[2].value != "G711a" {
		t.Fatal("codec table")
	}
}
