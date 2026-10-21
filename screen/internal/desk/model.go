package desk

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/Hustenreizjuengling/rcplug/sdk/console"
)

// Model is the live state of the mixer for the UI. It loads the tree from
// the console, follows the stream of changes, and writes single properties
// back optimistically (the mixer does not echo a client's own writes, so the
// local copy is updated at once and the console is told in the background).
//
// A Model is safe for concurrent use: the drawing goroutine reads it while
// the stream goroutine updates it.
type Model struct {
	conn *Conn
	logf func(string, ...interface{})

	mu            sync.RWMutex
	root          *Node
	runs          *runs
	epoch         uint64
	seq           uint64
	online        bool
	stale         bool
	control       string // off | basic | expert
	conState      string // the console's link to the desk
	info          Info
	faders        []Name
	sources       []Name
	outputs       []Name
	unity         int64
	physical      int
	perBank       int
	perSource     int // outputs per source in the send matrix
	mics          int // microphone inputs
	headphones    int // headphone outputs
	presetsLogged bool
	banks         int    // pad banks
	offline       bool   // no console: writes stay local (the preview)
	reloading     bool   // a resync reload is in flight (single-flight)
	rev           uint64 // bumps on every change, so the UI can tell something moved

	wmu     sync.Mutex
	pending map[string]writeOp // coalesced writes, flushed by runWriter

	enc *encoderState // the main knob's turns and presses
}

// Info is what the model knows about the device.
type Info struct {
	Model     string
	ModelName string
	Firmware  string
}

// New makes a model over the console behind the host (hostURL "" uses the
// host the plugin runs under).
func New(hostURL string, logf func(string, ...interface{})) *Model {
	if logf == nil {
		logf = func(string, ...interface{}) {}
	}
	return &Model{conn: Dial(hostURL), logf: logf, control: "off", unity: 90, physical: 4, perBank: 6, enc: newEncoderState()}
}

// NewOffline makes a model with no console: it is filled with LoadRaw and
// its writes only change the local copy (for the browser preview and tests).
func NewOffline(logf func(string, ...interface{})) *Model {
	m := New("", logf)
	m.offline = true
	m.control = "expert"
	return m
}

// Conn is the underlying console connection (for triggers and presses).
func (m *Model) Conn() *Conn { return m.conn }

// Run keeps the model current until ctx ends: it loads the tree, follows the
// change stream, and reloads when the stream says the state jumped or breaks.
func (m *Model) Run(ctx context.Context) {
	go m.runWriter(ctx)
	backoff := time.Second
	for ctx.Err() == nil {
		if err := m.load(ctx); err != nil {
			m.setOffline(err)
			if !sleep(ctx, backoff) {
				return
			}
			backoff = min2(backoff*2, 10*time.Second)
			continue
		}
		backoff = time.Second
		// Follow the stream; it returns when it breaks or ctx ends, then we
		// reload (a resync, or a reconnect after an error).
		err := m.conn.Client.Events(ctx, m.onEvent)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			m.logf("screen: desk stream: %v", err)
		}
		if !sleep(ctx, 500*time.Millisecond) {
			return
		}
	}
}

// load fetches the whole tree and replaces the model.
func (m *Model) load(ctx context.Context) error {
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	raw, err := m.conn.Client.Tree(cctx)
	if err != nil {
		return err
	}
	return m.adopt(raw)
}

// LoadRaw adopts a tree document directly (for the offline preview and
// tests), without a console.
func (m *Model) LoadRaw(raw []byte) error { return m.adopt(raw) }

// adopt parses a tree document and replaces the model with it.
func (m *Model) adopt(raw []byte) error {
	root, td, err := parseTree(raw)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.root = root
	m.runs = indexRuns(root)
	m.epoch, m.seq, m.stale = td.Epoch, td.Seq, td.Stale
	m.online = true
	m.info = Info{Model: td.Meta.Layout.Model, ModelName: td.Meta.Layout.ModelName, Firmware: td.Meta.Layout.Firmware}
	if len(td.Meta.Faders) > 0 {
		m.faders = td.Meta.Faders
	}
	if len(td.Meta.Sources) > 0 {
		m.sources = td.Meta.Sources
	}
	if len(td.Meta.Outputs) > 0 {
		m.outputs = td.Meta.Outputs
	}
	if td.Meta.UnityFader > 0 {
		m.unity = td.Meta.UnityFader
	}
	if td.Meta.Layout.PhysicalFaders > 0 {
		m.physical = td.Meta.Layout.PhysicalFaders
	}
	if td.Meta.Layout.PadsPerBank > 0 {
		m.perBank = td.Meta.Layout.PadsPerBank
	}
	if td.Meta.Layout.MixPerSource > 0 {
		m.perSource = td.Meta.Layout.MixPerSource
	}
	if td.Meta.Layout.MicInputs > 0 {
		m.mics = td.Meta.Layout.MicInputs
	}
	if td.Meta.Layout.Headphones > 0 {
		m.headphones = td.Meta.Layout.Headphones
	}
	if td.Meta.Layout.PadBanks > 0 {
		m.banks = td.Meta.Layout.PadBanks
	}
	m.rev++
	m.mu.Unlock()
	m.logf("screen: desk loaded (%s %s, %d strips, seq %d)", m.info.ModelName, m.info.Firmware, len(m.faders), td.Seq)
	m.logPresetsOnce()
	return nil
}

