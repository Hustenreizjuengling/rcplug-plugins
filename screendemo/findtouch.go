package main

import (
	"os"
	"path/filepath"
	"strings"
)

// sysInput lists the input devices with their names.
const sysInput = "/sys/class/input"

// findTouch is the touchscreen's event device, found by its name (the Duo's
// is a Goodix): RØDE's screen program binds the touchscreen's driver again
// when it starts, so its number can change. /dev/input/event0 otherwise.
func findTouch(sys string) string {
	dirs, _ := filepath.Glob(filepath.Join(sys, "event*"))
	for _, d := range dirs {
		b, err := os.ReadFile(filepath.Join(d, "device", "name"))
		if err != nil {
			continue
		}
		n := strings.ToLower(string(b))
		if strings.Contains(n, "goodix") || strings.Contains(n, "touch") {
			return "/dev/input/" + filepath.Base(d)
		}
	}
	return "/dev/input/event0"
}
