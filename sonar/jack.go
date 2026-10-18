package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// JACK capture, opt-in. On the RØDECaster Duo the whole desk is a JACK graph
// (jackd 1.9, JACK2): jackd holds the card's only capture device (hw:0,3), so
// arecord cannot open it, and every input and output of the desk is a JACK
// port (system:capture_*, JUCEJack:out_*, …). A jack source bridges two of
// those ports into Sonar with tools the firmware ships:
//
//  1. load snd-aloop as card RcplugLoop (index 7, two substreams), an ALSA
//     loopback, unless one is there already, and pick a substream nobody
//     uses on either end (a desktop's sound server may hold one);
//  2. run alsa_out as JACK client "rcplug-sonar", playing into it;
//  3. jack_connect the two chosen ports to its inputs;
//  4. record the other end of the loopback with arecord, like an ALSA source.
//
// Stopping the source ends alsa_out, which removes its connections. The
// module stays loaded. This adds a client to RØDE's audio graph, so the UI
// asks before choosing a jack source; it is never the default.

const (
	jackClient = "rcplug-sonar"
	// loopCard is the id snd-aloop gets from Sonar. Index 7 keeps it clear of
	// the desk's card 0 and of cards that USB devices would get.
	loopCard  = "RcplugLoop"
	loopIndex = "7"
)

// jackPort matches JACK port names as RØDE's graph has them, client:port.
var jackPort = regexp.MustCompile(`^[A-Za-z0-9_.-]+:[A-Za-z0-9_.-]+$`)

// parseJackSpec splits the part after "jack:" into the left and right port.
func parseJackSpec(spec string) (left, right string, err error) {
	l, r, ok := strings.Cut(spec, ",")
	if !ok || !jackPort.MatchString(l) || !jackPort.MatchString(r) || l == r {
		return "", "", fmt.Errorf("a JACK source names two different ports, jack:<client:port>,<client:port>")
	}
	return l, r, nil
}

// parseJackOutputs reads `jack_lsp -p` and returns the output ports, the ones
// that carry a signal Sonar can listen to. Each port line is followed by an
// indented "properties: output,physical,…" line.
func parseJackOutputs(out string) []string {
	var ports []string
	cur := ""
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		if line[0] != ' ' && line[0] != '\t' {
			cur = strings.TrimSpace(line)
			continue
		}
		props, ok := strings.CutPrefix(strings.TrimSpace(line), "properties:")
		if !ok || cur == "" || !jackPort.MatchString(cur) {
			continue
		}
		for _, p := range strings.Split(props, ",") {
			if strings.TrimSpace(p) == "output" {
				ports = append(ports, cur)
				break
			}
		}
	}
	return ports
}

// jackPairs offers stereo pairs for the UI: consecutive output ports of the
// same client, 1+2, 3+4, … in natural order. Sonar's own bridge is left out.
func jackPairs(ports []string) []string {
	byClient := map[string][]string{}
	var clients []string
	for _, p := range ports {
		c := p[:strings.IndexByte(p, ':')]
		if c == jackClient {
			continue
		}
		if _, seen := byClient[c]; !seen {
			clients = append(clients, c)
		}
		byClient[c] = append(byClient[c], p)
	}
	var pairs []string
	for _, c := range clients {
		ps := byClient[c]
		sort.SliceStable(ps, func(i, j int) bool { return naturalLess(ps[i], ps[j]) })
		for i := 0; i+1 < len(ps); i += 2 {
			pairs = append(pairs, ps[i]+","+ps[i+1])
		}
	}
	return pairs
}

// naturalLess orders "capture_2" before "capture_10".
func naturalLess(a, b string) bool {
	ta, na := splitNum(a)
	tb, nb := splitNum(b)
	if ta != tb {
		return ta < tb
	}
	return na < nb
}

func splitNum(s string) (string, int) {
	i := len(s)
	for i > 0 && s[i-1] >= '0' && s[i-1] <= '9' {
		i--
	}
	n, _ := strconv.Atoi(s[i:])
	return s[:i], n
}

// loopbackCard picks the snd-aloop card to use from /proc/asound/cards:
// Sonar's own, else one that is loaded already. "" if there is none.
func loopbackCard(cards string) string {
	other := ""
	for _, line := range strings.Split(cards, "\n") {
		// " 7 [RcplugLoop     ]: Loopback - Loopback"
		open, close := strings.IndexByte(line, '['), strings.IndexByte(line, ']')
		if open < 0 || close < open {
			continue
		}
		id := strings.TrimSpace(line[open+1 : close])
		if id == loopCard {
			return id
		}
		if other == "" && strings.Contains(line[close:], "Loopback") {
			other = id
		}
	}
	return other
}

// listJackPairs returns the stereo pairs JACK offers now, or nil when JACK is
// not running or its tools are missing (any computer but the device).
func listJackPairs(ctx context.Context) []string {
	ports, err := jackOutputs(ctx)
	if err != nil {
		return nil
	}
	return jackPairs(ports)
}

func jackOutputs(ctx context.Context) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "jack_lsp", "-p").Output()
	if err != nil {
		return nil, fmt.Errorf("JACK is not reachable (jack_lsp: %v)", err)
	}
	return parseJackOutputs(string(out)), nil
}

