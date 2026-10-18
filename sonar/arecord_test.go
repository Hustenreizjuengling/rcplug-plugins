package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeArecord puts a script named arecord first in PATH.
func fakeArecord(t *testing.T, script string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "arecord"), []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// Closing from two places at once (the capture loop's defer and the context
// ending) must not hang: exec.Cmd.Wait may only run once.
func TestArecordCloseTwiceAtOnce(t *testing.T) {
	fakeArecord(t, "exec cat /dev/zero\n")
	for i := 0; i < 20; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		src, err := newArecordSource(ctx, "hw:9,9", 48000)
		if err != nil {
			t.Fatal(err)
		}
		buf := make([]byte, 4096)
		if _, err := io.ReadFull(src, buf); err != nil {
			t.Fatal(err)
		}
		done := make(chan struct{})
		go func() {
			var wg sync.WaitGroup
			for k := 0; k < 2; k++ {
				wg.Add(1)
				go func() { defer wg.Done(); src.Close() }()
			}
			cancel()
			wg.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatalf("round %d: closing hung", i)
		}
	}
}

// When arecord gives up, the error says why, not just EOF.
func TestArecordFailureIsExplained(t *testing.T) {
	fakeArecord(t, "echo 'arecord: main:850: audio open error: No such device' >&2\nexit 1\n")
	src, err := newArecordSource(context.Background(), "hw:9,9", 48000)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	_, err = io.ReadFull(src, make([]byte, 4096))
	if err == nil || !strings.Contains(err.Error(), "No such device") || !strings.Contains(err.Error(), "exit status 1") {
		t.Fatalf("error = %v", err)
	}
}

// On the desk, devices that RØDE's mixer holds (or that do not exist there)
// are not offered.
func TestAlsaDevicesOnTheDesk(t *testing.T) {
	list := "null\n    Discard all samples\npulse\n    PulseAudio\ndefault\n    Default\nsysdefault:CARD=UAC2Gadget\n    UAC2_Gadget\nhw:CARD=RODECasterDUO,DEV=0\n    RODECaster DUO\nplughw:CARD=Loopback,DEV=1\n    Loopback\n"
	if got := strings.Join(alsaDevices(list, true), ","); got != "" {
		t.Fatalf("on the desk: %s", got)
	}
	if got := len(alsaDevices(list, false)); got != 6 {
		t.Fatalf("elsewhere: %d devices", got)
	}
}
