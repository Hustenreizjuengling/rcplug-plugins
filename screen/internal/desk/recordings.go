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
	Path     string
	Date     time.Time
	Duration time.Duration
	Multi    bool // a multitrack recording (else stereo)
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

// ParseRecording reads a recording's content string: fields separated by
// "|" (the mixer's list format, as the show and mix properties use it):
// name, path, date (seconds since the epoch), length (seconds), and a
// multitrack flag. Fields that are missing stay empty; unknown layouts keep
// the whole string as the name so a recording is never hidden.
func ParseRecording(content string) Recording {
	r := Recording{Content: content, Name: content}
	parts := strings.Split(content, "|")
	if len(parts) < 2 {
		return r
	}
	r.Name = parts[0]
	for _, p := range parts[1:] {
		switch {
		case strings.HasPrefix(p, "/"):
			r.Path = p
			if r.Name == "" {
				r.Name = strings.TrimSuffix(p[strings.LastIndex(p, "/")+1:], ".wav")
			}
		case p == "multi" || p == "multitrack":
			r.Multi = true
		case p == "stereo":
			r.Multi = false
		default:
			n, err := strconv.ParseFloat(p, 64)
			if err != nil {
				continue
			}
			switch {
			case n > 1e9 && r.Date.IsZero():
				r.Date = time.Unix(int64(n), 0)
			case r.Duration == 0 && n > 0:
				r.Duration = time.Duration(n * float64(time.Second))
			case n == 1 && !r.Multi:
				r.Multi = true
			}
		}
	}
	return r
}
