# Signaling messages

**Target:** `internal/protocol/messages.go`

Two families: **participant** (`:1234/api/v1`) and **host** (`:1235/api/v1`). WebSocket path on both: `/api/v1/ws`.

## Participant messages

```go
type ParticipantMsg struct {
	Type string `json:"type"`
	// discriminated union via Type
}

// Inbound (browser → server)
// join, leave, offer, answer, ice-candidate

// Outbound (server → browser)
// welcome, participant-view, error
```

```go
type WelcomeMsg struct {
	Type     string              `json:"type"` // "welcome"
	ClientID string              `json:"clientId"`
	View     ParticipantRoomView `json:"view"`
}

type ParticipantViewMsg struct {
	Type string              `json:"type"` // "participant-view"
	View ParticipantRoomView `json:"view"`
}

type JoinMsg struct {
	Type     string `json:"type"` // "join"
	Name     string `json:"name"`
	HasVideo bool   `json:"hasVideo"`
	HasAudio bool   `json:"hasAudio"`
}
```

WebRTC SDP/ICE messages match prior shape (`from`, `to`, `sdp`, `candidate`) — relayed to Pion hub.

## Host messages

```go
type HostStateMsg struct {
	Type  string    `json:"type"` // "host-state"
	State RoomState `json:"state"`
}

type ConfigMsg struct {
	Type   string                 `json:"type"` // "config"
	Config map[string]interface{} `json:"config"` // Partial HostConfig
}

type ListCaptureDevicesMsg struct {
	Type string `json:"type"` // "list-capture-devices"
}

type CaptureDevicesMsg struct {
	Type    string         `json:"type"` // "capture-devices"
	Devices CaptureDevices `json:"devices"`
}

type SetCaptureDevicesMsg struct {
	Type      string           `json:"type"` // "set-capture-devices"
	Selection CaptureSelection `json:"selection"`
}

type CaptureDevicesUpdatedMsg struct {
	Type  string       `json:"type"` // "capture-devices-updated"
	State CaptureState `json:"capture"`
	Error string       `json:"error,omitempty"`
}

type ParticipantURLMsg struct {
	Type string `json:"type"` // "participant-url"
	URL  string `json:"url"`
}

type SetStreamProcessingMsg struct {
	Type          string                `json:"type"` // "set-stream-processing"
	ParticipantID string                `json:"participantId"` // "host" for host mic
	Flags         StreamProcessingFlags `json:"flags"`
}

// Preview socket (GET /api/v1/ws/preview) — see architecture/preview.md
type PreviewStreamInitMsg struct {
	Type   string `json:"type"` // "preview-stream-init"
	Codec  string `json:"codec"` // "avc1.42E01E"
	Width  int    `json:"width"`
	Height int    `json:"height"`
	FPS    int    `json:"fps"` // 15
}

type PreviewCutMsg struct {
	Type          string `json:"type"` // "preview-cut"
	ActiveVideoID string `json:"activeVideoId"`
	Seq           uint64 `json:"seq"`
}

// Binary preview media: [flags u8][pts_us u64 BE][nal_len u32 BE][h264...]
// flags bit0 = keyframe. Not JSON.
```

## REST

Full route table: [API.md](../API.md).

## WebSocket message rates

| Message | Port | Path | Direction | Rate |
|---------|------|------|-----------|------|
| `host-state` | 1235 | `/api/v1/ws` | → host UI | 50 Hz |
| `preview-stream-init` | 1235 | `/api/v1/ws/preview` | → host UI | once per connect |
| `preview-cut` | 1235 | `/api/v1/ws/preview` | → host UI | on `activeVideoId` change |
| preview H.264 binary | 1235 | `/api/v1/ws/preview` | → host UI | 15 fps |
| `config` | 1235 | `/api/v1/ws` or REST | ← host UI | on change |
| `set-stream-processing` | 1235 | `/api/v1/ws` or REST | ← host UI | per-card toggle |
| `list-capture-devices` | 1235 | `/api/v1/ws` | ← host UI | settings open |
| `capture-devices` | 1235 | `/api/v1/ws` or REST | → host UI | response |
| `set-capture-devices` | 1235 | `/api/v1/ws` or REST | ← host UI | device save |
| `capture-devices-updated` | 1235 | `/api/v1/ws` | → host UI | after reopen |
| `copy-participant-url` | 1235 | `/api/v1/ws` | ← host UI | on click → `participant-url` |
| `participant-view` | 1234 | `/api/v1/ws` | → participant | on change, max 10 Hz |
| `offer/answer/ice` | 1234 | `/api/v1/ws` | ↔ participant | on demand |

Host UI **never** sends `metrics` or `selection` — daemon owns the engine.

`loopDelay` and `globalLatencyMs` inside `host-state` / `participant-view` update on **`LoopDelayPublishMs`** (~3 s), not every 50 Hz tick. Meters and scores still update at full rate.

## Validation

```go
func DecodeParticipantMessage(data []byte) (ParticipantMsg, error)
func DecodeHostMessage(data []byte) (HostMsg, error)
```
