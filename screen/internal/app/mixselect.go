package app

// The mix selection: while an output's routing mix page is on the stack the
// desk learns its output (SYSTEM.systemMixSelected), as RØDE's
// RoutingMixPage tells it; 13 when none is open.

// mixPage is a page that edits one output's mix.
type mixPage interface {
	MixOutput() int
}

// mixOpen is the output whose mix page is open, -1 for none.
func (a *App) mixOpen() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i := len(a.stack) - 1; i >= 0; i-- {
		if mp, ok := a.stack[i].(mixPage); ok {
			return mp.MixOutput()
		}
	}
	return -1
}
