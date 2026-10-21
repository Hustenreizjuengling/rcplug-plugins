package pages

import "github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"

// The assignable meter and the wireless pairing follow the pages as RØDE's
// screen program drives them: the channel page's gain panel puts the input
// on the assignable meter (SYSTEM.assignableMeterSource 0) whenever it is
// shown or hidden; the compressor, de-esser, noise gate and panning pages
// put their own reading on it when they come on screen (1, 2, 3, 4); the
// other processor pages leave it. A wireless panel that goes off screen
// while its slot is pairing cancels the pairing (RADIO.radioPair -1).

// Enter puts the input on the assignable meter (RØDE's gain panel, 0x729ed0).
func (p *ChannelPage) Enter(env *Env) {
	env.Desk.SetMeterSource(desk.MeterSourceInput)
}

// Leave repeats the gain panel's write and cancels a pairing the panel
// started.
func (p *ChannelPage) Leave(env *Env) {
	env.Desk.SetMeterSource(desk.MeterSourceInput)
	if env.Desk.SourceKindOf(env.Desk.SourceOf(p.strip)) == desk.KindWireless {
		env.Desk.CancelPairing()
	}
}

// meterSourceOf is the reading a processor page puts on the assignable
// meter when it comes on screen (-1: the page leaves it).
func meterSourceOf(idx int) int64 {
	switch idx {
	case procComp:
		return desk.MeterSourceCompressor
	case procDeesser:
		return desk.MeterSourceDeesser
	case procGate:
		return desk.MeterSourceNoiseGate
	case procPan:
		return desk.MeterSourcePan
	}
	return -1
}

// Enter puts the processor's reading on the assignable meter (a channel's
// processor pages; the master compellor and the pads have none).
func (p *RackPage) Enter(env *Env) {
	if p.node != desk.ChannelNode || p.isPad {
		return
	}
	if v := meterSourceOf(p.idx); v >= 0 {
		env.Desk.SetMeterSource(v)
	}
}

// Leave hands the meter back to the input: the channel page's gain panel
// shows again under the processor page and writes 0.
func (p *RackPage) Leave(env *Env) {
	if p.node != desk.ChannelNode || p.isPad {
		return
	}
	if meterSourceOf(p.idx) >= 0 {
		env.Desk.SetMeterSource(desk.MeterSourceInput)
	}
}
