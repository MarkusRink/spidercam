# Testing

One-shot implementation — **single CI gate**, no tiered nightly/manual pipelines. Tests give agents deterministic pass/fail without driving a browser against the daemon.

## HTTP layout (per listener)

Each port serves **static UI at `/`** and **API at `/api`** on the same origin. Dual-port split (D1) unchanged — host loopback `:1235`, participant LAN `:1234`.

```text
:1235 (host)                          :1234 (participant)
├── GET  /*              → host SPA   ├── GET  /*              → participant SPA
├── GET  /api/health                  ├── GET  /api/health
├── GET  /api/v1/host/state           ├── WS   /api/v1/ws       → participant signaling
├── POST /api/v1/host/config          └── (no host REST)
├── GET  /api/v1/capture/devices
├── POST /api/v1/capture/selection
├── WS   /api/v1/ws       → host control (host-state, inbound config)
└── WS   /api/v1/ws/preview → H.264 preview stream
```

Production UIs call `/api/*` on their port. Vite dev proxies `/api` → daemon.

REST snapshots exist for **tests and debug**; live dashboards stay WS-primary for high-rate `host-state`.

## Shared fixtures

Golden JSON in `testdata/` (or `web/test-fixtures/`) — same shapes as [domain/messages.md](./domain/messages.md) and [domain/types.md](./domain/types.md):

| Fixture | Used by |
|---------|---------|
| `host-state/*.json` | Go E2E, MSW host WS handler |
| `participant-view/*.json` | Go E2E, MSW participant WS handler |
| `capture-devices.json` | Go E2E, MSW REST |
| `pcm/*.raw` | Go unit — correlation, GCC-PHAT, bleed scenarios |
| `preview/keyframe.h264` | Go E2E + mock encoder — valid IDR NAL |

Go and TypeScript must not drift — fixtures are the contract.

---

## 1. Go unit tests

| Package | Cases |
|---------|-------|
| `internal/audio/math` | rmsDbfs, normLevel, frameScore, equalPowerGains, correlation, GCC-PHAT |
| `internal/selector` | silence, margin+hold, emergency, ref excluded, no host-VAD override |
| `internal/audio/reference` | ref VAD, echoPenalty on raw, delay tracker, global max |
| `internal/audio/processor` | dual branch: raw scores vs post-enhancement meters; VAD hangover; calibration |
| `internal/audio/aec` | passthrough mock; ERLE tolerance on synthetic FIR (optional golden) |
| `internal/audio/enhance` | passthrough; float↔int16 clip; denoiseUs EMA |
| `internal/protocol` | message decode, participant vs host types, golden JSON round-trip |
| `internal/preview` | chunk framing, subsample 30→15 fps, ForceKeyframe on id change |

Run: `go test ./internal/... -race -count=1`

Synthetic PCM from `testdata/pcm/` — no PipeWire, no browser.

---

## 2. Go API / WS E2E (daemon, no DOM)

Primary integration gate. Speaks **WebSocket + REST** to `spidercamd --mock --no-open-browser`. No Playwright, no DOM.

**Target:** `test/e2e/` or `internal/e2e/` with build tag `e2e`.

| Test file | Cases |
|-----------|-------|
| `host_api_test.go` | `GET /api/health`; `GET /api/v1/host/state`; WS `/api/v1/ws` → `host-state`; `POST /api/v1/host/config` or WS `config`; `POST /api/v1/host/stream-processing`; capture list/set |
| `host_preview_test.go` | WS `/api/v1/ws/preview` → `preview-stream-init` → binary key chunk (flags bit0) |
| `participant_api_test.go` | WS `/api/v1/ws` → `welcome`; `join` → `participant-view`; never receives full `RoomState` |
| `room_flow_test.go` | participant join → host state shows stream count; leave cleanup |
| `webrtc_test.go` | Pion offer/answer via participant WS (fake peer / pion test helpers) |

Bootstrap:

```bash
spidercamd --mock --no-open-browser &
go test ./test/e2e/... -tags=e2e -count=1
```

