# HTTP / WebSocket API

Canonical protocol types: `internal/protocol/` (Go), `web/protocol/` (TypeScript).

Each listener serves the SPA at `/` and the API at `/api` on the **same origin**. Default ports: participant **1234**, host **1235** (override via `--participant-addr` / `--host-addr`).

---

## Overview

| Port | Bind | REST | WebSocket |
| ---- | ---- | ---- | --------- |
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

---

## Host REST (`:1235`)

### `GET /api/v1/host/state`

Latest full room snapshot (same shape as WS `host-state` payload).

**Response** `200` → `RoomState` (`internal/protocol/types.go`)

### `POST /api/v1/host/config`

Apply partial mixer settings. Merges into session `HostConfig` in RAM.

**Request body** — partial JSON; any subset of fields from `HostConfig` / `HostConfigPatch` in `internal/protocol/config.go` (`audioHoldMs`, `crossfadeMs`, `referenceDuckDb`, `switchMargin`, `scoreWeights`, …).

**Response** `200`:

```json
{ "ok": true }
```

**Response** `400` — validation error:

```json
{ "ok": false, "error": "crossfadeMs out of range" }
```

WS equivalent: inbound `config` message.

### `GET /api/v1/capture/devices`

Enumerate selectable capture devices.

**Response** `200` → `CaptureDevices`

Native enumeration via `capture.ListDevices()`. Fixture device lists are used only in `--mock` mode (dev/CI).

On startup and on `list-capture-devices`, the daemon fills any missing mic/camera/sink IDs from the first available device in each category (after env-var bootstrap defaults, if set).

### `POST /api/v1/capture/selection`

Patch mic, camera, and/or playback sink. All fields in `CaptureSelection` are optional; omitted fields keep the current selection (or the first available device if the stored ID is no longer listed).

**Request body** → partial `CaptureSelection` (`micId`, `cameraId`, and/or `sinkId`)

**Response** `200` → `CaptureState`

**Response** `400` — unknown device id in the patch:

```json
{ "error": "unknown device id" }
```

WS equivalent: `set-capture-devices` → `capture-devices-updated` (also sent when `list-capture-devices` applies default selection).

### `POST /api/v1/host/stream-processing`

Set per-stream AEC / RNNoise toggles. Session RAM only.

**Request body:**

```json
{
  "participantId": "host",
  "flags": { "aecEnabled": true, "denoiseEnabled": false }
}
```

**Response** `200`: `{ "ok": true }`

WS equivalent: `set-stream-processing`.

---

## Host WebSocket — control (`:1235`)

**URL:** `ws://127.0.0.1:1235/api/v1/ws`

**Direction:** JSON text frames only.

### Server → client

| `type` | Payload | Rate |
| ------ | ------- | ---- |
| `host-state` | `state: RoomState` | 50 Hz |
| `capture-devices` | `devices: CaptureDevices` | on request |
| `capture-devices-updated` | `capture`, optional `error` | after selection |
| `participant-url` | `url: string` | on copy |

### Client → server

| `type` | Body | Response |
| ------ | ---- | -------- |
| `config` | `config: Partial<HostConfig>` | applies immediately; next `host-state` reflects change |
| `list-capture-devices` | — | `capture-devices`; may also send `capture-devices-updated` when defaults are applied |
| `set-capture-devices` | partial `CaptureSelection` | `capture-devices-updated` |
| `set-stream-processing` | `participantId`, `flags: StreamProcessingFlags` | next `host-state` |
| `copy-participant-url` | — | `participant-url` |

Host UI **never** sends metrics, selection, or preview data — daemon owns the engine.

Slow fields inside `host-state`: `loopDelay`, `globalLatencyMs` publish ~3 s.

---

## Host WebSocket — preview (`:1235`)

**URL:** `ws://127.0.0.1:1235/api/v1/ws/preview`

**Direction:** JSON control + binary H.264 chunks.

### Server → client (on connect)

```json
{
  "type": "preview-stream-init",
  "codec": "avc1.42E01E",
  "width": 1280,
  "height": 720,
  "fps": 15
}
```

### Server → client (on `activeVideoId` change)

```json
{
  "type": "preview-cut",
  "activeVideoId": "host",
  "seq": 1
}
```

### Server → client (media @ ~15 fps)

Binary frame — not JSON:

```text
[flags: u8][pts_us: u64 BE][nal_len: u32 BE][h264_au: nal_len bytes]
```

| Flag bit | Meaning |
| -------- | ------- |
| `0x01` | keyframe (IDR) |

Framing implemented in `internal/preview/pack.go` (`PackChunk`, `AnnexBToAVCC`).

### Client → server

None in v1. Preview socket is receive-only. Host UI decodes with WebCodecs (`web/host/src/adapters/preview-stream.ts`).

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
| ------ | ------- | ---- |
| `welcome` | `clientId`, `view: ParticipantRoomView` | on connect |
| `participant-view` | `view: ParticipantRoomView` | selection/join/leave, max 10 Hz |
| `error` | `message` | validation / room errors |
| `answer` | `sdp` | after client `offer` |
| `ice-candidate` | `candidate` | ICE from Pion hub |

`ParticipantRoomView` — slim projection; never includes full `metrics[]` or other participants' transport stats.

### Client → server

| `type` | Body |
| ------ | ---- |
| `join` | `name`, `hasVideo`, `hasAudio` |
| `leave` | — |
| `offer` | `sdp` — WebRTC offer (browser creates after join) |
| `ice-candidate` | `candidate` — ICE candidate JSON string |

WebRTC negotiation: participant **offers**, hub **answers** (D21). `from` / `to` fields on SDP messages are optional and ignored by the current hub.

---

## Static assets

| Method | Path | Port | Response |
| ------ | ---- | ---- | -------- |
| `GET` | `/*` | 1234 / 1235 | Embedded Solid SPA (`index.html` fallback) |

Vite dev proxies `/api` → daemon; production UIs call same-origin `/api`.

---

## Error conventions

| HTTP | When |
| ---- | ---- |
| `400` | malformed JSON, out-of-range config, unknown capture device id |
| `404` | unknown `/api` path |
| `405` | wrong method |
| `500` | internal failure |

WebSocket: recoverable errors as `error` JSON on participant socket; host control socket logs to daemon stderr for v1 (no `error` type on `:1235` control WS).

---

## Rate summary

| Channel | Rate |
| ------- | ---- |
| `host-state` | 50 Hz |
| Preview H.264 | 15 fps |
| `participant-view` | on change, max 10 Hz |
| Transport stats in `host-state` | 1 Hz (when wired) |
| Loop delay fields | ~0.3 Hz |

---

## Testing entry points

Go E2E (`spidercamd --mock`) exercises REST + both host WS paths + participant WS. Playwright + MSW mocks these URLs without the daemon.
