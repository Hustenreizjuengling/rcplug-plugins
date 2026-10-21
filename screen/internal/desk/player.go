package desk

import "time"

// The recording player (PLAYER) plays a recording back on the podcast
// player page: the file, its state, position and speed, the trim and fade
// envelope of a non-destructive edit, and a jump request signal.

// PlayerNode is the player's node type.
const PlayerNode = "PLAYER"

// Player is the player's state.
type Player struct {
	File     string
	Size     float64
	State    int64
	Speed    int64
	Position time.Duration
	Progress float64 // 0…1
	Env      [4]float64
}

// PlayerEnvProps are the player's envelope properties in drawing order:
// trim start, fade-in end, fade-out start, trim end.
var PlayerEnvProps = [4]string{"playerEnvStart", "playerEnvFadeIn", "playerEnvFadeOut", "playerEnvStop"}

// Player reads the player.
func (m *Model) Player() Player {
	p := Player{
		File:     m.Str(PlayerNode, 0, "playerFilePath"),
		Size:     m.Num(PlayerNode, 0, "playerFileSize"),
		State:    m.Int(PlayerNode, 0, "playerState"),
		Speed:    m.Int(PlayerNode, 0, "playerSpeed"),
		Position: time.Duration(m.Num(PlayerNode, 0, "playerCurrentPositionTime") * float64(time.Second)),
		Progress: m.Num(PlayerNode, 0, "playerProgress"),
	}
	for i, prop := range PlayerEnvProps {
		p.Env[i] = m.Num(PlayerNode, 0, prop)
	}
	return p
}

// PlayerLoad names the file the player should play.
func (m *Model) PlayerLoad(path string) { m.SetStr(PlayerNode, 0, "playerFilePath", path) }

// PlayerSetState asks for a player state.
func (m *Model) PlayerSetState(s int64) { m.SetInt(PlayerNode, 0, "playerState", s) }

// PlayerSetSpeed sets the playback speed index.
func (m *Model) PlayerSetSpeed(s int64) { m.SetInt(PlayerNode, 0, "playerSpeed", s) }

// PlayerJump asks the player to jump to a sample (playerJumpToSample
// [int32, bool]).
func (m *Model) PlayerJump(sample int64) bool {
	return m.SendSignal(PlayerNode, 0, "playerJumpToSample", signalInt(sample))
}

// PlayerSetEnv writes one envelope value.
func (m *Model) PlayerSetEnv(prop string, v float64) { m.SetFloat(PlayerNode, 0, prop, v) }
