package pages

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
)

// CallMe: the facts shared by the CallMe pages, popups and the CallMe
// channel panels. The desk keeps three call slots (SIPCALLSLOTS[0..2], the
// sources CallMe 1–3), the service state in SIPCALLING and the settings in
// SIPADVANCED; the quick dials are SIPPHONEBOOK children of
// SIPPHONEBOOKARRAY. Everything here follows RØDE's own screen program:
// the state words it parses, the texts it shows and the JSON it writes.

// Node types of the CallMe area (the desk's own spelling).
const (
	sipCallingNode      = "SIPCALLING"
	sipSlotsNode        = "SIPCALLSLOTS"
	sipAdvancedNode     = "SIPADVANCED"
	sipRegistrationNode = "SIPREGISTRATION"
	sipPhoneBookNode    = "SIPPHONEBOOK"
	sipPhoneBookArray   = "SIPPHONEBOOKARRAY"
	proxyNode           = "PROXY"
	wifiScanNode        = "WIFISCANRESULT"
)

// callMeSlots is how many call slots the desk has.
const callMeSlots = 3

// callState is a slot's call state as RØDE's screen sorts the desk's state
// word: idle, connecting, active or expiring (the last hour's end).
type callState int

const (
	callIdle callState = iota
	callConnecting
	callActive
	callExpiring
)

// parseCallState reads SIPCALLSLOTS[k].sipSlotCallState. The desk writes
// either a plain word ("Idle", "Connecting", "Active", "Expiring") or a
// JSON object {"state": "...", "reason": "..."}; the reason names why a
// call ended ("404 Not Found", "408 Request Timeout", "486 Rejected",
// "Remote drop", "Local drop").
func parseCallState(s string) (callState, string) {
	word, reason := s, ""
	if strings.HasPrefix(strings.TrimSpace(s), "{") {
		var obj map[string]interface{}
		if json.Unmarshal([]byte(s), &obj) == nil {
			word, _ = obj["state"].(string)
			reason, _ = obj["reason"].(string)
		}
	}
	switch {
	case strings.Contains(word, "Connecting"):
		return callConnecting, reason
	case strings.Contains(word, "Active"):
		return callActive, reason
	case strings.HasPrefix(word, "Expiring"):
		return callExpiring, reason
	}
	return callIdle, reason
}

// slotView is what a slot's widget shows (the state machine of RØDE's call
// widgets): not assigned to a fader, ready, connecting, connected, or one
// of three failures.
type slotView int

const (
	slotNotAssigned slotView = iota
	slotReady
	slotConnecting
	slotConnected
	slotFailedAddress  // "Call failed: Check Address"
	slotFailedRejected // "Call failed: Host rejected"
	slotFailedOffline  // "Call failed: Host offline?"
)

// failureOf maps a call's end reason to the failure the widgets show. The
// SIP codes are the screen program's own: 404 means the address was not
// found, 486 that the host rejected the call, 408 that it never answered.
func failureOf(reason string) slotView {
	switch {
	case strings.HasPrefix(reason, "404"):
		return slotFailedAddress
	case strings.HasPrefix(reason, "486"):
		return slotFailedRejected
	case strings.HasPrefix(reason, "408"):
		return slotFailedOffline
	}
	return slotReady
}

// callSlot is a snapshot of one slot.
type callSlot struct {
	K       int // 0-based slot
	Strip   int // the strip carrying CallMe K+1, or -1
	State   callState
	Reason  string
	IsHost  bool
	Address string // the other end's name or code (sipSlotCallAddress)
	UUID    string
	Allowed bool // within the licence
}

// callMeSource is the source index of CallMe k+1 (the source id "callmeN"),
// or -1.
func callMeSource(env *Env, k int) int64 {
	id := "callme" + strconv.Itoa(k+1)
	for i, s := range env.Desk.Sources() {
		if s.ID == id {
			return int64(i)
		}
	}
	return -1
}

