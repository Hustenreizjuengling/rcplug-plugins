package desk

// CancelPairing writes RADIO.radioPair = -1, as RØDE's wireless panel does
// when it goes off screen while its slot is pairing (its hide handler
// 0x74c2b0). The desk itself sets -1 when a pairing ends.
func (m *Model) CancelPairing() {
	if !m.Pairing() {
		return
	}
	m.SetInt(RadioNode, 0, "radioPair", -1)
}
