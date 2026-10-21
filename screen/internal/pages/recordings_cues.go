package pages

import (
	"encoding/binary"
	"io"
	"os"
	"sort"
	"time"
)

// The markers of a recording are not in the desk's tree: RØDE's player
// (PodcastPlayerPage::CueMarkersDisplay) reads them from the WAV file's cue
// chunk through juce::WavAudioFormat (the program carries the metadata keys
// NumCuePoints, CueLabel…), and its flag buttons jump to the previous and
// the next one. The plugin runs on the desk and reads the same file.

// wavCues reads a WAV file's sample rate and its cue points (sample
// offsets, ascending). A file without a cue chunk gives none; a file the
// plugin cannot read gives rate 0.
func wavCues(path string) (rate int64, cues []int64) {
	f, err := os.Open(path)
	if err != nil {
		return 0, nil
	}
	defer f.Close()
	return readWavCues(f)
}

// readWavCues walks the RIFF chunks of a WAV stream for "fmt " and "cue ".
func readWavCues(r io.ReadSeeker) (rate int64, cues []int64) {
	var head [12]byte
	if _, err := io.ReadFull(r, head[:]); err != nil || string(head[:4]) != "RIFF" || string(head[8:]) != "WAVE" {
		return 0, nil
	}
	for {
		var ch [8]byte
		if _, err := io.ReadFull(r, ch[:]); err != nil {
			break
		}
		id, size := string(ch[:4]), int64(binary.LittleEndian.Uint32(ch[4:]))
		switch id {
		case "fmt ":
			var fmtc [16]byte
			if size < 16 {
				return rate, cues
			}
			if _, err := io.ReadFull(r, fmtc[:]); err != nil {
				return rate, cues
			}
			rate = int64(binary.LittleEndian.Uint32(fmtc[4:8]))
			if _, err := r.Seek(size-16+size%2, io.SeekCurrent); err != nil {
				return rate, cues
			}
		case "cue ":
			var n [4]byte
			if size < 4 {
				return rate, cues
			}
			if _, err := io.ReadFull(r, n[:]); err != nil {
				return rate, cues
			}
			count := int64(binary.LittleEndian.Uint32(n[:]))
			if count*24 > size-4 || count > 100000 {
				return rate, cues
			}
			for i := int64(0); i < count; i++ {
				var cp [24]byte
				if _, err := io.ReadFull(r, cp[:]); err != nil {
					return rate, cues
				}
				// dwName, dwPosition, fccChunk, dwChunkStart, dwBlockStart,
				// dwSampleOffset: the offset is the sample position.
				cues = append(cues, int64(binary.LittleEndian.Uint32(cp[20:24])))
			}
			if _, err := r.Seek(size-4-count*24+size%2, io.SeekCurrent); err != nil {
				return rate, cues
			}
		default:
			if _, err := r.Seek(size+size%2, io.SeekCurrent); err != nil {
				return rate, cues
			}
		}
		if rate != 0 && cues != nil {
			break
		}
	}
	sort.Slice(cues, func(i, j int) bool { return cues[i] < cues[j] })
	return rate, cues
}

// cueMarkers is a recording's markers as positions in time.
type cueMarkers struct {
	rate int64
	cues []int64
}

// loadCueMarkers reads a recording's markers (none when the file cannot be
// read).
func loadCueMarkers(path string) cueMarkers {
	if path == "" {
		return cueMarkers{}
	}
	rate, cues := wavCues(path)
	return cueMarkers{rate: rate, cues: cues}
}

// before is the marker before pos (more than a second back, so repeated
// presses walk the markers), ok false when there is none.
func (c cueMarkers) before(pos time.Duration) (time.Duration, bool) {
	if c.rate <= 0 {
		return 0, false
	}
	limit := pos - time.Second
	for i := len(c.cues) - 1; i >= 0; i-- {
		if t := c.at(i); t <= limit {
			return t, true
		}
	}
	return 0, false
}

// after is the first marker after pos, ok false when there is none.
func (c cueMarkers) after(pos time.Duration) (time.Duration, bool) {
	if c.rate <= 0 {
		return 0, false
	}
	for i := range c.cues {
		if t := c.at(i); t > pos {
			return t, true
		}
	}
	return 0, false
}

func (c cueMarkers) at(i int) time.Duration {
	return time.Duration(float64(c.cues[i]) / float64(c.rate) * float64(time.Second))
}
