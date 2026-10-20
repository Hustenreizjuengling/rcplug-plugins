//go:build linux

package panel

import (
	"fmt"
	"image"
	"os"
	"syscall"
	"unsafe"
)

// The touchscreen through evdev, as RØDE's screen program reads it. The Duo
// has a Goodix GT911 (/dev/input/event0). The driver announces the ranges
// its device tree names (800 × 1280) but the chip reports the panel's own
// pixels (720 × 1280): scaling by the announced range squeezed every touch
// towards one edge (a tenth of the screen at the top), so a range that is
// at least the panel's size is taken as pixels; a smaller one is scaled.
const (
	evSyn = 0x00
	evKey = 0x01
	evAbs = 0x03

	synReport = 0x00
	btnTouch  = 0x14a

	absX         = 0x00
	absY         = 0x01
	absMTSlot    = 0x2f
	absMTX       = 0x35
	absMTY       = 0x36
	absMTTracker = 0x39

	eventSize = 24 // struct input_event on 64-bit
)

// ioctlTouch calls EVIOCGABS on the touchscreen fd.
func ioctlTouch(f *os.File, req uintptr, buf []byte) error {
	rc, err := f.SyscallConn()
	if err != nil {
		return err
	}
	var e syscall.Errno
	if cerr := rc.Control(func(fd uintptr) {
		_, _, e = syscall.Syscall(syscall.SYS_IOCTL, fd, req, uintptr(unsafe.Pointer(&buf[0])))
	}); cerr != nil {
		return cerr
	}
	if e != 0 {
		return e
	}
	return nil
}

// eviocgabs is EVIOCGABS(abs): the range of an axis (struct input_absinfo).
func eviocgabs(abs int) uintptr { return 0x80184540 + uintptr(abs) }

type axis struct{ min, max int32 }

// touchscreen reads one device.
type touchscreen struct {
	f    *os.File
	x, y axis
	mt   bool // positions from the multi-touch axes (first slot)
}

func openTouch(dev string) (*touchscreen, error) {
	f, err := os.OpenFile(dev, os.O_RDONLY|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	t := &touchscreen{f: f}
	get := func(abs int) (axis, bool) {
		b := make([]byte, 24)
		if ioctlTouch(f, eviocgabs(abs), b) != nil {
			return axis{}, false
		}
		a := axis{int32(le.Uint32(b[4:])), int32(le.Uint32(b[8:]))}
		return a, a.max > a.min
	}
	// The single-touch axes first: the kernel reports the first finger on
	// them for every multi-touch screen; the multi-touch axes (first slot)
	// only if they are missing.
	x, okX := get(absX)
	y, okY := get(absY)
	if !okX || !okY {
		x, okX = get(absMTX)
		y, okY = get(absMTY)
		t.mt = true
	}
	t.x, t.y = x, y
	if !okX || !okY {
		f.Close()
		return nil, fmt.Errorf("%s reports no positions", dev)
	}
	return t, nil
}

func (t *touchscreen) String() string {
	return fmt.Sprintf("x %d…%d, y %d…%d, multi-touch %v", t.x.min, t.x.max, t.y.min, t.y.max, t.mt)
}

// Run sends what the finger does until the device is closed.
func (t *touchscreen) Run(w, h int, out chan<- TouchEvent) {
	buf := make([]byte, eventSize*64)
	var x, y int32
	down, was, slot := false, false, int32(0)
	moved := false
	scale := func(v int32, a axis, n int) int { return mapAxis(v, a, n) }
	for {
		n, err := t.f.Read(buf)
		if err != nil {
			close(out)
			return
		}
		for o := 0; o+eventSize <= n; o += eventSize {
			typ, code, val := le.Uint16(buf[o+16:]), le.Uint16(buf[o+18:]), int32(le.Uint32(buf[o+20:]))
			switch {
			case typ == evKey && code == btnTouch:
				down = val != 0
			case typ == evAbs && code == absMTSlot:
				slot = val
			case typ == evAbs && t.mt && slot == 0 && code == absMTX,
				typ == evAbs && !t.mt && code == absX:
				x, moved = val, true
			case typ == evAbs && t.mt && slot == 0 && code == absMTY,
				typ == evAbs && !t.mt && code == absY:
				y, moved = val, true
			case typ == evAbs && t.mt && slot == 0 && code == absMTTracker && val < 0:
				down = false
			case typ == evSyn && code == synReport:
				if down || was {
					if down != was || moved {
						out <- TouchEvent{Down: down, At: image.Pt(scale(x, t.x, w), scale(y, t.y, h))}
					}
				}
				was, moved = down, false
			}
		}
	}
}

func (t *touchscreen) Close() { _ = t.f.Close() }

// mapAxis maps a reported coordinate onto n pixels: as is when the device's
// range covers the panel (its values are pixels), scaled when it is smaller.
func mapAxis(v int32, a axis, n int) int {
	span := int64(a.max-a.min) + 1
	var p int
	if span >= int64(n) {
		p = int(v - a.min)
	} else {
		p = int(int64(v-a.min) * int64(n) / span)
	}
	return max(0, min(n-1, p))
}
