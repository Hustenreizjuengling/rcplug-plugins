package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// A source provides raw little-endian S16_LE interleaved stereo samples as an
// io.ReadCloser. The main loop reads fixed-size blocks from it. Modelling every
// input as a byte stream keeps ALSA, WAV, stdin and the synthetic generator
// interchangeable.
type source interface {
	io.ReadCloser
	describe() string
}

// openSource builds a source from a spec string:
//
//	alsa:<device>   capture via `arecord -D <device>` (e.g. alsa:default, alsa:hw:0,0)
//	jack:<L>,<R>    two JACK ports, bridged through an ALSA loopback (jack.go);
//	                the way to the desk's audio on the RØDECaster Duo
//	wav:<path>      loop a 48 kHz 16-bit stereo WAV file
//	stdin           read raw S16_LE stereo from standard input
//	test            synthetic sweep (pan + delay) for offline demos
//
// Synthetic and file sources are paced to real time, so the DSP runs at the
// audio rate instead of as fast as the CPU allows.
func openSource(ctx context.Context, spec string, rate int) (source, error) {
	switch {
	case spec == "test" || spec == "":
		return paced(newTestSource(rate), rate), nil
	case spec == "stdin":
		return &readCloser{Reader: os.Stdin, name: "stdin (raw S16_LE stereo)"}, nil
	case strings.HasPrefix(spec, "wav:"):
		src, err := newWavSource(strings.TrimPrefix(spec, "wav:"))
		if err != nil {
			return nil, err
		}
		return paced(src, rate), nil
	case strings.HasPrefix(spec, "alsa:"):
		return newArecordSource(ctx, strings.TrimPrefix(spec, "alsa:"), rate)
	case strings.HasPrefix(spec, "jack:"):
		return newJackSource(ctx, strings.TrimPrefix(spec, "jack:"), rate)
	default:
		return nil, fmt.Errorf("unknown source %q", spec)
	}
}

// alsaName matches ALSA PCM names as listed by `arecord -L` (no shell
// metacharacters; arecord is started without a shell anyway).
var alsaName = regexp.MustCompile(`^[A-Za-z0-9_.-]+(:[A-Za-z0-9_.,=-]+)?$`)

// validateSource checks a source spec received over HTTP. Only the built-in
// test signal, ALSA capture devices, JACK port pairs and WAV files inside the
// plugin's data directory are accepted; stdin is available as a command-line
// flag only.
// It returns the spec to use: a relative WAV path is resolved against the data
// directory, so the file opened is the file that was checked.
func validateSource(spec, dataDir string) (string, error) {
	switch {
	case spec == "test":
		return spec, nil
	case strings.HasPrefix(spec, "alsa:"):
		if !alsaName.MatchString(strings.TrimPrefix(spec, "alsa:")) {
			return "", fmt.Errorf("invalid ALSA device name")
		}
		return spec, nil
	case strings.HasPrefix(spec, "jack:"):
		if _, _, err := parseJackSpec(strings.TrimPrefix(spec, "jack:")); err != nil {
			return "", err
		}
		return spec, nil
	case strings.HasPrefix(spec, "wav:"):
		dir, err := filepath.Abs(dataDir)
		if err != nil {
			return "", err
		}
		p := filepath.Clean(strings.TrimPrefix(spec, "wav:"))
		if !filepath.IsAbs(p) {
			p = filepath.Join(dir, p)
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("WAV files must be inside the plugin's data directory %s", dataDir)
		}
		return "wav:" + p, nil
	}
	return "", fmt.Errorf("unknown source %q (use test, alsa:<device>, jack:<port>,<port> or wav:<file>)", spec)
}

// pacedSource delivers frames no faster than real time.
type pacedSource struct {
	source
	rate   int
	start  time.Time
	frames int64
}

func paced(s source, rate int) source { return &pacedSource{source: s, rate: rate} }

func (p *pacedSource) Read(b []byte) (int, error) {
	if p.start.IsZero() {
		p.start = time.Now()
	}
	n, err := p.source.Read(b)
	p.frames += int64(n / 4)
	due := p.start.Add(time.Duration(p.frames) * time.Second / time.Duration(p.rate))
	if d := time.Until(due); d > 0 {
		time.Sleep(d)
	}
	return n, err
}

type readCloser struct {
	io.Reader
	name   string
	closer func() error
}

func (r *readCloser) Close() error {
	if r.closer != nil {
		return r.closer()
	}
	return nil
}
func (r *readCloser) describe() string { return r.name }

// newArecordSource spawns arecord and streams its raw output. arecord is part
// of alsa-utils and is present on the device; shelling out avoids linking
// libasound into the cross-compiled binary.
func newArecordSource(ctx context.Context, device string, rate int) (source, error) {
	if device == "" {
		device = "default"
	}
	cmd := exec.CommandContext(ctx, "arecord",
		"-D", device, "-f", "S16_LE", "-c", "2", "-r", fmt.Sprint(rate),
		"-t", "raw", "-q", "-")
	stdout, err := cmd.StdoutPipe()
	cmd.Stderr = os.Stderr
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start arecord: %w", err)
	}
	return &readCloser{
		Reader: stdout,
		name:   "alsa:" + device,
		closer: func() error { _ = cmd.Process.Kill(); return cmd.Wait() },
	}, nil
}