Mock flags: `--mock`, `SPIDERCAM_MOCK_CAPTURE=1`, `SPIDERCAM_MOCK_OUTPUT=1`, `SPIDERCAM_MOCK_PREVIEW=1` — no v4l2/PipeWire/libx264/librnnoise/WebRTC-APM in CI.

Scripted PCM injection for UC-H7/H8: feed correlated ref+mic frames through mock capture, assert scores and `loopDelay` via `GET /api/v1/host/state`.

---

## 3. UI tests — Playwright + MSW (no daemon)

Browser tests **mock `/api`** with [MSW](https://mswjs.io/) — static UI only, no `spidercamd` in this suite.

**Target:** `web/host/e2e/`, `web/participant/e2e/`

| Spec | MSW mocks | Asserts |
|------|-----------|---------|
| `host.spec.ts` | WS canned `host-state`; REST capture devices; preview WS mocked or omitted | OUT/REF meters, state timeline, stream grid cards (transport cells, on-air dot, score border), settings → config — **no preview pixel assert** |
| `participant.spec.ts` | WS `welcome` + `participant-view`; WS close → lost-host banner | single-screen layout, device pickers always visible, connect toggle, on-air label + header dot, loop delay, auto-reconnect flow |
| `mixer-settings.spec.ts` | host-state + config capture | ducking slider, hold/crossfade, score weight sliders |

Playwright `webServer`: `vite preview` (or static `dist/`) — **not** the Go binary.

MSW handlers import fixtures from `web/test-fixtures/` (mirrors `testdata/`).

WebRTC / `getUserMedia`: fake device flags only for connect UX if needed; **do not** assert mixer behavior through the browser.

Optional **Vitest** for pure functions (`StateTimeline` bucketing, `LoopDelayText` formatting, `TransportBlock` threshold tones, score border opacity) — no browser.

---

## What each layer owns

| Concern | Layer |
|---------|-------|
| Scoring, selector, GCC-PHAT, echoPenalty, dual-branch processor | Go unit |
| AEC/RNNoise passthrough + timing fields | Go unit + mock daemon |
| WS protocol, room projection, WebRTC hub | Go API E2E |
| REST handler shapes | Go API E2E |
| Rendering, forms, widgets | Playwright + MSW |
| PipeWire, v4l2, real Teams | Manual room test (out of CI) |

---

## Acceptance mapping

| Use case | Test |
|----------|------|
| UC-H1 | Go E2E daemon start + `/api/health` |
| UC-H2 | Go E2E host state + `/api/v1/ws/preview` init + keyframe binary assert |
| UC-H3 | `selector_test` + Playwright MSW state timeline + on-air dot |
| UC-H4 | Playwright MSW stream grid transport cells + score border |
| UC-H5 | Playwright MSW timeline segments; Go logs for deep diagnosis |
| UC-H6 | Playwright MSW settings → config payload; stream card AEC/NS toggle → `set-stream-processing` |
| UC-H7 | Go unit reference + Go E2E mock PCM bleed scenario |
| UC-H8 | Go unit delay tracker + Go E2E state snapshot |
| UC-H9 | Go E2E capture devices REST + reopen |
| UC-P1 | Playwright MSW participant connect toggle on single screen |
| UC-P2 | Playwright MSW loop delay text when connected |
| UC-P3 | Playwright MSW on-air label + header routed dot |
| UC-P5 | Playwright MSW WS drop → lost-host banner → auto-rejoin |

---

## CI (single gate)

One workflow — all required, no nightly tier:

```yaml
# .github/workflows/ci.yml (target)
- go vet ./...
- go test ./internal/... -race -count=1
- go test ./test/e2e/... -tags=e2e -count=1   # spawns spidercamd --mock
- npm ci && npm run lint && npm run format && npm run build
- npm run test:ui                              # Playwright + MSW, no daemon
```

## Quality gate (local / agent)

```bash
make check   # or equivalent single script mirroring CI exactly
```

```bash
go vet ./...
go test ./internal/... -race -count=1
go test ./test/e2e/... -tags=e2e -count=1
npm run lint && npm run format && npm run build
npm run test:ui
```

No optional/skipped steps — if it fails, the one-shot build is not done.
