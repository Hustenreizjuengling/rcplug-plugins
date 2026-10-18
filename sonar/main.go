// Command sonar is an rcplug plugin: it captures a channel's stereo audio and
// estimates the direction of the sound (level + stereo/phase), serving a live
// "radar" web UI. It reads S16_LE stereo from ALSA (arecord), from two JACK
// ports (on the RØDECaster Duo, whose desk is a JACK graph; see jack.go), a
// WAV file, stdin or a built-in synthetic generator, so it runs and demos with
// no hardware.
package main

import (
	"context"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"sync"
	"time"

	"github.com/Hustenreizjuengling/rcplug-plugins/sonar/internal/dsp"
	"github.com/Hustenreizjuengling/rcplug/sdk"
)

var version = "dev"

//go:embed web
var webFS embed.FS

func main() {
	addr := flag.String("addr", sdk.ListenAddr("127.0.0.1:8081"), "listen address")
	srcSpec := flag.String("source", envOr("SONAR_SOURCE", "test"),
		"audio source: test | stdin | wav:<path> | alsa:<device> | jack:<port>,<port>")
	rate := flag.Int("rate", 48000, "sample rate (Hz)")
	block := flag.Int("block", 1024, "DSP block size in frames")
	smoothing := flag.Float64("smoothing", 0.6, "EMA smoothing 0..1")
	flip := flag.Bool("flip", false, "flip left/right orientation")
	noiseFloor := flag.Float64("noise-floor", -60, "silence threshold dBFS")
	showVer := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *showVer {
		fmt.Println(version)
		return
	}
	_ = sdk.LowPriority() // no-op when started by rcplugd (already nice 19)

	cfg := dsp.DefaultConfig(*rate)
	cfg.Smoothing = *smoothing
	cfg.FlipSign = *flip
	cfg.NoiseFloor = *noiseFloor
	h := newHub(cfg, *rate, *block, *srcSpec, statePath())
	// A previously saved selection (source, smoothing, …) wins over the flag
	// defaults so the user's choice survives restarts and reboots.
	h.loadState()

	ctx, stop := sdk.SignalContext()
	defer stop()
	go h.run(ctx)
	go h.heartbeat(ctx)

	sub, _ := fs.Sub(webFS, "web")
	mux := http.NewServeMux()
	mux.Handle("GET /", http.FileServer(http.FS(sub)))
	mux.HandleFunc("GET /stream", h.handleStream)
	mux.HandleFunc("GET /api/sources", h.handleSources)
	mux.HandleFunc("GET /api/diagnostics", h.handleDiagnostics)
	mux.HandleFunc("GET /api/config", h.handleConfig)
	mux.HandleFunc("POST /api/config", h.handleConfig)

	log.Printf("sonar %s on %s, source=%s rate=%d", version, *addr, h.srcSpec, *rate)
	if err := sdk.Serve(ctx, *addr, mux); err != nil {
		log.Fatal(err)
	}
}

// hub owns the capture loop, the latest estimate and the SSE subscribers.
type hub struct {
	rate, block int
	statePath   string
	restart     chan struct{} // a source change restarts the capture

	mu      sync.Mutex
	cfg     dsp.Config
	srcSpec string
	last    dsp.Estimate
	srcName string
	srcErr  string
	// channel is the console source chosen in the UI ("" = none); channels
	// maps console sources to the audio source that carries them.
	channel  string
	channels map[string]string
	lastPub  time.Time
	proc     *dsp.Processor
	subs     map[chan []byte]struct{}
}

func newHub(cfg dsp.Config, rate, block int, spec, statePath string) *hub {
	return &hub{cfg: cfg, rate: rate, block: block, srcSpec: spec, statePath: statePath,
		restart: make(chan struct{}, 1), subs: map[chan []byte]struct{}{}, channels: map[string]string{}}
}

// run is the supervised capture loop. It (re)opens the current source and
// feeds the processor. A failing source is retried with growing backoff; a
// source change restarts it immediately.
func (h *hub) run(ctx context.Context) {
	backoff := time.Second
	for ctx.Err() == nil {
		h.mu.Lock()
		cfg, spec := h.cfg, h.srcSpec
		h.mu.Unlock()
		cctx, cancel := context.WithCancel(ctx)
		restarted := make(chan struct{})
		go func() {
			select {
			case <-h.restart:
				close(restarted)
				cancel()
			case <-cctx.Done():
			}
		}()
		began := time.Now()
		err := h.captureOnce(cctx, cfg, spec)
		cancel()
		select {
		case <-restarted:
			backoff = time.Second
			continue // new source chosen
		default:
		}
		if ctx.Err() != nil {
			return
		}
		if err == nil {
			err = fmt.Errorf("source ended")
		}
		if time.Since(began) > time.Minute {
			backoff = time.Second
		}
		h.mu.Lock()
		h.srcErr, h.srcName = err.Error(), spec
		h.mu.Unlock()
		log.Printf("source %s: %v (retrying in %s)", spec, err, backoff)
		h.publish(dsp.Estimate{LevelDb: cfg.NoiseFloor}) // show the error in the UI
		select {
		case <-ctx.Done():
			return
		case <-h.restart:
			backoff = time.Second
		case <-time.After(backoff):
			if backoff < 10*time.Second {
				backoff *= 2
			}
		}
	}
}

