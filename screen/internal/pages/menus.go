package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// A menu is a grid of large icon tiles. Each tile opens a page by id or
// pushes one built on the spot.

type tile struct {
	key  string
	icon string
	open string      // an id for Nav.Open, or ""
	push func() Page // a page to push, or nil
}

type menuPage struct {
	title string
	tiles []tile
	cols  int
}

func (m *menuPage) Title() string { return m.title }

func (m *menuPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	cols := m.cols
	if cols == 0 {
		cols = 3
	}
	rows := (len(m.tiles) + cols - 1) / cols
	if rows < 1 {
		rows = 1
	}
	// Tiles are at most ~150 tall; keep them from stretching too far.
	area := r
	if h := rows * 160; h < area.Dy() {
		area.Max.Y = area.Min.Y + h
	}
	cells := grid(area, cols, rows, 12)
	for i, t := range m.tiles {
		if i >= len(cells) {
			break
		}
		t := t
		b := menuTile(env.T(t.key), t.icon, func() {
			switch {
			case t.push != nil:
				env.Nav.Push(t.push())
			case t.open != "":
				env.Nav.Open(t.open)
			}
		})
		b.SetBounds(cells[i])
		c.Add(b)
	}
	return c
}

func init() {
	register(IDSettings, func() Page {
		return &menuPage{title: "SETTINGS", cols: 4, tiles: []tile{
			{key: "FADERS", icon: "IconFaders", push: func() Page { return newFadersPage() }},
			{key: "OUTPUTS", icon: "IconHeadphones", open: IDOutputs},
			{key: "SMART PADS", icon: "IconSmartPad", open: IDPads},
			{key: "DISPLAY", icon: "IconDisplay", open: IDDisplay},
			{key: "SHOWS", icon: "IconShows", open: IDShows},
			{key: "SYSTEM", icon: "IconSystem", open: IDSystem},
			{key: "ROUTING", icon: "IconRouting", open: IDRouting},
			{key: "rcplug", icon: "IconSystem", open: IDRcplug},
		}}
	})
	register(IDOutputs, func() Page {
		return &menuPage{title: "OUTPUTS", cols: 3, tiles: []tile{
			{key: "HEADPHONES", icon: "IconHeadphones", push: func() Page { return newMonitorPage() }},
			{key: "MONITOR", icon: "IconHeadphones", push: func() Page { return newMonitorPage() }},
			{key: "ROUTING", icon: "IconRouting", open: IDRouting},
			{key: "LISTEN", icon: "IconListen", push: func() Page { return newListenPage() }},
			{key: "MULTITRACK", icon: "IconRecOn", push: func() Page { return newMultitrackPage() }},
			{key: "PROCESSING", icon: "IconSystem", push: func() Page { return newMasterProcPage() }},
		}}
	})
	register(IDDisplay, func() Page {
		return &menuPage{title: "DISPLAY", cols: 2, tiles: []tile{
			{key: "BRIGHTNESS", icon: "IconDisplay", push: func() Page { return newBrightnessPage() }},
			{key: "METERING", icon: "IconDisplay", push: func() Page { return newMeteringPage() }},
			{key: "HAPTICS", icon: "IconDisplay", push: func() Page { return newHapticsPage() }},
			{key: "REC BUTTON", icon: "IconRecOn", push: func() Page { return newRecButtonPage() }},
		}}
	})
	register(IDSystem, func() Page {
		return &menuPage{title: "SYSTEM", cols: 3, tiles: []tile{
			{key: "LANGUAGE", icon: "IconSystem", push: func() Page { return newLanguagePage() }},
			{key: "NETWORK", icon: "IconSystem", open: IDNetwork},
			{key: "BLUETOOTH", icon: "IconBluetooth", open: IDBluetooth},
			{key: "INFORMATION", icon: "IconSystem", open: IDInfo},
			{key: "MIDI", icon: "IconSystem", push: func() Page { return newMidiPage() }},
			{key: "CallMe", icon: "IconCallMe", open: IDCallMe},
		}}
	})
	register(IDShows, func() Page {
		return &menuPage{title: "SHOWS", cols: 3, tiles: []tile{
			{key: "RECORDINGS", icon: "IconRecOn", open: IDRecordings},
		}}
	})
}