func (m *Model) setOffline(err error) {
	m.mu.Lock()
	first := m.online
	m.online = false
	m.rev++
	m.mu.Unlock()
	if first {
		m.logf("screen: desk offline: %v", err)
	}
}

// onEvent handles one server-sent event from the console.
func (m *Model) onEvent(e console.Event) {
	switch e.Name {
	case "changes":
		m.applyChanges(e.Data)
	case "resync":
		// The console's copy jumped; reload at the next turn of Run by
		// breaking the stream is not possible here, so reload inline.
		go m.reloadSoon()
	case "status", "control", "hello":
		m.applyStatus(e.Data)
	}
}

// changesDoc is {"epoch":E,"seq":S,"ch":[[[path],name,value,"code"],…]}.
type changesDoc struct {
	Epoch uint64            `json:"epoch"`
	Seq   uint64            `json:"seq"`
	Ch    []json.RawMessage `json:"ch"`
}

func (m *Model) applyChanges(data []byte) {
	var doc changesDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.root == nil {
		return
	}
	m.epoch, m.seq = doc.Epoch, doc.Seq
	for _, raw := range doc.Ch {
		path, name, val, ok := parseChange(raw)
		if !ok {
			continue
		}
		if n := m.root.At(path); n != nil {
			n.Props[name] = val
			if name == "encoderSignal" || name == "encoderPressed" {
				m.onEncoderProp(name, val)
			}
		}
	}
	m.rev++
}

// parseChange reads one [[path],name,value,"code"] entry.
func parseChange(raw json.RawMessage) ([]int, string, Value, bool) {
	var tuple []json.RawMessage
	if err := json.Unmarshal(raw, &tuple); err != nil || len(tuple) < 3 {
		return nil, "", Value{}, false
	}
	var path []int
	if err := json.Unmarshal(tuple[0], &path); err != nil {
		return nil, "", Value{}, false
	}
	var name string
	if err := json.Unmarshal(tuple[1], &name); err != nil {
		return nil, "", Value{}, false
	}
	var code byte
	if len(tuple) >= 4 {
		var cs string
		if json.Unmarshal(tuple[3], &cs) == nil && len(cs) == 1 {
			code = cs[0]
		}
	}
	var x interface{}
	dec := json.NewDecoder(bytes.NewReader(tuple[2]))
	dec.UseNumber()
	_ = dec.Decode(&x)
	return path, name, fromJSON(x, code), true
}

// statusDoc is the part of a status/hello event the model keeps.
type statusDoc struct {
	Epoch      uint64 `json:"epoch"`
	Seq        uint64 `json:"seq"`
	Stale      bool   `json:"stale"`
	Control    string `json:"control"`
	Connection struct {
		State string `json:"state"`
	} `json:"connection"`
	ModelName string `json:"modelName"`
	Firmware  string `json:"firmware"`
}

func (m *Model) applyStatus(data []byte) {
	var doc statusDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return
	}
	m.mu.Lock()
	if doc.Control != "" {
		m.control = doc.Control
	}
	if doc.Connection.State != "" {
		m.conState = doc.Connection.State
	}
	if doc.ModelName != "" {
		m.info.ModelName = doc.ModelName
	}
	if doc.Firmware != "" {
		m.info.Firmware = doc.Firmware
	}
	// A hello whose seq is ahead of ours means we missed changes; reload.
	behind := doc.Seq > m.seq+1 && m.root != nil
	m.rev++
	m.mu.Unlock()
	if behind {
		go m.reloadSoon()
	}
}

// reloadSoon reloads the tree once, out of band (after a resync). Only one
// reload runs at a time, so a flapping console does not pile up fetches.
func (m *Model) reloadSoon() {
	m.mu.Lock()
	if m.reloading {
		m.mu.Unlock()
		return
	}
	m.reloading = true
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		m.reloading = false
		m.mu.Unlock()
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := m.load(ctx); err != nil {
		m.logf("screen: desk resync: %v", err)
	}
}

func sleep(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

func min2(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

// logPresetsOnce notes the processing presets once: their names, and the
// contents of any the package does not understand (the format, see
// presets.go, was read off a device's diagnostics).
func (m *Model) logPresetsOnce() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.presetsLogged || m.runs == nil {
		return
	}
	ps := m.runs.byType["FXPRESET"]
	if len(ps) == 0 {
		return
	}
	m.presetsLogged = true
	var names []string
	for _, n := range ps {
		p, ok := ParsePreset(n.Str("fxPresetContents"))
		if !ok || !p.Complete() {
			m.logf("screen: processing preset %d not understood: %.80s", n.Int("fxPresetIdx"), n.Str("fxPresetContents"))
			continue
		}
		names = append(names, p.Name)
	}
	m.logf("screen: %d processing presets: %s", len(ps), strings.Join(names, ", "))
}
