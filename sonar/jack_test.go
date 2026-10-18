package main

import (
	"context"
	"errors"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

// jack_lsp -p as JACK2 prints it, with ports named like those on the Duo.
const jackLspSample = `system:capture_1
	properties: output,physical,terminal,
system:capture_2
	properties: output,physical,terminal,
system:capture_10
	properties: output,physical,terminal,
system:capture_3
	properties: output,physical,terminal,
system:playback_1
	properties: input,physical,terminal,
JUCEJack:in_1
	properties: input,
JUCEJack:out_1
	properties: output,
JUCEJack:out_2
	properties: output,
rcplug-sonar:playback_1
	properties: input,physical,terminal,
weird port with spaces:out
	properties: output,
`

func TestParseJackOutputs(t *testing.T) {
	got := parseJackOutputs(jackLspSample)
	want := []string{"system:capture_1", "system:capture_2", "system:capture_10", "system:capture_3", "JUCEJack:out_1", "JUCEJack:out_2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("outputs = %v", got)
	}
}

func TestJackPairs(t *testing.T) {
	got := jackPairs(append(parseJackOutputs(jackLspSample), "rcplug-sonar:monitor_1", "rcplug-sonar:monitor_2"))
	want := []string{"system:capture_1,system:capture_2", "system:capture_3,system:capture_10", "JUCEJack:out_1,JUCEJack:out_2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("pairs = %v", got)
	}
}

func TestParseJackSpec(t *testing.T) {
	if l, r, err := parseJackSpec("system:capture_1,system:capture_2"); err != nil || l != "system:capture_1" || r != "system:capture_2" {
		t.Fatalf("got %q %q %v", l, r, err)
	}
	for _, bad := range []string{"", "system:capture_1", "system:capture_1,system:capture_1", "a:b,c", "a:b;rm -rf /,c:d", "x:y,z:$(id)"} {
		if _, _, err := parseJackSpec(bad); err == nil {
			t.Errorf("parseJackSpec(%q) accepted", bad)
		}
	}
	if _, err := validateSource("jack:system:capture_1,system:capture_2", t.TempDir()); err != nil {
		t.Fatalf("validateSource: %v", err)
	}
	if _, err := validateSource("jack:system:capture_1", t.TempDir()); err == nil {
		t.Fatal("validateSource accepted one port")
	}
}

func TestLoopbackCard(t *testing.T) {
	duo := strings.Join([]string{
		" 0 [RODECasterDUO  ]: axg-sound-card - RODECaster DUO",
		"                      RODECaster DUO",
	}, "\n")
	if got := loopbackCard(duo); got != "" {
		t.Fatalf("no loopback on a stock Duo, got %q", got)
	}
	ours := duo + "\n 7 [RcplugLoop     ]: Loopback - Loopback\n                      Loopback 1\n"
	if got := loopbackCard(ours); got != loopCard {
		t.Fatalf("got %q", got)
	}
	other := duo + "\n 1 [Loopback       ]: Loopback - Loopback\n                      Loopback 1\n"
	if got := loopbackCard(other); got != "Loopback" {
		t.Fatalf("an existing loopback card is reused, got %q", got)
	}
}

func TestLoopSubstream(t *testing.T) {
	files := map[string]string{
		"/proc/asound/RcplugLoop/pcm0p/sub0/status": "state: RUNNING\nowner_pid   : 1443\n",
		"/proc/asound/RcplugLoop/pcm1c/sub0/status": "closed\n",
		"/proc/asound/RcplugLoop/pcm0p/sub1/status": "closed\n",
		"/proc/asound/RcplugLoop/pcm1c/sub1/status": "closed\n",
	}
	read := func(p string) ([]byte, error) {
		if b, ok := files[p]; ok {
			return []byte(b), nil
		}
		return nil, os.ErrNotExist
	}
	if n, err := loopSubstream(read, "RcplugLoop"); err != nil || n != 1 {
		t.Fatalf("got %d, %v; want substream 1 (0 is held by another program)", n, err)
	}
	files["/proc/asound/RcplugLoop/pcm1c/sub1/status"] = "state: PREPARED\n"
	if _, err := loopSubstream(read, "RcplugLoop"); err == nil || !strings.Contains(err.Error(), "in use") {
		t.Fatalf("err = %v", err)
	}
}

// Without a host the tests never reach one: the audio taps are absent.
func TestMain(m *testing.M) {
	audioPorts = func(context.Context) (bool, []string, error) { return false, nil, errors.New("no host in tests") }
	os.Exit(m.Run())
}

func TestJackSourceUsesTheHostsAudioTaps(t *testing.T) {
	oldPorts, oldTap := audioPorts, audioTap
	t.Cleanup(func() { audioPorts, audioTap = oldPorts, oldTap })
	ports := []string{"system:capture_1", "system:capture_2", "JUCEJack:out_1", "JUCEJack:out_2"}
	on := false
	audioPorts = func(context.Context) (bool, []string, error) { return on, ports, nil }
	var asked []string
	audioTap = func(ctx context.Context, p ...string) (io.ReadCloser, error) {
		asked = p
		return io.NopCloser(strings.NewReader("\x01\x00\x02\x00")), nil
	}
	ctx := context.Background()
	if _, err := newJackSource(ctx, "system:capture_1,system:capture_2", 48000); err != errTapsOff {
		t.Fatalf("taps switched off: %v", err)
	}
	if got := listJackPairs(ctx); got != nil {
		t.Fatalf("pairs while the taps are off: %v", got)
	}
	on = true
	if got := listJackPairs(ctx); len(got) != 2 {
		t.Fatalf("pairs %v", got)
	}
	if _, err := newJackSource(ctx, "system:capture_1,system:capture_2", 44100); err == nil {
		t.Fatal("another rate than the taps' was accepted")
	}
	src, err := newJackSource(ctx, "JUCEJack:out_1,JUCEJack:out_2", 48000)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	if strings.Join(asked, ",") != "JUCEJack:out_1,JUCEJack:out_2" || !strings.Contains(src.describe(), "audio taps") {
		t.Fatalf("asked %v, %s", asked, src.describe())
	}
}