// listAlsaCaptureDevices parses `arecord -L` into device ids for the UI.
func listAlsaCaptureDevices() []string {
	out, err := exec.Command("arecord", "-L").Output()
	if err != nil {
		return nil
	}
	var devs []string
	for _, line := range strings.Split(string(out), "\n") {
		if line == "" || strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}
		devs = append(devs, strings.TrimSpace(line))
	}
	return devs
}

// testSource synthesises a 440 Hz tone whose pan sweeps left↔right while a small
// inter-channel delay tracks the pan, so the sonar visibly moves with no
// hardware attached.
type testSource struct {
	rate  int
	phase float64
	t     float64
}

func newTestSource(rate int) *testSource { return &testSource{rate: rate} }

func (t *testSource) describe() string { return "test (synthetic sweep)" }
func (t *testSource) Close() error     { return nil }

func (t *testSource) Read(p []byte) (int, error) {
	frames := len(p) / 4
	const freq = 440.0
	sweepHz := 0.12 // full L→R→L cycle period ≈ 8 s
	for i := 0; i < frames; i++ {
		pan := math.Sin(2 * math.Pi * sweepHz * t.t) // -1..1
		gl := math.Cos((pan + 1) * math.Pi / 4)      // equal-power pan
		gr := math.Sin((pan + 1) * math.Pi / 4)
		s := math.Sin(t.phase)
		// small delay cue coupled to pan (±~300µs)
		dphase := 2 * math.Pi * freq * (pan * 0.0003)
		l := gl * s
		r := gr * math.Sin(t.phase-dphase)
		binary.LittleEndian.PutUint16(p[4*i:], uint16(int16(l*28000)))
		binary.LittleEndian.PutUint16(p[4*i+2:], uint16(int16(r*28000)))
		t.phase += 2 * math.Pi * freq / float64(t.rate)
		if t.phase > 2*math.Pi {
			t.phase -= 2 * math.Pi
		}
		t.t += 1.0 / float64(t.rate)
	}
	return frames * 4, nil
}

// wavSource streams a WAV file and loops it forever. Only 16-bit PCM stereo is
// supported; other formats are rejected with a clear error.
type wavSource struct {
	path string
	data []byte // PCM payload (S16_LE interleaved stereo)
	pos  int
	name string
}

// maxWav bounds a WAV file loaded into memory (~5 min of 48 kHz stereo).
const maxWav = 64 << 20

func newWavSource(path string) (source, error) {
	if fi, err := os.Stat(path); err != nil {
		return nil, err
	} else if fi.Size() > maxWav {
		return nil, fmt.Errorf("WAV file too large (%d MB, max %d MB)", fi.Size()>>20, maxWav>>20)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	pcm, ch, bits, err := parseWav(raw)
	if err != nil {
		return nil, err
	}
	if ch != 2 || bits != 16 {
		return nil, fmt.Errorf("wav must be 16-bit stereo, got %d-bit %d-ch", bits, ch)
	}
	return &wavSource{path: path, data: pcm, name: "wav:" + path}, nil
}

func (w *wavSource) describe() string { return w.name }
func (w *wavSource) Close() error     { return nil }

func (w *wavSource) Read(p []byte) (int, error) {
	if len(w.data) == 0 {
		return 0, io.EOF
	}
	n := 0
	for n < len(p) {
		if w.pos >= len(w.data) {
			w.pos = 0 // loop
		}
		c := copy(p[n:], w.data[w.pos:])
		w.pos += c
		n += c
	}
	return n, nil
}

// parseWav extracts the PCM data chunk and format from a RIFF/WAVE file.
func parseWav(b []byte) (pcm []byte, channels, bits int, err error) {
	if len(b) < 12 || string(b[0:4]) != "RIFF" || string(b[8:12]) != "WAVE" {
		return nil, 0, 0, fmt.Errorf("not a RIFF/WAVE file")
	}
	i := 12
	for i+8 <= len(b) {
		id := string(b[i : i+4])
		sz := int(binary.LittleEndian.Uint32(b[i+4 : i+8]))
		body := i + 8
		if body+sz > len(b) {
			sz = len(b) - body
		}
		switch id {
		case "fmt ":
			if sz >= 16 {
				channels = int(binary.LittleEndian.Uint16(b[body+2 : body+4]))
				bits = int(binary.LittleEndian.Uint16(b[body+14 : body+16]))
			}
		case "data":
			pcm = b[body : body+sz]
		}
		i = body + sz
		if sz%2 == 1 {
			i++ // chunks are word-aligned
		}
	}
	if pcm == nil {
		return nil, 0, 0, fmt.Errorf("no data chunk")
	}
	return pcm, channels, bits, nil
}
