// Package rode reads the pictures and texts that RØDE's screen program
// carries in its program file on the device (/usr/bin/rc_gui_juce): its
// icons (SVG and PNG) and its texts in every language. rcplug ships none of
// them; the screen uses what the device already has.
//
// The program keeps them as JUCE "binary data": a table of pointers to the
// resource names (such as "IconHome_svg") and next to it a table of
// pointers to the data. In a position-independent program both tables are
// filled in by relative relocations, so they are found by reading the
// relocations: the longest run of pointers to such names, and the run of
// pointers to data that lines up with it (the data's kind — SVG, PNG or
// text — must match the names). Nothing depends on fixed addresses.
package rode

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
)

// GUIProgram is where the program lies on the device.
const GUIProgram = "/usr/bin/rc_gui_juce"

// Resources are the resources found in the program.
type Resources struct {
	mem    memory
	items  map[string]item // by name without the suffix: "IconHome"
	closer io.Closer
}

// Close lets go of the program file.
func (r *Resources) Close() error {
	if r.closer == nil {
		return nil
	}
	return r.closer.Close()
}

type item struct {
	addr uint64
	kind string // svg, png, txt (from the data, not the name)
}

// memory reads the program's memory image.
type memory interface {
	// read returns up to n bytes at address a (fewer at the end).
	read(a uint64, n int) []byte
}

// Open reads the resource tables of the program at path.
func Open(path string) (*Resources, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	m, relocs, err := loadELF(f)
	if err != nil {
		f.Close()
		return nil, err
	}
	r, err := find(m, relocs)
	if err != nil {
		f.Close()
		return nil, err
	}
	r.closer = f
	return r, nil
}

