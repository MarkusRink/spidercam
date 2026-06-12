# HTTP / WebSocket API

Canonical protocol types: [domain/types.md](./domain/types.md), [domain/messages.md](./domain/messages.md).

Each listener serves the SPA at `/` and the API at `/api` on the **same origin**. Env: `SPIDERCAM_PARTICIPANT_PORT` (default `1234`), `SPIDERCAM_HOST_PORT` (default `1235`).

---

## Overview

| Port | Bind | REST | WebSocket |
|------|------|------|------------|
| **1234** | `0.0.0.0` | `GET /api/health` | `GET /api/v1/ws` — participant signaling |
| **1235** | `127.0.0.1` | see [Host REST](#host-rest-1235) | `GET /api/v1/ws` — control; `GET /api/v1/ws/preview` — H.264 preview |

Participant port never exposes host `RoomState` or preview. Host port never accepts participant `join` / WebRTC relay.

---

## Shared

### `GET /api/health`

Both ports.

**Response** `200 application/json`:

```json
{ "ok": true }
```

```go
type HealthResponse struct {
	OK bool `json:"ok"`
}
```

---

## Host REST (`:1235`)

### `GET /api/v1/host/state`

Latest full room snapshot (same shape as WS `host-state` payload).

**Response** `200` → [`RoomState`](./domain/types.md#roomstate)

### `POST /api/v1/host/config`

Apply partial mixer settings. Merges into session `HostConfig` in RAM.

**Request body** — partial JSON; any subset of fields from [domain/host-config.md](./domain/host-config.md) (`audioHoldMs`, `crossfadeMs`, `referenceDuckDb`, `switchMargin`, `scoreWeights`, …).

**Response** `200`:

```json
{ "ok": true }
```

```go
type ConfigOKResponse struct {
	OK bool `json:"ok"`
}
```

**Response** `400` — validation error:

```json
{ "ok": false, "error": "crossfadeMs out of range" }
```

WS equivalent: inbound `config` message ([domain/messages.md](./domain/messages.md)).

### `GET /api/v1/capture/devices`

Enumerate selectable capture devices.

**Response** `200` → [`CaptureDevices`](./domain/types.md#capturedevices)

### `POST /api/v1/capture/selection`

Set mic, camera, and playback sink; daemon calls `capture.Reopen`.

**Request body** → [`CaptureSelection`](./domain/types.md#captureselection)

**Response** `200` → [`CaptureState`](./domain/types.md#capturestate)

**Response** `500` — reopen failed:

```json
{ "error": "pw_open_mic: ..." }
```

WS equivalent: `set-capture-devices` → `capture-devices-updated`.

### `POST /api/v1/host/stream-processing`

Set per-stream AEC / RNNoise toggles. Session RAM only.

**Request body:**

```json
{
  "participantId": "host",
  "flags": { "aecEnabled": true, "denoiseEnabled": false }
}
```

→ [`SetStreamProcessingMsg`](./domain/messages.md) / [`StreamProcessingFlags`](./domain/types.md).

**Response** `200`: `{ "ok": true }`

WS equivalent: `set-stream-processing`.

---

## Host WebSocket — control (`:1235`)

**URL:** `ws://127.0.0.1:1235/api/v1/ws`

**Direction:** JSON text frames only.

### Server → client

| `type` | Payload | Rate | Type |
|--------|---------|------|------|
| `host-state` | `state: RoomState` | 50 Hz | [`HostStateMsg`](./domain/messages.md) |
| `capture-devices` | `devices: CaptureDevices` | on request | [`CaptureDevicesMsg`](./domain/messages.md) |
| `capture-devices-updated` | `state`, optional `error` | after reopen | [`CaptureDevicesUpdatedMsg`](./domain/messages.md) |
| `participant-url` | `url: string` | on copy | see below |

```go
type ParticipantURLMsg struct {
	Type string `json:"type"` // "participant-url"
	URL  string `json:"url"`
}
```

### Client → server

| `type` | Body | Response |
|--------|------|----------|
| `config` | `config: Partial<HostConfig>` | applies immediately; next `host-state` reflects change |
| `list-capture-devices` | — | `capture-devices` |
| `set-capture-devices` | `selection: CaptureSelection` | `capture-devices-updated` |
| `set-stream-processing` | `participantId`, `flags: StreamProcessingFlags` | next `host-state` |
| `copy-participant-url` | — | `participant-url` |

Host UI **never** sends metrics, selection, or preview data — daemon owns the engine.

Slow fields inside `host-state`: `loopDelay`, `globalLatencyMs` publish ~3 s ([domain/messages.md](./domain/messages.md)).

---

## Host WebSocket — preview (`:1235`)

**URL:** `ws://127.0.0.1:1235/api/v1/ws/preview`

**Direction:** JSON control + binary H.264 chunks. See [architecture/preview.md](./architecture/preview.md).

### Server → client (on connect)

```go
type PreviewStreamInitMsg struct {
	Type   string `json:"type"` // "preview-stream-init"
	Codec  string `json:"codec"` // "avc1.42E01E"
	Width  int    `json:"width"`
	Height int    `json:"height"`
	FPS    int    `json:"fps"` // 15
}
```

### Server → client (on `activeVideoId` change)

```go
type PreviewCutMsg struct {
	Type          string `json:"type"` // "preview-cut"
	ActiveVideoID string `json:"activeVideoId"`
	Seq           uint64 `json:"seq"`
}
```

### Server → client (media @ ~15 fps)

Binary frame — not JSON:

```text
[flags: u8][pts_us: u64 BE][nal_len: u32 BE][h264_au: nal_len bytes]
```

| Flag bit | Meaning |
|----------|---------|
| `0x01` | keyframe (IDR) |

### Client → server

None in v1. Preview socket is receive-only.

---

## Participant REST (`:1234`)

### `GET /api/health`

Same as [shared health](#get-apihealth).

No other REST routes on `:1234` in v1.

---

## Participant WebSocket (`:1234`)

**URL:** `ws://<host>:1234/api/v1/ws`

**Direction:** JSON text frames. WebRTC media uses separate SRTP (Pion), not this socket.

### Server → client

| `type` | Payload | When |
|--------|---------|------|
| `welcome` | `clientId`, `view: ParticipantRoomView` | on connect |
| `participant-view` | `view: ParticipantRoomView` | selection/join/leave, max 10 Hz |
| `error` | `message` | validation / room errors |

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

type ErrorMsg struct {
	Type    string `json:"type"` // "error"
	Message string `json:"message"`
}
```

[`ParticipantRoomView`](./domain/types.md#participantroomview) — slim projection; never includes full `metrics[]` or other participants' transport stats.

### Client → server

| `type` | Body |
|--------|------|
| `join` | `name`, `hasVideo`, `hasAudio` → [`JoinMsg`](./domain/messages.md) |
| `leave` | — |
| `offer` | `sdp` — WebRTC offer |
| `answer` | `sdp` — WebRTC answer (if hub offers) |
| `ice-candidate` | `candidate` — ICE candidate JSON string |

```go
type JoinMsg struct {
	Type     string `json:"type"` // "join"
	Name     string `json:"name"`
	HasVideo bool   `json:"hasVideo"`
	HasAudio bool   `json:"hasAudio"`
}

type LeaveMsg struct {
	Type string `json:"type"` // "leave"
}

type OfferMsg struct {
	Type string `json:"type"` // "offer"
	SDP  string `json:"sdp"`
}

type AnswerMsg struct {
	Type string `json:"type"` // "answer"
	SDP  string `json:"sdp"`
}

type ICECandidateMsg struct {
	Type      string `json:"type"` // "ice-candidate"
	Candidate string `json:"candidate"`
}
```

Relay handled by Pion hub — [architecture/webrtc.md](./architecture/webrtc.md).

---

## Static assets

| Method | Path | Port | Response |
|--------|------|------|----------|
| `GET` | `/*` | 1234 / 1235 | Embedded Solid SPA (`index.html` fallback) |

Vite dev proxies `/api` → daemon; production UIs call same-origin `/api`.

---

## Error conventions

| HTTP | When |
|------|------|
| `400` | malformed JSON, out-of-range config |
| `404` | unknown `/api` path |
| `405` | wrong method |
| `500` | capture reopen / internal failure |

WebSocket: recoverable errors as `error` JSON on participant socket; host control socket logs to daemon stderr for v1 (no `error` type on `:1235` control WS).

---

## Rate summary

| Channel | Rate |
|---------|------|
| `host-state` | 50 Hz |
| Preview H.264 | 15 fps |
| `participant-view` | on change, max 10 Hz |
| Transport stats in `host-state` | 1 Hz |
| Loop delay fields | ~0.3 Hz |

---

## Testing entry points

Go E2E (`spidercamd --mock`) exercises REST + both host WS paths + participant WS. Playwright+MSW mocks these URLs without the daemon — [testing.md](./testing.md).
