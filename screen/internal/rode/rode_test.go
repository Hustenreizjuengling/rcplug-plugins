package rode

import (
	"os"
	"strings"
	"testing"
)

// fakeMem is a sparse memory image.
type fakeMem map[uint64][]byte

func (m fakeMem) read(a uint64, n int) []byte {
	for base, b := range m {
		if a >= base && a < base+uint64(len(b)) {
			b = b[a-base:]
			if len(b) > n {
				b = b[:n]
			}
			return b
		}
	}
	return nil
}

func TestFindTables(t *testing.T) {
	mem := fakeMem{}
	var relocs []reloc
	names := []string{"IconHome_svg", "AphexRotary_png", "Knob_png", "german_txt"}
	for i := 0; i < 8; i++ { // padding so the table is long enough
		names = append(names, "Pad"+string(rune('A'+i))+"_svg")
	}
	data := map[string]string{
		"IconHome_svg":    `<svg viewBox="0 0 1 1"></svg>`,
		"AphexRotary_png": `<svg viewBox="0 0 2 2"></svg>`, // a "png" holding SVG, as RØDE has
		"Knob_png":        "\x89PNG\r\n\x1a\n....IEND\xaeB`\x82",
		"german_txt":      "language: german\ncountries: de\n\n\"MIXER\" = \"Mischpult\"\n\"A \\\"Q\\\"\" = \"B\\nC\"\n\x00",
	}
	addr := uint64(0x10000)
	nameTable := uint64(0x900000)
	// Two stray pointers lie between the tables, as in RØDE's program.
	dataTable := nameTable + 8*uint64(len(names)) + 16
	for i, n := range names {
		mem[addr] = append([]byte(n), 0)
		relocs = append(relocs, reloc{slot: nameTable + 8*uint64(i), value: addr})
		addr += 0x100
		d := data[n]
		if d == "" {
			d = `<svg viewBox="0 0 1 1"></svg>`
		}
		mem[addr] = []byte(d)
		relocs = append(relocs, reloc{slot: dataTable + 8*uint64(i), value: addr})
		addr += 0x1000
	}
	relocs = append(relocs, reloc{slot: nameTable + 8*uint64(len(names)), value: 0x500}, reloc{slot: nameTable + 8*uint64(len(names)) + 8, value: 0x600})
	r, err := find(mem, relocs)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Names()) != len(names) {
		t.Fatalf("names %v", r.Names())
	}
	if r.Kind("AphexRotary") != "svg" || r.Kind("Knob") != "png" || r.Kind("german") != "txt" {
		t.Fatalf("kinds %q %q %q", r.Kind("AphexRotary"), r.Kind("Knob"), r.Kind("german"))
	}
	if b, ok := r.Bytes("IconHome"); !ok || string(b) != data["IconHome_svg"] {
		t.Fatalf("IconHome: %q", b)
	}
	if b, ok := r.Bytes("Knob"); !ok || !strings.HasSuffix(string(b), "IEND\xaeB`\x82") {
		t.Fatalf("Knob: %q", b)
	}
	tr, ok := r.Translations("german")
	if !ok || tr["MIXER"] != "Mischpult" || tr[`A "Q"`] != "B\nC" {
		t.Fatalf("translations %q", tr)
	}
	// Without matching data there is no table.
	if _, err := find(fakeMem{0x10: []byte("IconA_svg\x00")}, []reloc{{0x900000, 0x10}}); err == nil {
		t.Fatal("a single name was taken for a table")
	}
}

func TestLanguage(t *testing.T) {
	if Language("de") != "german" || Language("xx") != "english" {
		t.Fatal(Language("de"))
	}
}

// TestRealProgram reads RØDE's program when one is at hand (never part of
// rcplug): RCPLUG_RODE_GUI=<path to rc_gui_juce>.
func TestRealProgram(t *testing.T) {
	path := os.Getenv("RCPLUG_RODE_GUI")
	if path == "" {
		t.Skip("set RCPLUG_RODE_GUI to RØDE's screen program")
	}
	r, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	n := r.Names()
	t.Logf("%d resources", len(n))
	if len(n) < 400 {
		t.Fatalf("only %d resources", len(n))
	}
	for _, name := range []string{"IconHome", "procCompressor", "AphexRotary", "german"} {
		if b, ok := r.Bytes(name); !ok || len(b) < 20 {
			t.Errorf("%s: %d bytes", name, len(b))
		}
	}
	de, ok := r.Translations("german")
	if !ok || de["MIXER"] == "" {
		t.Fatalf("german texts: %d", len(de))
	}
	t.Logf("german: %d texts, MIXER = %q", len(de), de["MIXER"])
}
