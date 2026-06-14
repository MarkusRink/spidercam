# Decisions

All architecture decisions for the Go daemon iteration. Status: **resolved**.

| ID | Decision | Rationale | Date |
| -- | -------- | --------- | ---- |
| D1 | Dual-port WS split | `:1234` participant-view + WebRTC; `:1235` full host-state. Different channels by construction. | 2026-06-11 |
| D2 | (superseded) | Replaced by Go dual-port — host UI on `:1235` loopback only. | 2026-06-11 |
| D3 | Pull-based frame engine | Go pull engine @ 10 ms; no browser AudioWorklet. | 2026-06-11 |
| D4 | Omit `cpuPercent` | OS-level monitoring sufficient. | 2026-06-11 |
| D5 | Passive loop latency | GCC-PHAT ref↔mic; per-participant bar + global max; `—` when unknown. | 2026-06-11 |
| D6 | Testing / CI | Static `/` + API `/api`; Go WS/REST E2E; Playwright+MSW UI; single CI gate. | 2026-06-11 |
| D7 | Pull-based mixer in Go | Engine ticker @ 10 ms; jitter buffers per stream. | 2026-06-11 |
| D8 | No host VAD special handling | Host mic competes on score only; no force-host or host-VAD ducking. | 2026-06-11 |
| D9 | SolidJS for UIs | Host + participant static SPAs. | 2026-06-11 |
| D10 | Tailwind CSS v4 | Shared `web/ui-theme`. | 2026-06-11 |
| D11 | PipeWire + C capture | Thin C shim for PW mic + sink monitor; host UI device pickers. | 2026-06-11 |
| D12 | Reference ducking | echoPenalty + duck slider 0…−12 dB; session RAM. | 2026-06-11 |
| D13 | Go + Pion daemon | Single process: native capture, WebRTC hub, mixer, virtual output. No Electron/Node media path. | 2026-06-11 |
| D14 | Playback reference | Host speaker monitor = Teams room audio; echo correlation + host-configurable duck. | 2026-06-11 |
| D15 | Host UI layout | Timeline, vertical meters, settings panel, session-only config. | 2026-06-11 |
| D16 | Host preview transport | H.264 @ 15 fps on `/api/v1/ws/preview`; WebCodecs in browser; libx264 cgo in daemon. | 2026-06-11 |
| D17 | Dual-branch audio pipeline | Raw analysis tap (scores, echoPenalty, loop delay) + enhancement branch (AEC, RNNoise) for mixer output. | 2026-06-11 |
| D18 | Per-stream AEC | WebRTC APM AEC3; one instance per mic; `playback-ref` far-end; host toggle per card; default off. | 2026-06-11 |
| D19 | Per-stream denoise | RNNoise via cgo; host toggle per card; `aecUs`/`denoiseUs` + `enhancementBudgetPct`; default off. | 2026-06-11 |
| D20 | Participant single screen | One viewport; clientId UUID + cosmetic name; header routed dot; lost-host auto-reconnect. | 2026-06-13 |
| D21 | WebRTC SDP direction | Browser offers after join; Pion hub answers. Star topology — hub receives participant media. | 2026-06-14 |
| D22 | PulseAudio virtual mic | `github.com/jfreymuth/pulse` — float32 mono @ 48 kHz to named sink. Proof P5.6. | 2026-06-14 |
| D23 | v4l2 camera library | `github.com/blackjack/webcam` for host cam capture. Proof P5.4. | 2026-06-14 |
| D24 | v4l2loopback pixel format | `VIDIOC_ENUM_FMT` + prefer YUYV/RGB24; `VIDIOC_G_FMT` fallback if `S_FMT` fails. Proof P5.5. | 2026-06-14 |
| D25 | PipeWire thread model | Dedicated `pthread` + `pw_main_loop`; SPSC rings; iterate loop during sync. Proof P5.2–P5.3, P5.9. | 2026-06-14 |
| D26 | libx264 in CI | Mock encoder in CI; `PackChunk` unit tests; real x264 on dev host only. Proof P5.7, P5.8. | 2026-06-14 |
| D27 | `!cgo` build fallback | Stub packages for `!cgo \|\| !linux`; `go test` without PW/x264 dev headers. | 2026-06-14 |
| D28 | Virtual device provisioning | Check-only at startup; operator docs for modprobe + null sink; no auto-create in v1. Proof P5.5, P5.6. | 2026-06-14 |
| D29 | Go module path | `github.com/markus/spidercam` — `go.mod` at repo root. Proof P5.8. | 2026-06-14 |
| D30 | Virtual cam device path | Discover v4l2loopback via sysfs name; env `SPIDERCAM_VIDEO_DEVICE` overrides. Proof P5.5. | 2026-06-14 |

Wave 5 proof evidence: [experiments/wave5/README.md](../experiments/wave5/README.md) (P5.1–P5.9).

---

## D1 — Host vs participant state channel

**Chosen:** Two HTTP listeners, two WS protocols.

- Participant never receives `RoomState.metrics[]`
- Host UI never uses participant port for state
- No broadcast fan-out of fat payloads

## D13 — Go + Pion daemon

**Chosen:** `cmd/spidercamd` replaces Electron, embedded Node server, browser audio engine, and WS bridge.

- Pion terminates participant WebRTC
- Native capture for host mic, cam, **speaker loopback**
- Direct virtual device output

**Rejected:** Electron renderer WebRTC + AudioWorklet path.

## D14 — Playback reference (room loopback)

