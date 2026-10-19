package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// The display through the kernel's DRM interface, as RØDE's screen program
// uses it: a "dumb" buffer in the panel's mode (720 × 1280, XRGB8888) set
// on the panel's CRTC. Only plain ioctls; the structures are those of
// <drm/drm_mode.h> (the same on arm64 and amd64).
const (
	ioctlSetMaster   = 0x0000641e
	ioctlGetRes      = 0xc04064a0
	ioctlGetCrtc     = 0xc06864a1
	ioctlSetCrtc     = 0xc06864a2
	ioctlGetEncoder  = 0xc01464a6
	ioctlGetConn     = 0xc05064a7
	ioctlAddFB       = 0xc01c64ae
	ioctlRmFB        = 0xc00464af
	ioctlCreateDumb  = 0xc02064b2
	ioctlMapDumb     = 0xc01064b3
	ioctlDestroyDumb = 0xc00464b4

	modeInfoSize = 68 // struct drm_mode_modeinfo
)

var le = binary.LittleEndian

// display is the panel with our buffer on it.
type display struct {
	f              *os.File
	W, H, pitch    int
	crtc, conn, fb uint32
	handle         uint32
	mode           []byte // the mode we set (drm_mode_modeinfo)
	mem            []byte // the buffer, mapped
	ModeName       string
}

// ioctl calls the kernel on f (through SyscallConn, which leaves the file
// non-blocking: a pending read still ends when it is closed).
func ioctl(f *os.File, req uintptr, arg []byte) error {
	rc, err := f.SyscallConn()
	if err != nil {
		return err
	}
	var e syscall.Errno
	if err := rc.Control(func(fd uintptr) {
		_, _, e = syscall.Syscall(syscall.SYS_IOCTL, fd, req, uintptr(unsafe.Pointer(&arg[0])))
	}); err != nil {
		return err
	}
	if e != 0 {
		return e
	}
	return nil
}

// ptr is the address of b for a field the kernel writes through.
func ptr(b []byte) uint64 {
	if len(b) == 0 {
		return 0
	}
	return uint64(uintptr(unsafe.Pointer(&b[0])))
}

// panel is what probing the card found: a connected connector, its first
// (preferred) mode and a CRTC that can drive it.
type panel struct {
	conn, crtc uint32
	mode       []byte
	w, h       int
	name       string
}

// probe finds the panel without changing anything (no master needed).
func probe(f *os.File) (panel, error) {
	res := make([]byte, 64) // drm_mode_card_res
	if err := ioctl(f, ioctlGetRes, res); err != nil {
		return panel{}, fmt.Errorf("resources: %w", err)
	}
	nCrtc, nConn, nEnc := int(le.Uint32(res[36:])), int(le.Uint32(res[40:])), int(le.Uint32(res[44:]))
	crtcs, conns, encs := make([]byte, 4*nCrtc+4), make([]byte, 4*nConn+4), make([]byte, 4*nEnc+4)
	res = make([]byte, 64)
	le.PutUint64(res[8:], ptr(crtcs))
	le.PutUint64(res[16:], ptr(conns))
	le.PutUint64(res[24:], ptr(encs))
	le.PutUint32(res[36:], uint32(nCrtc))
	le.PutUint32(res[40:], uint32(nConn))
	le.PutUint32(res[44:], uint32(nEnc))
	if err := ioctl(f, ioctlGetRes, res); err != nil {
		return panel{}, fmt.Errorf("resources: %w", err)
	}
	for i := 0; i < nConn; i++ {
		id := le.Uint32(conns[4*i:])
		c := make([]byte, 80) // drm_mode_get_connector
		le.PutUint32(c[48:], id)
		if err := ioctl(f, ioctlGetConn, c); err != nil {
			continue
		}
		nModes, nEncs := int(le.Uint32(c[32:])), int(le.Uint32(c[40:]))
		if le.Uint32(c[60:]) != 1 || nModes == 0 { // 1: connected
			continue
		}
		modes, cencs := make([]byte, modeInfoSize*nModes), make([]byte, 4*nEncs+4)
		c = make([]byte, 80)
		le.PutUint64(c[0:], ptr(cencs))
		le.PutUint64(c[8:], ptr(modes))
		le.PutUint32(c[32:], uint32(nModes))
		le.PutUint32(c[40:], uint32(nEncs))
		le.PutUint32(c[48:], id)
		if err := ioctl(f, ioctlGetConn, c); err != nil || int(le.Uint32(c[32:])) < 1 {
			continue
		}
		mode := append([]byte(nil), modes[:modeInfoSize]...)
		p := panel{conn: id, mode: mode, w: int(le.Uint16(mode[4:])), h: int(le.Uint16(mode[14:])), name: cstr(mode[36:68])}
		// The CRTC: the one the connector's encoder drives now, else the
		// first the encoder can drive.
		encIDs := []uint32{le.Uint32(c[44:])}
		for j := 0; j < int(le.Uint32(c[40:])) && j < nEncs; j++ {
			encIDs = append(encIDs, le.Uint32(cencs[4*j:]))
		}
		for _, eid := range encIDs {
			if eid == 0 {
				continue
			}
			e := make([]byte, 20) // drm_mode_get_encoder
			le.PutUint32(e[0:], eid)
			if ioctl(f, ioctlGetEncoder, e) != nil {
				continue
			}
			if cur := le.Uint32(e[8:]); cur != 0 {
				p.crtc = cur
				return p, nil
			}
			possible := le.Uint32(e[12:])
			for k := 0; k < nCrtc && k < 32; k++ {
				if possible&(1<<k) != 0 {
					p.crtc = le.Uint32(crtcs[4*k:])
					return p, nil
				}
			}
		}
	}
	return panel{}, errors.New("no connected panel")
}

