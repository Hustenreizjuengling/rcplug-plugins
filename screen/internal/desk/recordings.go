package desk

import (
	"strconv"
	"strings"
	"time"
)

// Recordings (RECORDINGS/RECORDING) are the recordings the mixer found on
// the recording storage: one node per file with a UID and a content string
// that carries the name and the file's facts.

// Node types of the recordings.
const (
	RecordingsNode = "RECORDINGS"
	RecordingNode  = "RECORDING"
)

// Recording is one recording as the list shows it.
type Recording struct {
	Node     int
	UID      int64
	Name     string
	Hash     string
	Path     string
	Show     string // the fourth text field (the show the recording belongs to)
	Date     time.Time
	Duration time.Duration
	Multi    bool    // a multitrack recording (else stereo)
	Flags    [2]bool // the two flags after the multitrack one
	Env      [4]float64
	Status   int64 // the cloud upload state the list's icon shows
	Content  string
}

// Recordings lists the recordings, newest first as RØDE's list shows them.
func (m *Model) Recordings() []Recording {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.runs == nil {
		return nil
	}
	var out []Recording
	for i, n := range m.runs.byType[RecordingNode] {
		r := ParseRecording(n.Str("recordingContent"))
		r.Node, r.UID = i, n.Int("recordingUID")
		out = append(out, r)
	}
	for i := 0; i < len(out)/2; i++ {
		out[i], out[len(out)-1-i] = out[len(out)-1-i], out[i]
	}
	return out
}

// RecordingTotals are the count and total length of the recordings.
func (m *Model) RecordingTotals() (count int64, total time.Duration) {
	return m.Int(RecordingsNode, 0, "recordingTotalCount"),
		time.Duration(m.Int(RecordingsNode, 0, "recordingTotalDuration")) * time.Second
}

// DeleteRecording asks the mixer to delete a recording by its UID.
func (m *Model) DeleteRecording(uid int64) { m.SetInt(RecordingsNode, 0, "requestDeleteUID", uid) }

// SaveRecordingEnv writes a recording's trim and fade envelope back into
// its content string, as RØDE's player does when an edit is kept
// (0x81ee00 rebuilds the fourteen fields with the player's envelope).
func (m *Model) SaveRecordingEnv(r Recording, env [4]float64) {
	parts := strings.Split(r.Content, "|")
	if len(parts) < recordingFields {
		return
	}
	for i, v := range env {
		parts[9+i] = strconv.FormatFloat(v, 'f', -1, 64)
	}
	m.SetStr(RecordingNode, r.Node, "recordingContent", strings.Join(parts, "|"))
}

// recordingFields is the number of fields in a content string.
const recordingFields = 14

// ParseRecording reads a recording's content string. RØDE's program keeps
// it as fourteen "|"-separated fields (RcSyncTypes::Recording::Content_t,
// its text form at 0x5b7950, its record at 0x8092b0): name, hash, path, a
// fourth text, the time in milliseconds since the epoch, the length in
// seconds, the multitrack flag and two more flags as 0/1, the four envelope
// values (trim start, fade-in end, fade-out start, trim end) and a status
// number. A shorter string keeps the whole text as the name so a recording
// is never hidden.
func ParseRecording(content string) Recording {
	r := Recording{Content: content, Name: content, Env: [4]float64{0, 0, 1, 1}}
	parts := strings.Split(content, "|")
	if len(parts) < recordingFields {
		if len(parts) > 1 {
			r.Name = parts[0]
		}
		return r
	}
	r.Name, r.Hash, r.Path, r.Show = parts[0], parts[1], parts[2], parts[3]
	if ms, err := strconv.ParseInt(parts[4], 10, 64); err == nil && ms != 0 {
		r.Date = time.UnixMilli(ms)
	}
	if secs, err := strconv.ParseFloat(parts[5], 64); err == nil && secs > 0 {
		r.Duration = time.Duration(secs * float64(time.Second))
	}
	r.Multi = flagField(parts[6])
	r.Flags = [2]bool{flagField(parts[7]), flagField(parts[8])}
	for i := range r.Env {
		if v, err := strconv.ParseFloat(parts[9+i], 64); err == nil {
			r.Env[i] = v
		}
	}
	r.Status, _ = strconv.ParseInt(parts[13], 10, 64)
	return r
}

// flagField reads a 0/1 field the way RØDE's program does (strtol != 0).
func flagField(s string) bool {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return err == nil && n != 0
}
