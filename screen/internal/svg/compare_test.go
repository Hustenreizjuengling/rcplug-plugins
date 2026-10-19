package svg

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
)

// TestSheet draws every SVG of a folder on a grid, for comparing with a
// browser by eye or by numbers. It runs only when asked:
// RCPLUG_SVG_DIR=<folder> RCPLUG_SVG_OUT=<sheet.png>.
func TestSheet(t *testing.T) {
	dir, out := os.Getenv("RCPLUG_SVG_DIR"), os.Getenv("RCPLUG_SVG_OUT")
	if dir == "" || out == "" {
		t.Skip("set RCPLUG_SVG_DIR and RCPLUG_SVG_OUT")
	}
	names, _ := filepath.Glob(filepath.Join(dir, "*.svg"))
	sort.Strings(names)
	const cell, cols = 128, 25
	rows := (len(names) + cols - 1) / cols
	sheet := gfx.NewImage(cols*cell, rows*cell)
	sheet.Clear(gfx.Hex(0x808080))
	var list []string
	for i, n := range names {
		b, err := os.ReadFile(n)
		if err != nil {
			t.Fatal(err)
		}
		list = append(list, filepath.Base(n))
		doc, err := Parse(b)
		if err != nil {
			t.Errorf("%s: %v", filepath.Base(n), err)
			continue
		}
		x, y := i%cols*cell, i/cols*cell
		doc.Draw(sheet, image.Rect(x, y, x+cell, y+cell))
	}
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, sheet.ToNRGBA()); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(strings.TrimSuffix(out, ".png")+".txt", []byte(strings.Join(list, "\n")+"\n"), 0o644)
}
