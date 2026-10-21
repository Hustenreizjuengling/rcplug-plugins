package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// A menu is a grid of icon tiles with their names underneath, as the
// device's settings pages. Each tile opens a page by id or pushes one built
// on the spot.

type tile struct {
	key  string
	de   string // fallbacks when the device has no text for key
	en   string
	icon string
	open string      // an id for Nav.Open, or ""
	push func() Page // a page to push, or nil
	text string      // a caption inside the tile instead of an icon
}

type menuPage struct {
	title  string
	tiles  []tile
	cols   int
	banner func(env *Env) string // a line above the tiles when not empty
}

func (m *menuPage) Title() string { return m.title }

func (m *menuPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	if m.banner != nil {
		if text := m.banner(env); text != "" {
			th := env.Ctx.Theme
			card := ui.NewPanelView(th.Warn)
			card.Radius = th.Radius
			card.SetBounds(image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+64))
			lbl := ui.NewLabel(text, th.Fonts.Label, th.TextOnLight, ui.AlignCenter)
			lbl.SetBounds(card.Bounds())
			c.Add(card, lbl)
			r.Min.Y += 80
		}
	}
	cols := m.cols
	if cols == 0 {
		cols = 4
	}
	side := 200
	if rows := (len(m.tiles) + cols - 1) / cols; rows*(side+48+24) > r.Dy() {
		side = (r.Dy()-rows*72)/rows - 8
	}
	cells := tileGrid(r, len(m.tiles), cols, side, 24)
	for i, t := range m.tiles {
		t := t
		open := func() {
			switch {
			case t.push != nil:
				env.Nav.Push(t.push())
			case t.open != "":
				env.Nav.Open(t.open)
			}
		}
		label := env.T(t.key)
		if t.de != "" {
			label = env.TR(t.key, t.de, t.en)
		}
		var b *ui.Button
		if t.text != "" {
			b = ui.NewTile(label, "", open)
			b.Sub = t.text
		} else {
			b = ui.NewTile(label, t.icon, open)
		}
		b.SetBounds(cells[i])
		c.Add(b)
	}
	return c
}

func init() {
	register(IDSettings, func() Page {
		return &menuPage{title: "CONFIGURATION", cols: 4, banner: func(env *Env) string {
			// The device's own notice when a firmware update is waiting.
			if env.Desk.Bool(desk.SystemNode, 0, "osUpdateAvailable") || env.Desk.Bool(desk.SystemNode, 0, "appUpdateAvailable") {
				return env.T("FIRMWARE UPDATE AVAILABLE")
			}
			return ""
		}, tiles: []tile{
			// RØDE's MenuConfigurationPage (1.7.6) in its order: Faders,
			// Outputs, Operation Mode, Display, Shows, System; the rcplug
			// tile is ours, at the end.
			{key: "FADERS", icon: "IconInputs", push: func() Page { return newFadersPage() }},
			{key: "OUTPUTS", icon: "IconOutputs", open: IDOutputs},
			{key: "OPERATION MODE", icon: "Oslash", push: func() Page { return newOperatingModePage() }},
			{key: "DISPLAY", icon: "IconDisplay", open: IDDisplay},
			{key: "SHOWS", icon: "IconShows", open: IDShows},
			{key: "SYSTEM", icon: "IconSystem", open: IDSystem},
			{key: "rcplug", icon: "IconRodeCentral", open: IDRcplug},
		}}
	})
	register(IDOutputs, func() Page {
		return &menuPage{title: "OUTPUTS", cols: 3, tiles: []tile{
			{key: "HEADPHONES", icon: "IconHeadphones", push: func() Page { return newHeadphonesPage() }},
			{key: "MONITOR", icon: "IconMonitor", push: func() Page { return newMonitorPage() }},
			{key: "ROUTING", icon: "IconRouting", open: IDRouting},
			{key: "LISTEN", icon: "IconListen", push: func() Page { return newListenPage() }},
			{key: "MULTITRACK", icon: "IconMultitrack", push: func() Page { return newMultitrackPage() }},
			{key: "PROCESSING", icon: "IconProcessingNavigation", push: func() Page { return NewMasterRack() }},
		}}
	})
	register(IDDisplay, func() Page {
		return &menuPage{title: "DISPLAY", cols: 4, tiles: []tile{
			{key: "BRIGHTNESS", icon: "IconBrightness", push: func() Page { return newBrightnessPage() }},
			{key: "METERING", icon: "IconMetering", push: func() Page { return newMeteringPage() }},
			{key: "HAPTICS", icon: "IconHaptics", push: func() Page { return newHapticsPage() }},
			{key: "REC BUTTON", icon: "IconRecordingSettings", push: func() Page { return newRecButtonPage() }},
		}}
	})
	register(IDSystem, func() Page {
		return &menuPage{title: "SYSTEM", cols: 4, tiles: []tile{
			{key: "LANGUAGE", icon: "IconLanguage", push: func() Page { return newLanguagePage() }},
			{key: "DATE & TIME", icon: "IconDateTime", push: func() Page { return newDateTimePage() }},
			{key: "NETWORK", icon: "IconNetwork", open: IDNetwork},
			{key: "BLUETOOTH", icon: "bluetooth", open: IDBluetooth},
			{key: "INFORMATION", icon: "IconInformation", open: IDInfo},
			{key: "MIDI", icon: "smart_pads_trigger", push: func() Page { return newMidiPage() }},
			{key: "CLOUD CONNECTION", de: "Cloud-Verbindung", en: "Cloud Connection", icon: "cloud_outline", push: func() Page { return newCloudPage() }},
			{key: "CallMe", icon: "CallMe", open: IDCallMe},
		}}
	})
}
