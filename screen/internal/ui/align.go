package ui

import "github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"

// Align values re-exported so callers need not import the font package.
type Align = font.Align

const (
	AlignLeft   = font.Left
	AlignCenter = font.Center
	AlignRight  = font.Right
)