// Names lists the resources (without suffix), sorted.
func (r *Resources) Names() []string {
	out := make([]string, 0, len(r.items))
	for n := range r.items {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// Kind is "svg", "png" or "txt" ("" if there is no such resource).
func (r *Resources) Kind(name string) string { return r.items[name].kind }

// maxSize bounds one resource.
const maxSize = 8 << 20

// Bytes is the resource's data.
func (r *Resources) Bytes(name string) ([]byte, bool) {
	it, ok := r.items[name]
	if !ok {
		return nil, false
	}
	return readResource(r.mem, it.addr, it.kind)
}

func readResource(m memory, a uint64, kind string) ([]byte, bool) {
	var end []byte
	switch kind {
	case "png":
		end = []byte("IEND\xaeB`\x82")
	case "svg":
		end = []byte("</svg>")
	default:
		end = []byte{0}
	}
	var buf []byte
	for step := 64 << 10; len(buf) < maxSize; step *= 2 {
		buf = m.read(a, step)
		if i := bytes.Index(buf, end); i >= 0 {
			if kind == "txt" {
				return buf[:i], true
			}
			return buf[:i+len(end)], true
		}
		if len(buf) < step {
			return nil, false
		}
	}
	return nil, false
}

// kindAt tells what data starts at a.
func kindAt(m memory, a uint64) string {
	b := m.read(a, 16)
	switch {
	case bytes.HasPrefix(b, []byte("\x89PNG\r\n\x1a\n")):
		return "png"
	case bytes.HasPrefix(b, []byte("<svg")), bytes.HasPrefix(b, []byte("<?xml")):
		return "svg"
	case bytes.HasPrefix(b, []byte("language:")):
		return "txt"
	}
	return ""
}

var resourceName = regexp.MustCompile(`^[A-Za-z0-9_]+_(svg|png|txt)$`)

// cstring reads a NUL-terminated name of at most 128 bytes.
func cstring(m memory, a uint64) string {
	b := m.read(a, 129)
	i := bytes.IndexByte(b, 0)
	if i < 0 {
		return ""
	}
	return string(b[:i])
}

// reloc is one relative relocation: the pointer at slot gets value.
type reloc struct{ slot, value uint64 }

// find locates the two tables and pairs names with data.
func find(m memory, relocs []reloc) (*Resources, error) {
	sort.Slice(relocs, func(i, j int) bool { return relocs[i].slot < relocs[j].slot })
	at := make(map[uint64]uint64, len(relocs))
	for _, r := range relocs {
		at[r.slot] = r.value
	}
	// The longest run of consecutive pointers to resource names.
	var names []string
	var nameStart uint64
	for i := 0; i < len(relocs); {
		j := i
		var run []string
		for j < len(relocs) && (j == i || relocs[j].slot == relocs[j-1].slot+8) {
			n := cstring(m, relocs[j].value)
			if !resourceName.MatchString(n) {
				break
			}
			run = append(run, n)
			j++
		}
		if len(run) > len(names) {
			names, nameStart = run, relocs[i].slot
		}
		i = max(j, i+1)
	}
	if len(names) < 10 {
		return nil, errors.New("rode: no resource name table found")
	}
	// The data table: a run of pointers to resources whose kinds match the
	// names, after the name table. A few names at the start may have no data
	// in it (RØDE's 1.7.6 begins with three such, "_1_svg" to "_3_svg") and
	// a few other pointers may come first: both are tried, smallest first.
	nameEnd := nameStart + 8*uint64(len(names))
	try := func(skipNames int, start uint64) map[string]item {
		items := map[string]item{}
		for i, n := range names[skipNames:] {
			v, ok := at[start+8*uint64(i)]
			if !ok {
				return nil
			}
			ext := n[strings.LastIndexByte(n, '_')+1:]
			k := kindAt(m, v)
			// Some "png" resources hold SVG data; anything else is a mismatch.
			if k == "" || k != ext && !(ext == "png" && k == "svg") {
				return nil
			}
			items[strings.TrimSuffix(n, "_"+ext)] = item{addr: v, kind: k}
		}
		return items
	}
	const slack = 8
	for total := 0; total <= 2*slack; total++ {
		for sn := 0; sn <= min(total, slack); sn++ {
			sd := total - sn
			if sd > slack || len(names)-sn < 10 {
				continue
			}
			if items := try(sn, nameEnd+8*uint64(sd)); items != nil {
				return &Resources{mem: m, items: items}, nil
			}
		}
	}
	return nil, fmt.Errorf("rode: found %d resource names but no data table that matches them", len(names))
}

// loadELF maps the program's loadable segments and reads its relative
// relocations (64-bit little-endian, as on the Duo).
func loadELF(f *os.File) (memory, []reloc, error) {
	ef, err := elf.NewFile(f)
	if err != nil {
		return nil, nil, fmt.Errorf("rode: %w", err)
	}
	if ef.Class != elf.ELFCLASS64 || ef.Data != elf.ELFDATA2LSB {
		return nil, nil, errors.New("rode: not a 64-bit little-endian program")
	}
	mem := &elfMemory{r: f}
	for _, p := range ef.Progs {
		if p.Type == elf.PT_LOAD && p.Filesz > 0 {
			mem.segs = append(mem.segs, seg{vaddr: p.Vaddr, off: p.Off, size: p.Filesz})
		}
	}
	var relocs []reloc
	for _, s := range ef.Sections {
		if s.Type != elf.SHT_RELA {
			continue
		}
		d, err := s.Data()
		if err != nil {
			continue
		}
		for o := 0; o+24 <= len(d); o += 24 {
			off := binary.LittleEndian.Uint64(d[o:])
			info := binary.LittleEndian.Uint64(d[o+8:])
			add := binary.LittleEndian.Uint64(d[o+16:])
			if elf.R_AARCH64(info&0xffffffff) == elf.R_AARCH64_RELATIVE || elf.R_X86_64(info&0xffffffff) == elf.R_X86_64_RELATIVE && ef.Machine == elf.EM_X86_64 {
				relocs = append(relocs, reloc{slot: off, value: add})
			}
		}
	}
	if len(relocs) == 0 {
		return nil, nil, errors.New("rode: the program has no relative relocations")
	}
	return mem, relocs, nil
}

type seg struct{ vaddr, off, size uint64 }

type elfMemory struct {
	r    io.ReaderAt
	segs []seg
}

func (m *elfMemory) read(a uint64, n int) []byte {
	for _, s := range m.segs {
		if a >= s.vaddr && a < s.vaddr+s.size {
			n = int(min(uint64(n), s.vaddr+s.size-a))
			b := make([]byte, n)
			got, _ := m.r.ReadAt(b, int64(s.off+(a-s.vaddr)))
			return b[:got]
		}
	}
	return nil
}

// Translations reads one language's texts ("german", "french", …): the
// English text (as the program shows it in English) → the translation.
func (r *Resources) Translations(lang string) (map[string]string, bool) {
	b, ok := r.Bytes(lang)
	if !ok || r.Kind(lang) != "txt" {
		return nil, false
	}
	return ParseTranslations(b), true
}

// ParseTranslations reads JUCE's translation format: lines of
// "ENGLISH" = "translation" (with \n, \" and \\ escapes).
func ParseTranslations(b []byte) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, `"`) {
			continue
		}
		k, rest, ok := quoted(line)
		if !ok {
			continue
		}
		rest = strings.TrimSpace(rest)
		if !strings.HasPrefix(rest, "=") {
			continue
		}
		v, _, ok := quoted(strings.TrimSpace(rest[1:]))
		if ok {
			out[k] = v
		}
	}
	return out
}

// quoted reads a "…" string at the start of s and returns the rest.
func quoted(s string) (string, string, bool) {
	if !strings.HasPrefix(s, `"`) {
		return "", "", false
	}
	var b strings.Builder
	for i := 1; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\\' && i+1 < len(s):
			i++
			switch s[i] {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			default:
				b.WriteByte(s[i])
			}
		case c == '"':
			return b.String(), s[i+1:], true
		default:
			b.WriteByte(c)
		}
	}
	return "", "", false
}

// Language names the translation resource for a language code.
func Language(code string) string {
	switch strings.ToLower(code) {
	case "de":
		return "german"
	case "fr":
		return "french"
	case "es":
		return "spanish"
	case "it":
		return "italian"
	case "pt":
		return "portuguese"
	case "ru":
		return "russian"
	case "ja":
		return "japanese"
	case "ko":
		return "korean"
	case "zh":
		return "chinese"
	case "hi":
		return "hindi"
	}
	return "english"
}
