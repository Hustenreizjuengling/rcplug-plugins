package desk

import (
	"testing"
	"time"
)

func TestDecodeEncoderSignal(t *testing.T) {
	cases := []struct {
		hex   string
		delta int
		ival  int64
		bval  bool
	}{
		{"01050100000000010102", 0, 0, true},    // idle sync: int 0 → no step
		{"01050101000000010103", 1, 1, false},   // one detent right (the bool is a flip)
		{"010501ffffffff010102", -1, -1, true},  // one detent left
		{"010501fdffffff010103", -3, -3, false}, // three detents left (a quick turn)
		{"0105017f000000010102", 32, 127, true}, // capped
		{"010501", 0, 0, false},                 // cut short: nothing
	}
	for _, c := range cases {
		raw := mustHex(t, c.hex)
		delta, ival, bval, _ := decodeEncoderSignal(raw)
		if delta != c.delta || ival != c.ival || bval != c.bval {
			t.Errorf("%s: delta %d int %d bool %v, want %d %d %v", c.hex, delta, ival, bval, c.delta, c.ival, c.bval)
		}
	}
}

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	out := make([]byte, 0, len(s)/2)
	for i := 0; i+1 < len(s); i += 2 {
		var b byte
		for _, c := range s[i : i+2] {
			b <<= 4
			switch {
			case c >= '0' && c <= '9':
				b |= byte(c - '0')
			case c >= 'a' && c <= 'f':
				b |= byte(c-'a') + 10
			}
		}
		out = append(out, b)
	}
	return out
}

func TestEncoderEvents(t *testing.T) {
	m := New("http://127.0.0.1:0", nil)
	m.onEncoderProp("encoderSignal", Value{Kind: 'x', S: "01050102000000010102"})
	m.onEncoderProp("encoderPressed", Bool(true))
	m.onEncoderProp("encoderPressed", Bool(false))
	want := []string{"turn 2", "press", "release"}
	for _, w := range want {
		select {
		case ev := <-m.Encoder():
			got := "release"
			switch {
			case ev.Delta != 0:
				got = "turn " + itoa(ev.Delta)
			case ev.Press:
				got = "press"
			}
			if got != w {
				t.Fatalf("got %q, want %q", got, w)
			}
			if w == "release" && ev.Held < 0 {
				t.Fatal("negative hold")
			}
		case <-time.After(time.Second):
			t.Fatalf("no event for %q", w)
		}
	}
}

func itoa(n int) string {
	if n < 0 {
		return "-" + itoa(-n)
	}
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}

// TestChannelButton: the strip's number is sent once when the button goes
// down (systemChannelSelected leaves -1), never when it comes up.
func TestChannelButton(t *testing.T) {
	m := New("http://127.0.0.1:0", nil)
	for _, v := range []int64{0, -1, 2, -1, -1} {
		m.onChannelSelected(Int64(v))
	}
	var got []int
	for {
		select {
		case n := <-m.ChannelButton():
			got = append(got, n)
			continue
		default:
		}
		break
	}
	if len(got) != 2 || got[0] != 0 || got[1] != 2 {
		t.Fatalf("button presses %v, want [0 2]", got)
	}
}