// ensureLoopback returns the loopback card's id, loading snd-aloop first if
// no loopback card exists.
func ensureLoopback(ctx context.Context) (string, error) {
	read := func() string {
		b, _ := os.ReadFile("/proc/asound/cards")
		return string(b)
	}
	if id := loopbackCard(read()); id != "" {
		return id, nil
	}
	log.Printf("jack source: loading snd-aloop as card %s (index %s)", loopCard, loopIndex)
	out, err := exec.CommandContext(ctx, "modprobe", "snd-aloop",
		"index="+loopIndex, "id="+loopCard, "pcm_substreams=2").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("loading snd-aloop failed: %v %s", err, strings.TrimSpace(string(out)))
	}
	for i := 0; i < 20; i++ {
		if id := loopbackCard(read()); id != "" {
			return id, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return "", errors.New("snd-aloop is loaded, but no loopback card appeared in /proc/asound/cards")
}

// loopSubstream returns a substream of the loopback card whose two ends Sonar
// uses are both closed: snd-aloop pairs playback on device 0 substream N with
// capture on device 1 substream N. read reads a /proc file.
func loopSubstream(read func(string) ([]byte, error), card string) (int, error) {
	for n := 0; n < 8; n++ {
		play, err1 := read(fmt.Sprintf("/proc/asound/%s/pcm0p/sub%d/status", card, n))
		capt, err2 := read(fmt.Sprintf("/proc/asound/%s/pcm1c/sub%d/status", card, n))
		if err1 != nil || err2 != nil {
			break // no such substream
		}
		if strings.TrimSpace(string(play)) == "closed" && strings.TrimSpace(string(capt)) == "closed" {
			return n, nil
		}
	}
	return 0, fmt.Errorf("every substream of the loopback card %s is in use (another sound server may hold it)", card)
}

// newJackSource bridges two JACK ports into Sonar (see the top of this file).
func newJackSource(ctx context.Context, spec string, rate int) (source, error) {
	left, right, err := parseJackSpec(spec)
	if err != nil {
		return nil, err
	}
	for _, tool := range []string{"jack_lsp", "jack_connect", "alsa_out", "arecord", "modprobe"} {
		if _, err := exec.LookPath(tool); err != nil {
			return nil, fmt.Errorf("the JACK source needs %s, which is not installed here", tool)
		}
	}
	ports, err := jackOutputs(ctx)
	if err != nil {
		return nil, err
	}
	for _, p := range []string{left, right} {
		if !contains(ports, p) {
			return nil, fmt.Errorf("JACK has no output port %s", p)
		}
	}
	card, err := ensureLoopback(ctx)
	if err != nil {
		return nil, err
	}
	sub, err := loopSubstream(os.ReadFile, card)
	if err != nil {
		return nil, err
	}

	bridge := exec.CommandContext(ctx, "alsa_out", "-j", jackClient, "-d", fmt.Sprintf("hw:%s,0,%d", card, sub),
		"-c", "2", "-r", strconv.Itoa(rate))
	bridge.Stdout, bridge.Stderr = os.Stderr, os.Stderr
	if err := bridge.Start(); err != nil {
		return nil, fmt.Errorf("start alsa_out: %w", err)
	}
	exited := make(chan error, 1)
	go func() { exited <- bridge.Wait() }()
	stopBridge := func() {
		_ = bridge.Process.Kill()
		<-exited
		exited <- nil // later calls return at once
	}
	if err := waitJackPorts(ctx, exited, jackClient+":playback_1", jackClient+":playback_2"); err != nil {
		stopBridge()
		return nil, err
	}
	for i, p := range []string{left, right} {
		dst := fmt.Sprintf("%s:playback_%d", jackClient, i+1)
		if out, err := exec.CommandContext(ctx, "jack_connect", p, dst).CombinedOutput(); err != nil {
			stopBridge()
			return nil, fmt.Errorf("jack_connect %s %s: %v %s", p, dst, err, strings.TrimSpace(string(out)))
		}
	}
	log.Printf("jack source: %s + %s → %s → hw:%s,0,%d", left, right, jackClient, card, sub)

	// plughw: the loopback runs at whatever format alsa_out chose.
	rec, err := newArecordSource(ctx, fmt.Sprintf("plughw:%s,1,%d", card, sub), rate)
	if err != nil {
		stopBridge()
		return nil, err
	}
	inner := rec
	return &readCloser{
		Reader: inner,
		name:   "jack:" + left + " + " + right,
		closer: func() error {
			err := inner.Close()
			stopBridge()
			return err
		},
	}, nil
}

// waitJackPorts waits up to 5 s for alsa_out to register its ports, and
// stops early if alsa_out exits (its reason is in the plugin log).
func waitJackPorts(ctx context.Context, exited chan error, want ...string) error {
	deadline := time.Now().Add(5 * time.Second)
	for {
		select {
		case err := <-exited:
			exited <- err
			return fmt.Errorf("alsa_out exited (%v) before it registered in JACK; see the Sonar log", err)
		default:
		}
		out, err := exec.CommandContext(ctx, "jack_lsp").Output()
		if err == nil {
			have := strings.Fields(string(out))
			ok := true
			for _, w := range want {
				ok = ok && contains(have, w)
			}
			if ok {
				return nil
			}
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			return fmt.Errorf("alsa_out did not register %s in JACK", strings.Join(want, ", "))
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
