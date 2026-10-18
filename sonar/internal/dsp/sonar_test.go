package dsp

import (
	"math"
	"math/rand"
	"testing"
)

const rate = 48000

// toneBlock returns n frames of a sine at freq with independent L/R gains.
func toneBlock(n int, freq, gainL, gainR float64) []int16 {
	out := make([]int16, 2*n)
	for i := 0; i < n; i++ {
		s := math.Sin(2 * math.Pi * freq * float64(i) / rate)
		out[2*i] = int16(gainL * s * 30000)
		out[2*i+1] = int16(gainR * s * 30000)
	}
	return out
}

// delayedNoise returns broadband noise where R is a copy of L delayed by d
// samples (d>0 ⇒ R later ⇒ source toward the left).
func delayedNoise(n, d int, seed int64) []int16 {
	rng := rand.New(rand.NewSource(seed))
	base := make([]float64, n+d+8)
	for i := range base {
		base[i] = rng.NormFloat64() * 0.2
	}
	out := make([]int16, 2*n)
	for i := 0; i < n; i++ {
		l := base[i+d] // L is "earlier" in time
		r := base[i]   // R is the delayed copy
		out[2*i] = int16(clampF(l) * 30000)
		out[2*i+1] = int16(clampF(r) * 30000)
	}
	return out
}

func clampF(x float64) float64 {
	if x > 1 {
		return 1
	}
	if x < -1 {
		return -1
	}
	return x
}

func steady(p *Processor, block []int16, reps int) Estimate {
	var e Estimate
	for i := 0; i < reps; i++ {
		e = p.ProcessInterleaved(block)
	}
	return e
}

func TestILDHardLeft(t *testing.T) {
	p := NewProcessor(DefaultConfig(rate))
	e := steady(p, toneBlock(2048, 440, 1.0, 0.05), 20)
	if e.Azimuth > -40 {
		t.Fatalf("hard-left tone should read strongly negative, got az=%.1f", e.Azimuth)
	}
	if e.L <= e.R {
		t.Fatalf("left level should exceed right: L=%.2f R=%.2f", e.L, e.R)
	}
}

func TestILDHardRight(t *testing.T) {
	p := NewProcessor(DefaultConfig(rate))
	e := steady(p, toneBlock(2048, 440, 0.05, 1.0), 20)
	if e.Azimuth < 40 {
		t.Fatalf("hard-right tone should read strongly positive, got az=%.1f", e.Azimuth)
	}
}

func TestCenter(t *testing.T) {
	p := NewProcessor(DefaultConfig(rate))
	e := steady(p, toneBlock(2048, 440, 1.0, 1.0), 20)
	if math.Abs(e.Azimuth) > 8 {
		t.Fatalf("centred tone should read near 0, got az=%.1f", e.Azimuth)
	}
}

func TestITDDelayPointsLeft(t *testing.T) {
	cfg := DefaultConfig(rate)
	cfg.Smoothing = 0 // measure the raw estimate
	p := NewProcessor(cfg)
	// R delayed by 20 samples (~417µs) relative to L, equal amplitude.
	e := steady(p, delayedNoise(4096, 20, 1), 5)
	if e.Azimuth >= 0 {
		t.Fatalf("R-delayed signal should read left (negative), got az=%.1f", e.Azimuth)
	}
	if e.Conf < 0.3 {
		t.Fatalf("coherent delayed copy should have decent confidence, got %.2f", e.Conf)
	}
	if e.LagUs <= 50 {
		t.Fatalf("expected a positive lag estimate around 417µs, got %.0fµs", e.LagUs)
	}
}

func TestSilenceLowConfidence(t *testing.T) {
	p := NewProcessor(DefaultConfig(rate))
	quiet := make([]int16, 4096) // all zero
	e := steady(p, quiet, 5)
	if e.Conf != 0 {
		t.Fatalf("silence must have zero confidence, got %.2f", e.Conf)
	}
}

func TestFlipSign(t *testing.T) {
	cfg := DefaultConfig(rate)
	cfg.FlipSign = true
	p := NewProcessor(cfg)
	e := steady(p, toneBlock(2048, 440, 1.0, 0.05), 20)
	if e.Azimuth < 40 {
		t.Fatalf("with FlipSign, hard-left tone should read positive, got az=%.1f", e.Azimuth)
	}
}