func cstr(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

// openDisplay takes the panel: a buffer in its mode, shown on its CRTC.
func openDisplay(card string) (*display, error) {
	f, err := os.OpenFile(card, os.O_RDWR|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	d := &display{f: f}
	if err := d.setup(); err != nil {
		d.Close()
		return nil, err
	}
	return d, nil
}

func (d *display) setup() error {
	// Master: the first program to open the card has it anyway; RØDE's
	// screen program has ended, so nobody else holds it.
	_ = ioctl(d.f, ioctlSetMaster, make([]byte, 8))
	if err := d.buffer(); err != nil {
		return err
	}
	return d.show()
}

// buffer finds the panel and makes a buffer in its mode, mapped and black.
func (d *display) buffer() error {
	p, err := probe(d.f)
	if err != nil {
		return err
	}
	d.W, d.H, d.conn, d.crtc, d.mode, d.ModeName = p.w, p.h, p.conn, p.crtc, p.mode, p.name

	cd := make([]byte, 32) // drm_mode_create_dumb
	le.PutUint32(cd[0:], uint32(d.H))
	le.PutUint32(cd[4:], uint32(d.W))
	le.PutUint32(cd[8:], 32)
	if err := ioctl(d.f, ioctlCreateDumb, cd); err != nil {
		return fmt.Errorf("buffer: %w", err)
	}
	d.handle, d.pitch = le.Uint32(cd[16:]), int(le.Uint32(cd[20:]))
	size := int(le.Uint64(cd[24:]))

	fb := make([]byte, 28) // drm_mode_fb_cmd
	le.PutUint32(fb[4:], uint32(d.W))
	le.PutUint32(fb[8:], uint32(d.H))
	le.PutUint32(fb[12:], uint32(d.pitch))
	le.PutUint32(fb[16:], 32)
	le.PutUint32(fb[20:], 24)
	le.PutUint32(fb[24:], d.handle)
	if err := ioctl(d.f, ioctlAddFB, fb); err != nil {
		return fmt.Errorf("frame buffer: %w", err)
	}
	d.fb = le.Uint32(fb[0:])

	md := make([]byte, 16) // drm_mode_map_dumb
	le.PutUint32(md[0:], d.handle)
	if err := ioctl(d.f, ioctlMapDumb, md); err != nil {
		return fmt.Errorf("map: %w", err)
	}
	rc, err := d.f.SyscallConn()
	if err != nil {
		return err
	}
	if cerr := rc.Control(func(fd uintptr) {
		d.mem, err = syscall.Mmap(int(fd), int64(le.Uint64(md[8:])), size, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	}); cerr != nil {
		err = cerr
	}
	if err != nil {
		return fmt.Errorf("map: %w", err)
	}
	clear(d.mem)
	return nil
}

// show puts our buffer on the panel.
func (d *display) show() error {
	conns := make([]byte, 8)
	le.PutUint32(conns, d.conn)
	c := make([]byte, 104) // drm_mode_crtc
	le.PutUint64(c[0:], ptr(conns))
	le.PutUint32(c[8:], 1)
	le.PutUint32(c[12:], d.crtc)
	le.PutUint32(c[16:], d.fb)
	le.PutUint32(c[32:], 1) // mode_valid
	copy(c[36:], d.mode)
	if err := ioctl(d.f, ioctlSetCrtc, c); err != nil {
		return fmt.Errorf("showing the buffer: %w", err)
	}
	return nil
}

// Draw copies a whole frame (pitch × H bytes, as turn writes it) to the
// panel at once.
func (d *display) Draw(frame []byte) { copy(d.mem, frame) }

// Pitch is the length of a row in bytes.
func (d *display) Pitch() int { return d.pitch }

// Close gives the panel up: the buffer goes; RØDE's screen program sets
// its own when it starts again.
func (d *display) Close() {
	if d.mem != nil {
		_ = syscall.Munmap(d.mem)
	}
	if d.fb != 0 {
		b := make([]byte, 4)
		le.PutUint32(b, d.fb)
		_ = ioctl(d.f, ioctlRmFB, b)
	}
	if d.handle != 0 {
		b := make([]byte, 4)
		le.PutUint32(b, d.handle)
		_ = ioctl(d.f, ioctlDestroyDumb, b)
	}
	_ = d.f.Close()
}
