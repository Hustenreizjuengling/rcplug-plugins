package desk

import (
	"context"
	"time"
)

// SYSTEM.systemMixSelected tells the desk which output's mix is being
// edited. RØDE's RoutingMixPage writes the output's MIXMINUSES index when it
// comes on screen (its page-shown handler 0x7a9b30 calls 0x7a4970 with the
// page's output) and 13 when it goes (0x7a5850, from the page's vtable),
// whatever the desk's number of outputs; nothing else writes it. The mixer
// uses it for the encoder and the custom mix's meters.

// MixSelectedNone is the value RØDE's screen writes when no mix page is open.
const MixSelectedNone int64 = 13

// SelectMix tells the desk which output's mix page is open (its MIXMINUSES
// index; -1 for none).
func (m *Model) SelectMix(output int) {
	v := MixSelectedNone
	if output >= 0 {
		v = int64(output)
	}
	if !m.Has(SystemNode, 0, "systemMixSelected") || m.Int(SystemNode, 0, "systemMixSelected") == v {
		return
	}
	m.SetInt(SystemNode, 0, "systemMixSelected", v)
}

// ReleaseMix clears the mix selection at once, for the end of the
// interface: the write goes out before the process ends.
func (m *Model) ReleaseMix() {
	if !m.Has(SystemNode, 0, "systemMixSelected") || m.Int(SystemNode, 0, "systemMixSelected") == MixSelectedNone {
		return
	}
	path, epoch, ok := m.writeLocal(SystemNode, 0, "systemMixSelected", Int64(MixSelectedNone))
	if !ok || m.offline {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := m.conn.Set(ctx, epoch, path, "systemMixSelected", MixSelectedNone); err != nil {
		m.logf("screen: write systemMixSelected=%d: %v", MixSelectedNone, err)
	}
}