// heartbeat keeps the SSE stream warm while a source is broken so the UI shows
// the error instead of silently stalling. During normal capture it stays
// quiet; the capture loop is already streaming at frame rate.
func (h *hub) heartbeat(ctx context.Context) {
	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		h.mu.Lock()
		stale := time.Since(h.lastPub) > time.Second
		last := h.last
		h.mu.Unlock()
		if stale {
			h.publish(last)
		}
	}
}

func (h *hub) captureOnce(ctx context.Context, cfg dsp.Config, spec string) error {
	src, err := openSource(ctx, spec, h.rate)
	if err != nil {
		return err
	}
	defer src.Close()
	stop := context.AfterFunc(ctx, func() { _ = src.Close() }) // unblock a pending read
	defer stop()

	h.mu.Lock()
	h.srcName = src.describe()
	h.srcErr = "connecting to " + src.describe() + " …"
	proc := dsp.NewProcessor(cfg)
	h.proc = proc
	h.mu.Unlock()
	firstBlock := true

	buf := make([]byte, h.block*4)
	s16 := make([]int16, h.block*2)
	emit := time.NewTicker(33 * time.Millisecond) // ~30 fps to the UI
	defer emit.Stop()
	for ctx.Err() == nil {
		if _, err := io.ReadFull(src, buf); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		if firstBlock {
			h.mu.Lock()
			h.srcErr = "" // audio is flowing
			h.mu.Unlock()
			firstBlock = false
		}
		for i := range s16 {
			s16[i] = int16(uint16(buf[2*i]) | uint16(buf[2*i+1])<<8)
		}
		h.mu.Lock() // SetConfig may run concurrently from the HTTP handler
		latest := proc.ProcessInterleaved(s16)
		h.mu.Unlock()
		select {
		case <-emit.C:
			h.publish(latest)
		default:
		}
	}
	return nil
}

func (h *hub) publish(e dsp.Estimate) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.last = e
	h.lastPub = time.Now()
	payload, _ := json.Marshal(struct {
		dsp.Estimate
		Source string `json:"source"`
		Err    string `json:"err,omitempty"`
		Ts     int64  `json:"ts"`
	}{e, h.srcName, h.srcErr, time.Now().UnixMilli()})
	msg := append([]byte("data: "), payload...)
	msg = append(msg, '\n', '\n')
	for ch := range h.subs {
		select {
		case ch <- msg:
		default: // drop if a slow client can't keep up
		}
	}
}

// maxStreams bounds concurrent SSE clients.
const maxStreams = 16

func (h *hub) handleStream(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	ch := make(chan []byte, 8)
	h.mu.Lock()
	if len(h.subs) >= maxStreams {
		h.mu.Unlock()
		sdk.WriteError(w, sdk.NewError(http.StatusServiceUnavailable, "too_many_clients", "too many live connections"))
		return
	}
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		delete(h.subs, ch)
		h.mu.Unlock()
	}()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	fmt.Fprintf(w, "retry: 2000\n\n")
	fl.Flush()
	for {
		select {
		case <-r.Context().Done():
			return
		case msg := <-ch:
			if _, err := w.Write(msg); err != nil {
				return
			}
			fl.Flush()
		}
	}
}

func (h *hub) handleSources(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	cur := h.srcSpec
	h.mu.Unlock()
	sdk.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"current": cur,
		"alsa":    listAlsaCaptureDevices(),
		"jack":    listJackPairs(r.Context()),
		"builtin": []string{"test"},
	})
}

type configPatch struct {
	Source     *string  `json:"source"`
	Smoothing  *float64 `json:"smoothing"`
	Flip       *bool    `json:"flip"`
	NoiseFloor *float64 `json:"noiseFloor"`
	// Channel selects a console source ("combo1", "" = none); its mapped
	// audio source, if any, becomes the current source.
	Channel *string `json:"channel"`
	// Assign maps a console source to an audio source ("" removes it).
	Assign *struct {
		Channel string `json:"channel"`
		Source  string `json:"source"`
	} `json:"assign"`
}