**Chosen:** Capture monitor of host machine default speaker (Teams output).

- Reference stream `playback-ref` for analysis only
- `echoPenalty` via normalized correlation per 10 ms frame
- `referenceDuckDb` (0 … −12 dB) — host slider in settings panel (D12)
- Host operator in Teams → speaker output matches conference audio played in room

**Rejected:** Relying on Teams AEC to fix multi-mic room bleed.

## D5 — Passive loop latency

**Chosen:** Passive delay estimation only — no chirp/MLS test tones.

- **Probe:** remote Teams speech on `playback-ref` (speaker monitor).
- **Return path:** participant mic uplink (room hears TV/speakers).
- **Method:** GCC-PHAT over lag window during gated phases throughout the meeting.
- **Per participant:** median τ, spread → `loopDelay` with `ms`, `uncertaintyMs`, `known`.
- **Global:** `max(participant loopDelay.ms)` where `known`; JSON `null` → UI `—`.
- **Publish:** ~3 s cadence; stale after 5 min without good phases.

**Rejected:** RTT+jitter segment sum; audible calibration tones; global latency bar/range display.

## D8 — No host VAD special handling

**Chosen:** Host native mic is a normal candidate stream. Routing uses the same score-based hysteresis as participants.

- No `forceHostWhenVad` override in selector
- No host-VAD-triggered participant ducking in mixer
- Host may still rank higher via static `HostPriority` score weight (role, not VAD)

## D11 — PipeWire + C capture

**Chosen:** Native PipeWire in a **thin C layer** (`internal/capture/native/`), called from Go via cgo.

- Mic: PW capture source (user-selected in host UI).
- Playback reference: **monitor port** of user-selected **output sink** (Teams speaker).
- Camera: v4l2 list + user-selected device.
- Device enumeration over host WS; `set-capture-devices` → `capture.Reopen` without process restart.
- Env vars seed bootstrap defaults only.

**Rejected:** PortAudio/malgo as primary path; PulseAudio device-name monitor hacks; headless-only env configuration without UI pickers.

## D6 — Testing / CI

**Chosen:** Separate static UI from daemon API; single CI gate.

- **HTTP:** each port serves SPA at `/`, API at `/api/v1/...`
- **Go E2E:** `spidercamd --mock` — tests speak WebSocket/REST directly
- **UI E2E:** Playwright + MSW mocks `/api` — no daemon in UI test job
- **CI:** one workflow — `go test`, lint, build, `npm run test:ui`

**Rejected:** Playwright driving live daemon for PR CI; tiered CI.

## D12 — Reference ducking

**Chosen:** `echoPenalty` **and** reference ducking via a single **Ducking** slider (0 … −12 dB).

- **Always on:** `echoPenalty` in score (lag-0 correlation with playback reference).
- **Reference ducking:** when `referenceActive` and `referenceDuckDb < 0`, attenuate participant mic streams by that amount (default −12 dB). `0` = off.
- **Session RAM only** (D15).

## D15 — Host UI layout & session config

**Chosen:** Host console layout and config model for operator clarity.

- **Preview panel:** hard-cut video; vertical OUT + REF meters; dBFS values; 45 s **state timeline** (`_`/`L`/`H`/`S`).
- **Stream grid:** fixed **168×240** cards, **`grid-cols-5`**; host first, then participants by join order.
- **Settings panel:** devices + hold/crossfade/ducking/switch margin/score weights; AEC/NS on stream cards.
- **Persistence:** `HostConfig` and capture selection in daemon RAM for the session. No disk persistence.

## D16 — Host preview stream

**Chosen:** Separate preview subsystem on second host WebSocket path.

- **Transport:** `GET /api/v1/ws/preview` on `:1235` — JSON init/cut + binary H.264 @ **15 fps**
- **Source:** compositor RGBA as v4l2 output; subsample 30→15; **ForceKeyframe** on `activeVideoId` change
- **UI:** WebCodecs `VideoDecoder` → canvas
- **Encoder:** libx264 via cgo (`tune=zerolatency`, baseline); mock encoder for `--mock` CI

## D17 — Dual-branch audio pipeline

**Chosen:** Two parallel signal paths after HPF + calibration.

- **Analysis tap (raw):** `echoPenalty`, GCC-PHAT loop delay, VAD, SNR, noise floor, selector scores.
- **Enhancement branch:** optional AEC then optional RNNoise → gate/duck → mixer.
- Card meters show post-enhancement; scores use raw acoustics.

## D18 / D19 — Per-stream AEC and denoise

**Chosen:** WebRTC APM AEC3 and RNNoise, each independently toggled per stream on host cards. Default **off**. `enhancementBudgetPct` in header when active.

## D20 — Participant monitor

**Chosen:** One viewport for all participant UX; Connect/Disconnect toggle; lost-host auto-reconnect with exponential backoff.

## D21 — WebRTC SDP negotiation

**Chosen:** Participant browser creates the SDP offer after `join`; Pion hub answers. Implemented in `internal/webrtc/` and `web/participant/src/adapters/live-peer.ts`.

## D22–D30 — Native I/O (Wave 5)

Validated in `experiments/wave5/` before production packages:

- **D22:** PulseAudio via `github.com/jfreymuth/pulse` → `internal/output/pulse.go`
- **D23:** v4l2 capture via `github.com/blackjack/webcam` → `internal/capture/v4l2_camera.go`
- **D24–D30:** loopback format negotiation, PW thread model, CI mock encoder, cgo stubs, check-only device provisioning, module path, sysfs loopback discovery
