package main

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// backlights is where the kernel lists the panel's backlight (the Duo has a
// PWM backlight, levels 0 to 255).
const backlights = "/sys/class/backlight"

// light makes sure the page can be seen: RØDE's screen program may have
// dimmed or switched off the backlight before it was stopped. Only a
// backlight below a tenth is turned up (to half); the returned function
// puts back what was there, before RØDE's screen program starts again and
// sets its own.
func light(root string) (restore func()) {
	var undo []func()
	dirs, _ := filepath.Glob(filepath.Join(root, "*"))
	for _, d := range dirs {
		if p, err := readInt(filepath.Join(d, "bl_power")); err == nil && p != 0 {
			if set(filepath.Join(d, "bl_power"), 0) {
				log.Printf("screendemo: backlight %s was off; on while the page is shown", filepath.Base(d))
				undo = append(undo, func() { set(filepath.Join(d, "bl_power"), p) })
			}
		}
		maxi, err1 := readInt(filepath.Join(d, "max_brightness"))
		cur, err2 := readInt(filepath.Join(d, "brightness"))
		if err1 != nil || err2 != nil || maxi <= 0 || cur*10 >= maxi {
			continue
		}
		if set(filepath.Join(d, "brightness"), maxi/2) {
			log.Printf("screendemo: backlight %s was at %d of %d; at %d while the page is shown", filepath.Base(d), cur, maxi, maxi/2)
			undo = append(undo, func() { set(filepath.Join(d, "brightness"), cur) })
		}
	}
	return func() {
		for i := len(undo) - 1; i >= 0; i-- {
			undo[i]()
		}
	}
}

func readInt(path string) (int, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(b)))
}

func set(path string, v int) bool {
	if err := os.WriteFile(path, []byte(strconv.Itoa(v)), 0); err != nil {
		log.Printf("screendemo: %v", err)
		return false
	}
	return true
}