// callMeStrip is the strip carrying CallMe k+1, or -1.
func callMeStrip(env *Env, k int) int {
	src := callMeSource(env, k)
	if src < 0 {
		return -1
	}
	for i := 0; i < env.Desk.StripCount(); i++ {
		if env.Desk.SourceOf(i) == src {
			return i
		}
	}
	return -1
}

// callMeSlotOfStrip is the slot (0-based) whose source sits on a strip, or
// -1 when the strip carries something else.
func callMeSlotOfStrip(env *Env, strip int) int {
	id := env.Desk.SourceName(env.Desk.SourceOf(strip)).ID
	if !strings.HasPrefix(id, "callme") {
		return -1
	}
	n, err := strconv.Atoi(strings.TrimPrefix(id, "callme"))
	if err != nil || n < 1 || n > callMeSlots {
		return -1
	}
	return n - 1
}

// licenceSlots is how many slots the licence allows: Lite (0) one, CallMe
// (1) two, Pro (2) three.
func licenceSlots(mode int64) int {
	n := int(mode) + 1
	if n < 1 {
		n = 1
	}
	if n > callMeSlots {
		n = callMeSlots
	}
	return n
}

// slotOf reads slot k.
func slotOf(env *Env, k int) callSlot {
	d := env.Desk
	st, reason := parseCallState(d.Str(sipSlotsNode, k, "sipSlotCallState"))
	return callSlot{
		K:       k,
		Strip:   callMeStrip(env, k),
		State:   st,
		Reason:  reason,
		IsHost:  d.Bool(sipSlotsNode, k, "sipSlotCallIsHost"),
		Address: d.Str(sipSlotsNode, k, "sipSlotCallAddress"),
		UUID:    d.Str(sipSlotsNode, k, "sipSlotCallUUID"),
		Allowed: k < licenceSlots(d.Int(sipCallingNode, 0, "sipLicenceMode")),
	}
}

// view is what the slot's widget shows.
func (s callSlot) view() slotView {
	if s.Strip < 0 {
		return slotNotAssigned
	}
	switch s.State {
	case callConnecting:
		return slotConnecting
	case callActive, callExpiring:
		return slotConnected
	}
	if f := failureOf(s.Reason); f != slotReady {
		return f
	}
	return slotReady
}

// inCall reports whether the slot carries a call (connecting or up).
func (s callSlot) inCall() bool { return s.State != callIdle }

// hosting reports whether this device hosts the call (the other end joined
// us), which the widgets word as "Is Hosting" instead of "is Connected".
func (s callSlot) hosting(env *Env) bool {
	return !s.IsHost && s.Address == env.Desk.Str(sipCallingNode, 0, "sipRodeCode")
}

// callMeOnline reports whether the CallMe service is reachable: registered
// with RØDE's service and the device on a network.
func callMeOnline(env *Env) bool {
	d := env.Desk
	if d.Bool(sipRegistrationNode, 0, "sipRegistrationIsRegistered") {
		return true
	}
	return d.Bool(desk.NetworkNode, 0, "wiredConnected") && d.Str(desk.NetworkNode, 0, "ipAddress") != "" ||
		d.Bool(desk.NetworkNode, 0, "wifi") && d.Str(desk.NetworkNode, 0, "wifiIpAddress") != ""
}

// hoursText formats remaining minutes as RØDE's "H:MM" ("0:00" below zero).
func callHours(minutes int64) string {
	if minutes < 0 {
		return "0:00"
	}
	return fmt.Sprintf("%01d:%02d", minutes/60, minutes%60)
}

// --- what the screen writes ---

// dialJSON is the call request for a RØDECaster code or a stream address
// (SIPCALLING.sipOutgoingCallDetails).
func dialJSON(dest string, slot int) string {
	return `{"dest":"` + dest + `","sip_slot":` + strconv.Itoa(slot) + `}`
}

// dialSIPJSON is the call request for a SIP address with its codec,
// transport and bitrate.
func dialSIPJSON(dest, codec, transport string, bitrate int64, slot int) string {
	return `{"dest":"` + dest + `","codec":"` + codec + `","transport":"` + transport +
		`","bitrate":` + strconv.FormatInt(bitrate, 10) + `,"sip_slot":` + strconv.Itoa(slot) + `}`
}

