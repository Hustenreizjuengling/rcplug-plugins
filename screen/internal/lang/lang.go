// Package lang gives the screens their text in the device's own words. The
// device's screen program carries RØDE's translations (English keys mapped
// to each language); this package looks a key up in the chosen language and
// falls back to the key itself (which is the English text) when there is no
// translation. So the UI reads exactly as RØDE's does, without shipping any
// of RØDE's text in our repository.
package lang

import "strings"

// Texts is a set of labels in one language.
type Texts struct {
	code  string
	table map[string]string
}

// New makes a Texts for a language code over a key→text table (as read from
// the device). A nil table is fine: everything falls back to its key.
func New(code string, table map[string]string) *Texts {
	if table == nil {
		table = map[string]string{}
	}
	return &Texts{code: code, table: table}
}

// Code is the language ("en", "de", …).
func (t *Texts) Code() string { return t.code }

// German reports whether the language is German.
func (t *Texts) German() bool { return t.code == "de" }

// T returns the text for an English key in the chosen language, or the key
// itself when there is no translation.
func (t *Texts) T(key string) string {
	if t == nil {
		return key
	}
	if s, ok := t.table[key]; ok && s != "" {
		return s
	}
	return key
}

// Title is T with the first letter of each word kept as the device shows
// headers (many RØDE keys are already upper case); it is a convenience for
// code that wants a readable caption without assuming the case.
func (t *Texts) Title(key string) string { return t.T(key) }

// Has reports whether a translation exists for the key.
func (t *Texts) Has(key string) bool {
	_, ok := t.table[key]
	return ok
}

// Count is how many translations are loaded.
func (t *Texts) Count() int { return len(t.table) }

// Upper upper-cases a label (for the few captions the UI shows in caps when a
// translation is title-case).
func Upper(s string) string { return strings.ToUpper(s) }
