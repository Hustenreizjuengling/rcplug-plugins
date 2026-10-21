package app_test

import (
	"testing"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/lang"
)

// The interface follows the desk's language setting: a start in the
// fallback language switches once the desk's setting is known, and a change
// on the language page applies at once.
func TestFollowLanguage(t *testing.T) {
	a, model := newTestApp(t) // texts start as "de", the fixture's GUI.lang is "de"
	var loaded []string
	a.FollowLanguage(func(code string) *lang.Texts {
		loaded = append(loaded, code)
		return lang.New(code, nil)
	})
	a.RenderOnce()
	if len(loaded) != 0 {
		t.Fatalf("same language, yet texts were loaded: %v", loaded)
	}
	model.SetStr(desk.GUINode, 0, "lang", "en")
	a.RenderOnce()
	if a.Texts().Code() != "en" || len(loaded) != 1 {
		t.Fatalf("after GUI.lang=en the texts are %q (loaded %v), want en once", a.Texts().Code(), loaded)
	}
	a.RenderOnce()
	if len(loaded) != 1 {
		t.Fatalf("texts loaded again without a change: %v", loaded)
	}
	model.SetStr(desk.GUINode, 0, "lang", "de")
	a.RenderOnce()
	if a.Texts().Code() != "de" {
		t.Fatalf("after GUI.lang=de the texts are %q", a.Texts().Code())
	}
}