// isStreamAddress reports whether a dial address is a stream (RTSP or
// Icecast) rather than a SIP address; streams dial without codec details.
func isStreamAddress(dest string) bool {
	for _, p := range []string{"rtsp://", "icecast://", "rtsps://", "icecasts://"} {
		if strings.HasPrefix(dest, p) {
			return true
		}
	}
	return false
}

// acceptJSON answers an incoming call (SIPCALLING.sipIncomingCallAccept):
// the call's id and the slot it goes to, or -1 to decline.
func acceptJSON(id int64, slot int) string {
	return `{"id":` + strconv.FormatInt(id, 10) + `,"sip_slot":` + strconv.Itoa(slot) + `}`
}

// ratingJSON is the post-call rating (SIPCALLING.sipCallRating).
func ratingJSON(uuid string, outgoing bool, rating int) string {
	return `{"uuid": "` + uuid + `", "outgoing": ` + strconv.FormatBool(outgoing) + `, "rating": ` + strconv.Itoa(rating) + `}`
}

// incomingCall is the parsed SIPCALLING.sipIncomingCallDetails: the call's
// id and the caller's name as far as the desk gives them.
type incomingCall struct {
	ID   int64
	Name string
	Raw  string
}

// parseIncoming reads the incoming call details (a JSON object with "id";
// the caller's name is taken from the first text field found).
func parseIncoming(s string) (incomingCall, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return incomingCall{}, false
	}
	call := incomingCall{Raw: s}
	var obj map[string]interface{}
	if json.Unmarshal([]byte(s), &obj) != nil {
		call.Name = s
		return call, true
	}
	switch v := obj["id"].(type) {
	case float64:
		call.ID = int64(v)
	case string:
		call.ID, _ = strconv.ParseInt(v, 10, 64)
	}
	for _, key := range []string{"name", "display_name", "caller", "from", "address", "dest"} {
		if v, ok := obj[key].(string); ok && v != "" {
			call.Name = v
			break
		}
	}
	return call, true
}

// --- enumerations of the dialling and the advanced page ---

// choiceOpt is one value of a text-valued setting with its caption.
type choiceOpt struct {
	value string
	label string
}

// labelOf is the caption of a value, or the value itself when unknown.
func labelOf(opts []choiceOpt, value string) string {
	for _, o := range opts {
		if o.value == value {
			return o.label
		}
	}
	return value
}

// indexOf is the position of a value in a list, or -1.
func indexOf(opts []choiceOpt, value string) int {
	for i, o := range opts {
		if o.value == value {
			return i
		}
	}
	return -1
}

// The codecs of a SIP call (SIPPHONEBOOK.codecType) and the bitrates
// (SIPPHONEBOOK.bitRate, bit/s) with their captions.
var codecOpts = []choiceOpt{{"Opus", "Opus"}, {"G722", "G722"}, {"G711a", "G711a"}, {"G711u", "G711u"}, {"PCM", "PCM"}}

var bitrateOpts = []struct {
	value int64
	label string
}{
	{8000, "8kbp/s"}, {16000, "16kbp/s"}, {32000, "32kbp/s"}, {64000, "64kbp/s"},
	{128000, "128kbp/s"}, {256000, "256kbp/s"}, {510000, "510kbp/s"},
}

// defaultBitrateFor is the bitrate the screen sets when the codec changes:
// Opus takes 128 kbit/s, the narrow codecs 64 kbit/s.
func defaultBitrateFor(codec string) int64 {
	if codec == "Opus" {
		return 128000
	}
	return 64000
}

func bitrateLabel(v int64) string {
	for _, b := range bitrateOpts {
		if b.value == v {
			return b.label
		}
	}
	return strconv.FormatInt(v/1000, 10) + "kbp/s"
}

