// Package dsp turns blocks of interleaved stereo PCM into a direction estimate
// for the Sonar plugin. It combines two classic localisation cues:
//
//   - ILD (inter-level difference): which channel is louder → left/right balance.
//   - ITD (inter-channel time/phase difference): the earlier channel points
//     toward the source. Estimated by normalised cross-correlation; the peak
//     correlation doubles as a confidence score.
//
// The result is an azimuth in [-90°,+90°] (positive = right), an overall level,
// a confidence and a rough distance. A stereo pair tells left from right only:
// front and back sound alike to it, and it has no sense of height.
//
// Distance combines two cues: the level (closer is louder) and how much of
// what the two microphones hear is the same signal (the peak of their
// cross-correlation). Close to the microphones the direct sound dominates and
// both hear nearly the same; further away the room's reverberation, which
// differs between the two, takes over. Both depend on the room and the
// source, so a calibration (a reading near and one far) makes it usable. Everything is pure Go and allocation-light so it runs
// comfortably at lowest priority on the device.
package dsp

import "math"

// Estimate is one direction reading.
type Estimate struct {
	Azimuth float64 `json:"az"`      // degrees, -90 (hard left) .. +90 (hard right)
	Level   float64 `json:"level"`   // overall level, 0..1 (mapped from dBFS)
	LevelDb float64 `json:"levelDb"` // overall level in dBFS (<=0)
	Conf    float64 `json:"conf"`    // 0..1 confidence (phase coherence × level gate)
	L       float64 `json:"l"`       // left level 0..1
	R       float64 `json:"r"`       // right level 0..1
	LagUs   float64 `json:"lagUs"`   // inter-channel delay estimate, microseconds (R vs L)
	Dist    float64 `json:"dist"`    // rough distance, 0 (near) .. 1 (far)
	Coh     float64 `json:"coh"`     // peak cross-correlation of the block, 0..1 (before smoothing)
}

// Config tunes the processor.
type Config struct {
	Rate       int     // sample rate (Hz)
	Smoothing  float64 // EMA factor 0..1; higher = smoother/slower. 0 disables.
	MaxLagUs   float64 // max |ITD| considered, microseconds (human max ≈ 700µs)
	NoiseFloor float64 // dBFS below which input is treated as silence
	FlipSign   bool    // flip left/right orientation
	// Calibration for the distance: level (dBFS) and correlation measured
	// near the microphones and far from them. Zero values: not calibrated.
	NearDb, FarDb   float64
	NearCoh, FarCoh float64
}

// Calibrated reports whether near and far readings are set and near is
// clearly (3 dB) louder.
func (c Config) Calibrated() bool {
	return c.NearDb != 0 && c.FarDb != 0 && c.NearDb > c.FarDb+3
}

// DefaultConfig returns sensible defaults for a 48 kHz line signal.
func DefaultConfig(rate int) Config {
	return Config{
		Rate:       rate,
		Smoothing:  0.6,
		MaxLagUs:   900,
		NoiseFloor: -60,
		FlipSign:   false,
	}
}

// Processor holds smoothing state across blocks.
type Processor struct {
	cfg    Config
	maxLag int
	azS    float64
	lvlS   float64
	distS  float64
	primed bool
}

func NewProcessor(cfg Config) *Processor {
	if cfg.Rate <= 0 {
		cfg.Rate = 48000
	}
	maxLag := int(cfg.MaxLagUs * float64(cfg.Rate) / 1e6)
	if maxLag < 1 {
		maxLag = 1
	}
	return &Processor{cfg: cfg, maxLag: maxLag}
}

// Config returns the active configuration.
func (p *Processor) Config() Config { return p.cfg }

// SetConfig changes tuning (smoothing, noise floor, flip, max lag) without
// resetting the smoothing state, so live tweaks don't make the display jump.
// The sample rate cannot change on a running processor.
func (p *Processor) SetConfig(cfg Config) {
	cfg.Rate = p.cfg.Rate
	p.cfg = cfg
	p.maxLag = int(cfg.MaxLagUs * float64(cfg.Rate) / 1e6)
	if p.maxLag < 1 {
		p.maxLag = 1
	}
}

// ProcessInterleaved accepts interleaved 16-bit stereo samples (L,R,L,R,…) and
// returns an Estimate. A short or odd-length block still produces a reading.
func (p *Processor) ProcessInterleaved(pcm []int16) Estimate {
	n := len(pcm) / 2
	if n == 0 {
		return p.decay()
	}
	// Deinterleave into normalised float [-1,1].
	l := make([]float64, n)
	r := make([]float64, n)
	var sumL2, sumR2 float64
	for i := 0; i < n; i++ {
		lv := float64(pcm[2*i]) / 32768.0
		rv := float64(pcm[2*i+1]) / 32768.0
		l[i] = lv
		r[i] = rv
		sumL2 += lv * lv
		sumR2 += rv * rv
	}
	return p.process(l, r, sumL2, sumR2)
}

// ProcessFloat accepts separate left/right float channels (for tests and WAV).
func (p *Processor) ProcessFloat(l, r []float64) Estimate {
	n := len(l)
	if len(r) < n {
		n = len(r)
	}
	if n == 0 {
		return p.decay()
	}
	l, r = l[:n], r[:n]
	var sumL2, sumR2 float64
	for i := 0; i < n; i++ {
		sumL2 += l[i] * l[i]
		sumR2 += r[i] * r[i]
	}
	return p.process(l, r, sumL2, sumR2)
}

