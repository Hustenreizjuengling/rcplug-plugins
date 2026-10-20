//go:build !linux

package main

import (
	"errors"
	"image"
)

// The display and the touchscreen exist only on the device (Linux); elsewhere
// only --png works.

var errNotLinux = errors.New("the device's screen needs Linux; use --png")

type display struct {
	W, H, Refresh int
	ModeName      string
	Flips         bool
	bufs          [][]byte
}

func openDisplay(string) (*display, error) { return nil, errNotLinux }
func (d *display) Back() []byte            { return nil }
func (d *display) Present() (bool, error)  { return false, nil }
func (d *display) Pitch() int              { return 0 }
func (d *display) Close()                  {}
func unameSRM() string                     { return "" }

type touchscreen struct{}

type touchEvent struct {
	Down bool
	At   image.Point
}

func openTouch(string) (*touchscreen, error)           { return nil, errNotLinux }
func (t *touchscreen) Run(int, int, chan<- touchEvent) {}
func (t *touchscreen) String() string                  { return "" }
func (t *touchscreen) Close()                          {}
