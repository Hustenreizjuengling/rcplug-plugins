package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// Shows: the current show's name, the recordings, and a way to RØDE's own
// screen for the rest (new, load, import and export change files the
// console does not touch).

func init() {
	register(IDShows, func() Page {
		return &funcPage{title: "SHOWS", build: func(env *Env, r image.Rectangle) ui.View {
			c := &ui.Container{}
			c.SetBounds(r)
			d := env.Desk
			name := func() string {
				if s := d.Str("CURRENTSHOW", 0, "currentShowName"); s != "" {
					return s
				}
				return "—"
			}
			infoCard(env, c, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+100), [][2]func() string{
				{func() string { return env.T("CURRENT SHOW") }, name},
			})
			tiles := []tile{
				{key: "RECORDINGS", icon: "IconRecordings", open: IDRecordings},
			}
			cells := tileGrid(image.Rect(r.Min.X, r.Min.Y+120, r.Max.X, r.Max.Y-150), len(tiles), 3, 190, 24)
			for i, t := range tiles {
				t := t
				b := ui.NewTile(env.T(t.key), t.icon, func() { env.Nav.Open(t.open) })
				b.SetBounds(cells[i])
				c.Add(b)
			}
			c.Add(noteCard(env, image.Rect(r.Min.X, r.Max.Y-130, r.Max.X-480, r.Max.Y),
				env.TR("", "Neue Show, Laden, Import und Export: auf RØDEs eigenem Bildschirm.", "New show, load, import and export: on RØDE's own screen."))...)
			c.Add(rodeButton(env, image.Rect(r.Max.X-440, r.Max.Y-130, r.Max.X, r.Max.Y)))
			return c
		}}
	})
}

// newBluetoothOutputPage sets the Bluetooth output's level and mutes.
func newBluetoothOutputPage() Page {
	return &funcPage{title: "BLUETOOTH", build: func(env *Env, r image.Rectangle) ui.View {
		c := &ui.Container{}
		c.SetBounds(r)
		th := env.Ctx.Theme
		d := env.Desk
		card := ui.NewCard()
		card.SetBounds(image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+124))
		sl := ui.NewSlider(func() float64 { return d.Num(desk.OutputNode, 0, "outputBTLevel") },
			func(v float64) { d.SetFloat(desk.OutputNode, 0, "outputBTLevel", v) })
		sl.Label = env.T("LEVEL")
		sl.Text = pct
		sl.SetBounds(ui.Inset(card.Bounds(), 16))
		c.Add(card, sl)
		rows := []toggleRow{
			{key: "MUTE", node: desk.OutputNode, prop: "outputBTMute"},
			{key: "AUTO-MUTE BLUETOOTH OUTPUT", node: desk.OutputNode, prop: "outputBTAutoMute"},
		}
		y := r.Min.Y + 124 + th.Gap
		for _, row := range rows {
			row := row
			rr := image.Rect(r.Min.X, y, r.Max.X, y+th.Row)
			cd := ui.NewCard()
			cd.SetBounds(rr)
			tog := ui.NewToggle(func() bool { return d.Bool(row.node, 0, row.prop) },
				func(on bool) { d.SetBool(row.node, 0, row.prop, on) })
			tog.Label = env.TR(row.key, "Stumm", "Mute")
			if row.key != "MUTE" {
				tog.Label = env.T(row.key)
			}
			tog.SetBounds(ui.Inset(rr, 14))
			c.Add(cd, tog)
			y += th.Row + th.Gap
		}
		return c
	}}
}