func (p *Processor) process(l, r []float64, sumL2, sumR2 float64) Estimate {
	n := len(l)
	rmsL := math.Sqrt(sumL2 / float64(n))
	rmsR := math.Sqrt(sumR2 / float64(n))
	const eps = 1e-9

	// Overall level in dBFS and a 0..1 linear map over [NoiseFloor, 0] dB.
	rms := math.Sqrt((sumL2 + sumR2) / float64(2*n))
	levelDb := 20 * math.Log10(rms+eps)
	levelLin := (levelDb - p.cfg.NoiseFloor) / (0 - p.cfg.NoiseFloor)
	levelLin = clamp01(levelLin)

	// Silence gate: below the noise floor we keep no direction.
	if levelDb < p.cfg.NoiseFloor {
		return p.decay()
	}

	// ILD cue: signed balance in [-1,1] → azimuth.
	balance := (rmsR - rmsL) / (rmsR + rmsL + eps)
	azILD := balance * 90

	// ITD cue: normalised cross-correlation over ±maxLag.
	bestLag, coh := bestLagCorr(l, r, p.maxLag)
	// Positive bestLag means R is delayed after L (R[n] ≈ L[n-lag]), i.e. the
	// wavefront reached L first → source toward the left → negative azimuth.
	azITD := -float64(bestLag) / float64(p.maxLag) * 90
	conf := clamp01(coh)

	// Combine: ILD is reliable for amplitude-panned signals and always counts;
	// ITD is added in proportion to phase coherence. Confidence is additionally
	// gated by level so quiet, noisy input reads as low confidence.
	az := azILD*0.65 + azITD*0.35*conf
	near := p.nearness(levelDb, conf)
	if p.cfg.FlipSign {
		az = -az
	}
	az = clampDeg(az)
	confOut := conf * clamp01(levelLin*1.5)

	// Temporal smoothing (EMA).
	a := p.cfg.Smoothing
	dist := 1 - near
	if !p.primed {
		p.azS, p.lvlS, p.distS, p.primed = az, levelLin, dist, true
	} else if a > 0 {
		p.azS = a*p.azS + (1-a)*az
		p.lvlS = a*p.lvlS + (1-a)*levelLin
		// Distance moves slower: single words should not make it jump.
		ad := math.Max(a, 0.85)
		p.distS = ad*p.distS + (1-ad)*dist
	} else {
		p.azS, p.lvlS, p.distS = az, levelLin, dist
	}

	lagUs := float64(bestLag) / float64(p.cfg.Rate) * 1e6
	if p.cfg.FlipSign {
		lagUs = -lagUs
	}
	return Estimate{
		Azimuth: p.azS,
		Level:   p.lvlS,
		LevelDb: levelDb,
		Conf:    confOut,
		L:       clamp01((20*math.Log10(rmsL+eps) - p.cfg.NoiseFloor) / -p.cfg.NoiseFloor),
		R:       clamp01((20*math.Log10(rmsR+eps) - p.cfg.NoiseFloor) / -p.cfg.NoiseFloor),
		LagUs:   lagUs,
		Dist:    p.distS,
		Coh:     conf,
	}
}

// nearness is 1 for a source at the microphones and 0 for one far away,
// from the level and the correlation of the two channels: between the
// calibrated readings if there are some, otherwise on typical scales (speech
// at -15 dBFS close, 30 dB lower far; correlation 0.9 close, 0.3 far).
func (p *Processor) nearness(levelDb, coh float64) float64 {
	c := p.cfg
	nearDb, farDb, nearCoh, farCoh := -15.0, -45.0, 0.9, 0.3
	wLevel := 0.5
	if c.Calibrated() {
		nearDb, farDb = c.NearDb, c.FarDb
		if c.NearCoh > c.FarCoh+0.05 {
			nearCoh, farCoh = c.NearCoh, c.FarCoh
		} else {
			wLevel = 0.85 // the correlation did not tell near from far here
		}
	}
	lv := clamp01((levelDb - farDb) / (nearDb - farDb))
	ch := clamp01((coh - farCoh) / (nearCoh - farCoh))
	return wLevel*lv + (1-wLevel)*ch
}

// decay relaxes the smoothed state toward centre/silence when there is no input.
func (p *Processor) decay() Estimate {
	p.lvlS *= 0.8
	p.azS *= 0.9
	return Estimate{Azimuth: p.azS, Level: p.lvlS, LevelDb: p.cfg.NoiseFloor, Conf: 0, Dist: p.distS}
}

// bestLagCorr returns the integer lag in [-maxLag,maxLag] that maximises the
// normalised cross-correlation between l and r, and that peak value in [-1,1].
// A positive lag means r is delayed relative to l.
func bestLagCorr(l, r []float64, maxLag int) (int, float64) {
	n := len(l)
	if n < 4 {
		return 0, 0
	}
	// Precompute energies for normalisation.
	var el, er float64
	for i := 0; i < n; i++ {
		el += l[i] * l[i]
		er += r[i] * r[i]
	}
	norm := math.Sqrt(el*er) + 1e-12
	bestLag, best := 0, -2.0
	for lag := -maxLag; lag <= maxLag; lag++ {
		var acc float64
		// Sum over the overlapping region of l[i] * r[i+lag].
		var i0, i1 int
		if lag >= 0 {
			i0, i1 = 0, n-lag
		} else {
			i0, i1 = -lag, n
		}
		for i := i0; i < i1; i++ {
			acc += l[i] * r[i+lag]
		}
		c := acc / norm
		if c > best {
			best, bestLag = c, lag
		}
	}
	return bestLag, best
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func clampDeg(x float64) float64 {
	if x < -90 {
		return -90
	}
	if x > 90 {
		return 90
	}
	return x
}
