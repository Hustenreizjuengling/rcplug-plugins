package pages

import (
	"math"
	"strconv"
)

// The desk stores every processing parameter as 0…1; RØDE's program turns
// a value into its unit with one of a few parameter classes (read out of
// rc_gui_juce 1.7.6 and already used by the rcplug console):
//
//	frequency, ratio   lo + (hi − lo) · v²          FrequencyProperty, RatioProperty
//	gain, threshold    lo + (hi − lo) · v, 0.01 steps  GainProperty
//	Q and the rest     lo + (hi − lo) · v           ParameterProperty
//	times, de-esser f  lo · (hi / lo)^v             Attack-, Release-, DeesserFrequencyProperty
//	slope              6 … 24 dB/oct, rounded       SlopeProperty
//	gate range         100 · √(1 − v), rounded      GateRangeProperty
//
// The screen shows them as "70.0Hz", "-20.0dB", "2.0:1", "0.7ms", "12dB/Oct".

type scale func(v float64) float64

func quad(lo, hi float64) scale {
	return func(v float64) float64 { return lo + (hi-lo)*clamp01f(v)*clamp01f(v) }
}
func linear(lo, hi float64) scale {
	return func(v float64) float64 { return lo + (hi-lo)*clamp01f(v) }
}
func gainScale(lo, hi float64) scale {
	return func(v float64) float64 { return lo + (hi-lo)*math.Round(clamp01f(v)*100)/100 }
}
func expo(lo, hi float64) scale {
	return func(v float64) float64 { return lo * math.Pow(hi/lo, clamp01f(v)) }
}
func slopeScale(v float64) float64 { return math.Round(6 + 18*clamp01f(v)) }
func rangeScale(v float64) float64 { return math.Round(100 * math.Sqrt(1-clamp01f(v))) }

// invQuad is quad's inverse (a frequency back to the stored value).
func invQuad(lo, hi float64) func(x float64) float64 {
	return func(x float64) float64 { return clamp01f(math.Sqrt((x - lo) / (hi - lo))) }
}

func clamp01f(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// --- the units as the screen prints them ---

func hzText(f float64) string    { return strconv.FormatFloat(f, 'f', 1, 64) + "Hz" }
func db1Text(x float64) string   { return strconv.FormatFloat(x, 'f', 1, 64) + "dB" }
func ratioText(r float64) string { return strconv.FormatFloat(r, 'f', 1, 64) + ":1" }
func msText(t float64) string    { return strconv.FormatFloat(t, 'f', 1, 64) + "ms" }
func slopeText(s float64) string {
	return strconv.Itoa(int(s)) + "dB/Oct"
}
func rangeText(r float64) string { return strconv.FormatFloat(r, 'f', 1, 64) + "dB" }
func qValText(q float64) string  { return strconv.FormatFloat(q, 'f', 2, 64) }
func pctText(v float64) string   { return strconv.Itoa(int(v*100+0.5)) + "%" }

// show composes a scale with a unit printer.
func show(s scale, unit func(float64) string) func(float64) string {
	return func(v float64) string { return unit(s(v)) }
}

// panText is the panning value: "L:100%", "C", "R:40%".
func panText(v float64) string {
	d := int(math.Round((v - 0.5) * 200))
	switch {
	case d == 0:
		return "C"
	case d < 0:
		return "L:" + strconv.Itoa(-d) + "%"
	}
	return "R:" + strconv.Itoa(d) + "%"
}

// Scales per property, for the graphs and the knobs.
var (
	scEqLow      = quad(20, 200)
	scEqMid      = quad(200, 2000)
	scEqHigh     = quad(2000, 12000)
	scEqGain     = gainScale(-12, 12)
	scQ          = linear(0.1, 10)
	scHpf        = quad(20, 200)
	scThreshold  = gainScale(-60, 0)
	scCompRatio  = quad(1.5, 4.5)
	scMakeup     = gainScale(0, 9)
	scCompAtt    = expo(0.1, 10)
	scCompRel    = expo(5, 200)
	scDeessRatio = quad(1, 10)
	scDeessAtt   = expo(0.1, 10)
	scDeessRel   = expo(50, 2000)
	scDeessFreq  = expo(1000, 8000)
	scGateAtt    = expo(0.1, 1000)
	scGateHold   = expo(50, 2000)
	scGateRel    = expo(50, 2000)
	scHyst       = linear(0, 30)
	scAETune     = quad(600, 5000)
	scBBTune     = quad(60, 312)
)
