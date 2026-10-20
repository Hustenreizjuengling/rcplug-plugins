package desk

// Exported node-type names, so pages read and write the desk in its own
// spelling (e.g. desk.ChannelNode, "channelOutputMute").
const (
	ChannelNode   = "CHANNEL"
	FaderNode     = "FADER"
	InputNode     = "INPUTSOURCE"
	MeterNode     = "METER"
	OutputNode    = "OUTPUT"
	GUINode       = "GUI"
	SystemNode    = "SYSTEM"
	RecorderNode  = "RECORDER"
	MasterNode    = "MASTERCHANNEL"
	PadNode       = "PAD"
	PadButtonNode = "PADBUTTON"
	NetworkNode   = "NETWORK"
	GuiNode       = "GUI"
	EffectsNode   = "EFFECTS_PARAMETERS"
	MixMinusNode  = "MIXMINUSES"
	DuckerNode    = "DUCKER"
)
