// Package desk is the live model of the mixer for the screen UI. It reads
// the desk's state through the console plugin's HTTP API (the full tree, the
// semantic summary and a stream of changes) and writes single properties
// back. The console holds the single control connection to the mixer; this
// package is one more surface on top of it, like the web console.
//
// The model is safe for use from the drawing goroutine while the event
// stream updates it: reads take a read lock, the stream and writes take the
// write lock. Writes are applied to the local copy at once (the mixer does
// not echo a client's own writes) and sent to the console in the background.
package desk

import (
	"encoding/json"
	"strconv"
)

// Value is a decoded property value (a juce var as the console sends it).
// Numbers keep both an integer and a floating form so the UI can read either
// without caring which the desk uses.
type Value struct {
	Kind byte // i l b d s x a v u r (best effort; see the console's TypeCode)
	I    int64
	F    float64
	S    string
	Bool bool
}

// Num is the value as a number (0 for non-numbers).
func (v Value) Num() float64 {
	switch v.Kind {
	case 'd':
		return v.F
	case 'i', 'l':
		return float64(v.I)
	case 'b':
		if v.Bool {
			return 1
		}
	}
	return 0
}

// Int is the value as a whole number.
func (v Value) Int() int64 {
	switch v.Kind {
	case 'i', 'l':
		return v.I
	case 'd':
		return int64(v.F)
	case 'b':
		if v.Bool {
			return 1
		}
	}
	return 0
}

// Truth is the value as a boolean.
func (v Value) Truth() bool {
	switch v.Kind {
	case 'b':
		return v.Bool
	case 'i', 'l':
		return v.I != 0
	case 'd':
		return v.F != 0
	}
	return false
}

// Str is the value as text.
func (v Value) Str() string { return v.S }

// String renders the value for logs.
func (v Value) String() string {
	switch v.Kind {
	case 'b':
		return strconv.FormatBool(v.Bool)
	case 'd':
		return strconv.FormatFloat(v.F, 'g', -1, 64)
	case 's', 'x', 'r':
		return v.S
	case 'v', 'u', 0:
		return ""
	default:
		return strconv.FormatInt(v.I, 10)
	}
}

// Bool makes a boolean value.
func Bool(b bool) Value { return Value{Kind: 'b', Bool: b} }

// Int64 makes an integer value.
func Int64(i int64) Value { return Value{Kind: 'l', I: i, F: float64(i)} }

// Float makes a floating value.
func Float(f float64) Value { return Value{Kind: 'd', F: f, I: int64(f)} }

// Text makes a string value.
func Text(s string) Value { return Value{Kind: 's', S: s} }

// fromJSON builds a value from a decoded JSON value (numbers arrive as
// json.Number when the decoder uses numbers) and an optional type letter.
func fromJSON(x interface{}, code byte) Value {
	v := Value{Kind: code}
	switch t := x.(type) {
	case bool:
		v.Kind, v.Bool = 'b', t
	case float64:
		v.F, v.I = t, int64(t)
		if v.Kind == 0 {
			v.Kind = 'd'
		}
	case json.Number:
		f, _ := t.Float64()
		i, err := t.Int64()
		v.F = f
		if err == nil {
			v.I = i
		} else {
			v.I = int64(f)
		}
		if v.Kind == 0 {
			if err == nil {
				v.Kind = 'l'
			} else {
				v.Kind = 'd'
			}
		}
	case string:
		v.S = t
		if v.Kind == 0 {
			v.Kind = 's'
		}
	case nil:
		if v.Kind == 0 {
			v.Kind = 'v'
		}
	default:
		if v.Kind == 0 {
			v.Kind = 'u'
		}
	}
	// A type letter that says number but a string arrived (hex binary) keeps
	// the string; numbers already filled I and F above.
	switch v.Kind {
	case 'i', 'l':
		if v.F != 0 && v.I == 0 {
			v.I = int64(v.F)
		}
	}
	return v
}
