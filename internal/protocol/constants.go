package protocol

const HostStreamID = "host"
const PlaybackRefStreamID = "playback-ref"

type MixerState string

const (
	MixerLocked  MixerState = "LOCKED"
	MixerHold    MixerState = "HOLD"
	MixerSwitch  MixerState = "SWITCH"
	MixerSilence MixerState = "SILENCE"
)

type ClientRole string

const RoleParticipant ClientRole = "participant"

type StreamRole string

const (
	StreamRoleHost        StreamRole = "host"
	StreamRoleParticipant StreamRole = "participant"
	StreamRoleReference   StreamRole = "reference"
)