// validChannel accepts console source ids such as "combo1" or "usb2".
func validChannel(s string) bool { return channelID.MatchString(s) }

var channelID = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)

func (h *hub) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var p configPatch
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&p); err != nil {
			sdk.WriteError(w, sdk.NewError(http.StatusBadRequest, "bad_request", err.Error()))
			return
		}
		if p.Source != nil {
			spec, err := validateSource(*p.Source, sdk.DataDir())
			if err != nil {
				sdk.WriteError(w, sdk.NewError(http.StatusBadRequest, "invalid_source", err.Error()))
				return
			}
			p.Source = &spec
		}
		var assignSpec string
		if p.Assign != nil {
			if !validChannel(p.Assign.Channel) {
				sdk.WriteError(w, sdk.NewError(http.StatusBadRequest, "invalid_channel", "unknown channel"))
				return
			}
			if p.Assign.Source != "" {
				spec, err := validateSource(p.Assign.Source, sdk.DataDir())
				if err != nil {
					sdk.WriteError(w, sdk.NewError(http.StatusBadRequest, "invalid_source", err.Error()))
					return
				}
				assignSpec = spec
			}
		}
		if p.Channel != nil && *p.Channel != "" && !validChannel(*p.Channel) {
			sdk.WriteError(w, sdk.NewError(http.StatusBadRequest, "invalid_channel", "unknown channel"))
			return
		}
		h.mu.Lock()
		if p.Assign != nil {
			if assignSpec == "" {
				delete(h.channels, p.Assign.Channel)
			} else {
				h.channels[p.Assign.Channel] = assignSpec
			}
		}
		if p.Channel != nil {
			h.channel = *p.Channel
		}
		// The chosen channel's audio source wins unless a source was given
		// explicitly in the same request.
		if p.Source == nil && (p.Channel != nil || p.Assign != nil) {
			if spec, ok := h.channels[h.channel]; ok && h.channel != "" {
				p.Source = &spec
			}
		}
		if p.Smoothing != nil {
			h.cfg.Smoothing = clamp(*p.Smoothing, 0, 0.99)
		}
		if p.Flip != nil {
			h.cfg.FlipSign = *p.Flip
		}
		if p.NoiseFloor != nil && *p.NoiseFloor < 0 && *p.NoiseFloor >= -120 {
			h.cfg.NoiseFloor = *p.NoiseFloor
		}
		changed := p.Source != nil && *p.Source != h.srcSpec
		if changed {
			h.srcSpec = *p.Source
		} else if h.proc != nil {
			h.proc.SetConfig(h.cfg) // tuning applies live, no capture restart
		}
		h.mu.Unlock()
		if changed {
			select {
			case h.restart <- struct{}{}:
			default:
			}
		}
		h.saveState()
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	channels := map[string]string{}
	for k, v := range h.channels {
		channels[k] = v
	}
	sdk.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"source": h.srcSpec, "smoothing": h.cfg.Smoothing, "flip": h.cfg.FlipSign,
		"noiseFloor": h.cfg.NoiseFloor, "rate": h.rate,
		"channel": h.channel, "channels": channels,
	})
}

func clamp(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

// handleDiagnostics is Sonar's part of the host's diagnostics bundle: the
// settings, the source state, the last estimate and what ALSA reports, which
// is what is needed to find the right capture device on a new device.
func (h *hub) handleDiagnostics(w http.ResponseWriter, r *http.Request) {
	run := func(args ...string) string {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		out, err := exec.CommandContext(ctx, args[0], args[1:]...).CombinedOutput()
		if err != nil {
			return string(out) + "\n(" + err.Error() + ")"
		}
		return string(out)
	}
	h.mu.Lock()
	channels := map[string]string{}
	for k, v := range h.channels {
		channels[k] = v
	}
	doc := map[string]interface{}{
		"version": version, "source": h.srcSpec, "sourceName": h.srcName, "sourceError": h.srcErr,
		"channel": h.channel, "channels": channels, "lastEstimate": h.last, "lastPublished": h.lastPub,
		"config": h.cfg, "rate": h.rate, "block": h.block, "streams": len(h.subs),
	}
	h.mu.Unlock()
	doc["arecordDevices"] = run("arecord", "-l")
	doc["arecordPCMs"] = run("arecord", "-L")
	doc["jackPorts"] = run("jack_lsp", "-c", "-p")
	if b, err := os.ReadFile("/proc/asound/cards"); err == nil {
		doc["asoundCards"] = string(b)
	}
	sdk.WriteJSON(w, http.StatusOK, doc)
}
