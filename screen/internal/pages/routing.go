package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// Routing: a list of outputs; each opens its mix mode (Main / Mix-minus /
// Custom). The mode is MIXMINUSES[o].outputMixMinus (0/1/2), which the
// console allows.

func init() {
	register(IDRouting, func() Page {
		return &funcPage{title: "ROUTING", build: func(env *Env, r image.Rectangle) ui.View {
			c := &ui.Container{}
			c.SetBounds(r)
			outs := env.Desk.Outputs()
			n := env.Desk.Count(desk.MixMinusNode)
			var rows []ui.View
			for o := 0; o < len(outs); o++ {
				if o >= n {
					break
				}
				o := o
				label := outs[o].Label
				row := ui.NewButton(label, func() { env.Nav.Push(newRoutingMix(o, label)) })
				row.Icon = "IconRouting"
				rows = append(rows, row)
			}
			list := ui.NewScrollList(64, 8)
			list.SetBounds(r)
			// Size each row to full width by wrapping in bounds on layout.
			for _, rw := range rows {
				if b, ok := rw.(*ui.Button); ok {
					_ = b
				}
			}
			list.SetRows(rows)
			c.Add(list)
			return c
		}}
	})
}

func newRoutingMix(output int, label string) Page {
	return newRadioPage(label, desk.MixMinusNode, output, "outputMixMinus", []radioSpec{
		{title: "MAIN MIX", desc: "ALL CHANNELS WILL BE INCLUDED", value: 0},
		{title: "MIX MINUS", desc: "", value: 1},
		{title: "CUSTOM", desc: "", value: 2},
	})
}
