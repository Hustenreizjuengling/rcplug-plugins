package pages

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// wavWithCues builds a minimal WAV stream: fmt, a data chunk and a cue
// chunk with the given sample offsets (in any order).
func wavWithCues(rate uint32, offsets []uint32) []byte {
	var body bytes.Buffer
	chunk := func(id string, data []byte) {
		body.WriteString(id)
		binary.Write(&body, binary.LittleEndian, uint32(len(data)))
		body.Write(data)
		if len(data)%2 == 1 {
			body.WriteByte(0)
		}
	}
	var fmtc bytes.Buffer
	binary.Write(&fmtc, binary.LittleEndian, uint16(1))
	binary.Write(&fmtc, binary.LittleEndian, uint16(2))
	binary.Write(&fmtc, binary.LittleEndian, rate)
	binary.Write(&fmtc, binary.LittleEndian, rate*4)
	binary.Write(&fmtc, binary.LittleEndian, uint16(4))
	binary.Write(&fmtc, binary.LittleEndian, uint16(16))
	chunk("fmt ", fmtc.Bytes())
	chunk("data", []byte{1, 2, 3})
	var cue bytes.Buffer
	binary.Write(&cue, binary.LittleEndian, uint32(len(offsets)))
	for i, off := range offsets {
		binary.Write(&cue, binary.LittleEndian, uint32(i+1))
		binary.Write(&cue, binary.LittleEndian, uint32(0))
		cue.WriteString("data")
		binary.Write(&cue, binary.LittleEndian, uint32(0))
		binary.Write(&cue, binary.LittleEndian, uint32(0))
		binary.Write(&cue, binary.LittleEndian, off)
	}
	chunk("cue ", cue.Bytes())
	var out bytes.Buffer
	out.WriteString("RIFF")
	binary.Write(&out, binary.LittleEndian, uint32(4+body.Len()))
	out.WriteString("WAVE")
	out.Write(body.Bytes())
	return out.Bytes()
}

// TestWavCues reads the cue chunk's sample offsets and the rate, and walks
// the markers before and after a position.
func TestWavCues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "take.wav")
	if err := os.WriteFile(path, wavWithCues(48000, []uint32{480000, 96000, 2400000}), 0o644); err != nil {
		t.Fatal(err)
	}
	m := loadCueMarkers(path)
	if m.rate != 48000 || len(m.cues) != 3 || m.cues[0] != 96000 || m.cues[2] != 2400000 {
		t.Fatalf("markers %+v", m)
	}
	if next, ok := m.after(3 * time.Second); !ok || next != 10*time.Second {
		t.Fatalf("after 3 s: %v %v, want 10 s", next, ok)
	}
	if prev, ok := m.before(10*time.Second + 500*time.Millisecond); !ok || prev != 2*time.Second {
		t.Fatalf("before 10.5 s: %v %v, want 2 s (a marker within a second does not count)", prev, ok)
	}
	if _, ok := m.before(time.Second); ok {
		t.Fatal("no marker before 1 s")
	}
	if _, ok := m.after(time.Minute); ok {
		t.Fatal("no marker after the last")
	}
	if none := loadCueMarkers(filepath.Join(t.TempDir(), "missing.wav")); none.rate != 0 || none.cues != nil {
		t.Fatalf("a missing file gives no markers: %+v", none)
	}
	if _, ok := none().after(0); ok {
		t.Fatal("no markers without a file")
	}
}

func none() cueMarkers { return cueMarkers{} }