// The SIP transports (SIPPHONEBOOK.transportType and
// SIPADVANCED.sipAccountTransport).
var transportOpts = []choiceOpt{{"auto", "Auto"}, {"udp", "UDP"}, {"tcp", "TCP"}, {"tls", "TLS"}}

// dialTransportOpts are the transports offered when dialling (the same
// four; the dial pages write "auto", "udp", "tcp" or "tls").
var dialTransportOpts = transportOpts

// Settings of the advanced page with a fixed list of text values.
var (
	incomingRoutingOpts = []choiceOpt{{"callme1", "CallMe Slot 1"}, {"callme2", "CallMe Slot 2"}, {"callme3", "CallMe Slot 3"}, {"drop", "Drop"}, {"nextavailable", "First Available"}}
	dtmfOpts            = []choiceOpt{{"auto", "Auto"}, {"rtpevent", "RTP (RFC 4733)"}, {"info", "SIP INFO"}}
	natOpts             = []choiceOpt{{"none", "None"}, {"stun", "STUN"}, {"ice", "ICE"}}
	registerOpts        = []choiceOpt{{"enabled", "Enabled"}, {"enabledNoKeepAlive", "Enabled (No Keepalive)"}, {"disabled", "Disabled"}}
	qualityOpts         = []choiceOpt{{"good", "Good"}, {"excellent", "Excellent"}}
	icecastBitrateOpts  = []struct {
		value int64
		label string
	}{{64000, "64kbp/s"}, {128000, "128kbp/s"}, {192000, "192kbp/s"}, {256000, "256kbp/s"}, {320000, "320kbp/s"}}
)

// phoneBookEntry is one quick dial (a SIPPHONEBOOK node).
type phoneBookEntry struct {
	Index     int // node index
	Idx       int64
	Name      string
	Address   string
	CallType  int64 // 0 RØDECaster code, 1 SIP address
	Transport string
	Codec     string
	Bitrate   int64
}

const (
	callTypeRode = 0
	callTypeSIP  = 1
)

// phoneBook lists the saved quick dials in node order.
func phoneBook(env *Env) []phoneBookEntry {
	d := env.Desk
	n := d.Count(sipPhoneBookNode)
	out := make([]phoneBookEntry, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, phoneBookEntry{
			Index:     i,
			Idx:       d.Int(sipPhoneBookNode, i, "phoneBookIdx"),
			Name:      d.Str(sipPhoneBookNode, i, "entryName"),
			Address:   d.Str(sipPhoneBookNode, i, "phoneBookCallAddress"),
			CallType:  d.Int(sipPhoneBookNode, i, "phoneBookCallType"),
			Transport: d.Str(sipPhoneBookNode, i, "transportType"),
			Codec:     d.Str(sipPhoneBookNode, i, "codecType"),
			Bitrate:   d.Int(sipPhoneBookNode, i, "bitRate"),
		})
	}
	return out
}

// quickDialSlots is how many quick dials the dial page shows (a 2 × 4 grid).
const quickDialSlots = 8

// dial places a call on a slot: a SIP address goes with its codec details,
// a RØDECaster code or a stream address without.
func dial(env *Env, slot int, e phoneBookEntry) {
	d := env.Desk
	if e.CallType == callTypeSIP && !isStreamAddress(e.Address) {
		d.SetStr(sipCallingNode, 0, "sipOutgoingCallDetails", dialSIPJSON(e.Address, e.Codec, e.Transport, e.Bitrate, slot))
	} else {
		d.SetStr(sipCallingNode, 0, "sipOutgoingCallDetails", dialJSON(e.Address, slot))
	}
}

// hangUp ends the call on a slot (the slot's disconnect trigger).
func hangUp(env *Env, slot int) { env.Desk.Trigger(sipSlotsNode, slot, "sipSlotCallDisconnect") }

// firstFreeSlot is the lowest licensed slot on a fader without a call, or -1.
func firstFreeSlot(env *Env) int {
	for k := 0; k < callMeSlots; k++ {
		s := slotOf(env, k)
		if s.Allowed && s.Strip >= 0 && !s.inCall() {
			return k
		}
	}
	return -1
}
