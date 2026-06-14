# Resolved decisions

| ID  | Decision                     | Rationale                                                                                                | Date       |
| --- | ---------------------------- | -------------------------------------------------------------------------------------------------------- | ---------- |
| D1  | Dual-port WS split           | `:1234` participant-view + WebRTC; `:1235` full host-state. Different channels by construction.          | 2026-06-11 |
| D2  | (superseded)                 | Replaced by Go dual-port — host UI on `:1235` loopback only.                                             | 2026-06-11 |
| D4  | Omit `cpuPercent`            | OS-level monitoring sufficient.                                                                          | 2026-06-11 |
| D7  | Pull-based mixer in Go       | Engine ticker @ 10 ms; jitter buffers per stream.                                                        | 2026-06-11 |
| D9  | SolidJS for UIs              | Host + participant static SPAs.                                                                          | 2026-06-11 |
| D10 | Tailwind CSS v4              | Shared `web/ui-theme`.                                                                                   | 2026-06-11 |
| D13 | Go + Pion daemon             | Single process: native capture, WebRTC hub, mixer, virtual output. No Electron/Node media path.          | 2026-06-11 |
| D14 | Playback reference           | Host speaker monitor = Teams room audio; echo correlation + host-configurable duck.                      | 2026-06-11 |
| D5  | Passive loop latency         | GCC-PHAT ref↔mic; per-participant bar + global max; `—` when unknown.                                    | 2026-06-11 |
| D8  | No host VAD special handling | Host mic competes on score only; no force-host or host-VAD ducking.                                      | 2026-06-11 |
| D11 | PipeWire + C capture         | Thin C shim for PW mic + sink monitor; host UI device pickers.                                           | 2026-06-11 |
| D6  | Testing / CI                 | Static `/` + API `/api`; Go WS/REST E2E; Playwright+MSW UI; single CI gate.                              | 2026-06-11 |
| D12 | Reference ducking            | echoPenalty + duck slider 0…−12 dB; session RAM.                                                         | 2026-06-11 |
| D15 | Host UI layout               | Timeline, vertical meters, settings panel, session-only config.                                          | 2026-06-11 |
| D16 | Host preview transport       | H.264 @ 15 fps on `/api/v1/ws/preview`; WebCodecs in browser; libx264 cgo in daemon.                     | 2026-06-11 |
| D17 | Dual-branch audio pipeline   | Raw analysis tap (scores, echoPenalty, loop delay) + enhancement branch (AEC, RNNoise) for mixer output. | 2026-06-11 |
| D18 | Per-stream AEC               | WebRTC APM AEC3; one instance per mic; `playback-ref` far-end; host toggle per card; default off.        | 2026-06-11 |
| D19 | Per-stream denoise           | RNNoise via cgo; host toggle per card; `aecUs`/`denoiseUs` + `enhancementBudgetPct`; default off.        | 2026-06-11 |
| D20 | Participant single screen    | One viewport; clientId UUID + cosmetic name; header routed dot; lost-host auto-reconnect.                | 2026-06-13 |
| D21 | WebRTC SDP direction         | Browser offers after join; Pion hub answers. Star topology — hub receives participant media.             | 2026-06-14 |
| D22 | PulseAudio virtual mic       | `github.com/jfreymuth/pulse` — float32 mono @ 48 kHz to named sink. Proof P5.6.                          | 2026-06-14 |
| D23 | v4l2 camera library          | `github.com/blackjack/webcam` for host cam capture. Proof P5.4.                                          | 2026-06-14 |
| D24 | v4l2loopback pixel format    | `VIDIOC_ENUM_FMT` + prefer YUYV/RGB24; `VIDIOC_G_FMT` fallback if `S_FMT` fails. Proof P5.5.             | 2026-06-14 |
| D25 | PipeWire thread model        | Dedicated `pthread` + `pw_main_loop`; SPSC rings; iterate loop during sync. Proof P5.2–P5.3, P5.9.       | 2026-06-14 |
| D26 | libx264 in CI                | Mock encoder in CI; `PackChunk` unit tests; real x264 on dev host only. Proof P5.7, P5.8.                | 2026-06-14 |
| D27 | `!cgo` build fallback        | Stub packages for `!cgo \|\| !linux`; `go test` without PW/x264 dev headers.                             | 2026-06-14 |
| D28 | Virtual device provisioning  | Check-only at startup; operator docs for modprobe + null sink; no auto-create in v1. Proof P5.5, P5.6.   | 2026-06-14 |
| D29 | Go module path               | `github.com/markus/spidercam` — `go.mod` at repo root. Proof P5.8.                                       | 2026-06-14 |
| D30 | Virtual cam device path      | Discover v4l2loopback via sysfs name; env `SPIDERCAM_VIDEO_DEVICE` overrides. Proof P5.5.                | 2026-06-14 |

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

## D9 / D10

SolidJS + Tailwind on `web/host` and `web/participant`, embedded in Go binary.

## D4

No CPU metric in host header.

## D5 — Passive loop latency

**Chosen:** Passive delay estimation only — no chirp/MLS test tones.

- **Probe:** remote Teams speech on `playback-ref` (speaker monitor).
- **Return path:** participant mic uplink (room hears TV/speakers).
- **Method:** GCC-PHAT over lag window during gated phases throughout the meeting.
- **Per participant:** median τ, spread → `loopDelay` with `ms`, `uncertaintyMs`, `known`.
- **Global:** `max(participant loopDelay.ms)` where `known`; JSON `null` → UI `—`. Single number, no range.
- **UI:** approximate loop text on stream cards; header `globalLatencyMs` only; see [ui/design-system.md](./ui/design-system.md).
- **Publish:** ~3 s cadence; stale after 5 min without good phases.

**Rejected:** RTT+jitter segment sum; audible calibration tones; global latency bar/range display.

## D8 — No host VAD special handling

**Chosen:** Host native mic is a normal candidate stream. Routing uses the same score-based hysteresis as participants.

- No `forceHostWhenVad` override in selector
- No host-VAD-triggered participant ducking in mixer
- Host may still rank higher via static `HostPriority` score weight (role, not VAD)

**Rejected:** Force main talker to host when host VAD active; duck participants when host speaks.

## D11 — PipeWire + C capture

**Chosen:** Native PipeWire in a **thin C layer** (`internal/capture/native/`), called from Go via cgo.

- Mic: PW capture source (user-selected in host UI).
- Playback reference: **monitor port** of user-selected **output sink** (Teams speaker).
- Camera: v4l2 list + user-selected device.
- Device enumeration over host WS; `set-capture-devices` → `capture.Reopen` without process restart.
- Env vars / config file for bootstrap defaults only.

**Rejected:** PortAudio/malgo as primary path; PulseAudio device-name monitor hacks; headless-only env configuration without UI pickers.

## D6 — Testing / CI

**Chosen:** Separate static UI from daemon API; single CI gate for one-shot implementation.

- **HTTP:** each port serves SPA at `/`, API at `/api/v1/...` (REST snapshots + WS on same origin).
- **Go E2E:** `spidercamd --mock` — tests speak WebSocket/REST directly, no DOM, no Playwright against daemon.
- **UI E2E:** Playwright + MSW mocks `/api` — no daemon in UI test job.
- **Fixtures:** shared golden JSON (`testdata/` ↔ `web/test-fixtures/`) for Go and MSW.
- **CI:** one workflow — `go test` (unit + e2e), lint, build, `npm run test:ui`. No nightly/manual tiers.

**Rejected:** Playwright driving live daemon for PR CI; tiered CI (PR vs nightly); browser-based backend verification.

## D12 — Reference ducking

**Chosen:** `echoPenalty` **and** reference ducking via a single **Ducking** slider (0 … −12 dB).

- **Always on:** `echoPenalty` in score (lag-0 correlation with playback reference).
- **Reference ducking:** when `referenceActive` and `referenceDuckDb < 0`, attenuate participant mic streams by that amount (default −12 dB). `0` = off.
- **Host UI:** slider in always-visible settings panel; partial `config` applies immediately; **session RAM only** (D15).
- **UI feedback:** score border brightness on cards; ducking effect audible in mix (no duck pill on cards).

**Rejected:** Penalty-only as the only bleed mitigation; separate enable toggle; preset buttons; disk persistence.

## D15 — Host UI layout & session config

**Chosen:** Host console layout and config model for operator clarity.

- **Preview panel:** hard-cut video; vertical OUT + REF meters (RMS + peak, clip LED at top); dBFS values; 45 s **state timeline** (`_`/`L`/`H`/`S`).
- **Timeline colors:** Silence = track bg; Locked = green accent; Hold = **teal**; Switch = yellow.
- **Stream grid:** fixed **168×240** cards, **`grid-cols-5`**; host first, then participants by join order; vertical meter; **red on-air dot**; loop delay **text**; **transport 2×3** on card (host: `—` for WebRTC slots); **score border** opacity ∝ `scoreSmooth`; no VAD pill; no expand/collapse.
- **No transport footer or debug drawer** — diagnosis via timeline + cards + daemon logs.
- **Settings panel:** replaces mixer-brain column and settings overlay — devices + hold/crossfade/ducking/switch margin/score weights; AEC/NS on stream cards; no presets.
- **Audio/video:** equal-power crossfade on **audio only**; video **hard cut**.
- **Persistence:** `HostConfig` and capture selection in daemon RAM for the session; `DefaultHostConfig` + env bootstrap on restart. No `host-config.json` / `devices.json`.

**Rejected:** OUT sparkline; REF strip on rail; LoopDelayBar; settings overlay; preset buttons; disk persistence; video crossfade; green border for on-air; transport table; debug drawer; VAD pill on cards; variable card sizing (`minmax`).

## D16 — Host preview stream

**Chosen:** Separate preview subsystem on second host WebSocket path.

- **Transport:** `GET /api/v1/ws/preview` on `:1235` (loopback) — JSON init/cut + binary H.264 @ **15 fps**
- **Source:** same compositor RGBA as v4l2 output; subsample 30→15; **ForceKeyframe** on `activeVideoId` change
- **UI:** WebCodecs `VideoDecoder` → canvas; meters/timeline still from `/api/v1/ws` `host-state`
- **Encoder:** libx264 via cgo (`tune=zerolatency`, baseline); mock encoder for `--mock` CI
- **Deps:** `libx264-dev`; no npm video packages

**Rejected:** JPEG/RGBA frames on control WS; third TCP port; WebRTC loopback to host UI; MSE/fMP4 mux (WebCodecs preferred).

## D17 — Dual-branch audio pipeline

**Chosen:** Two parallel signal paths after HPF + calibration.

- **Analysis tap (raw):** `echoPenalty`, GCC-PHAT loop delay, VAD, SNR, noise floor, selector scores.
- **Enhancement branch:** optional AEC then optional RNNoise → gate/duck → mixer.
- Card meters show post-enhancement; scores use raw acoustics.

**Rejected:** Single-path processing (enhancement before analysis); global NS slider.

## D18 — Per-stream AEC

**Chosen:** WebRTC Audio Processing Module **AEC3** via C++ shim + cgo.

- One `AudioProcessing` instance per stream (host + participants).
- `ProcessReverseStream(playback-ref)` + `ProcessStream(mic)` each 10 ms.
- Host toggle on stream card; session RAM; default **off**.
- `aecUs` EMA on `StreamMetrics`.

**Rejected:** SpeexDSP as primary; custom NLMS; correlation-only as sole echo fix.

## D20 — Participant monitor (single screen + reconnect)

**Chosen:** One viewport for all participant UX; connection is a toggle, not a route change.

- **Layout:** display name (default `client-{random}`) + **clientId** UUID + local preview + device pickers always visible; Connect / Disconnect on same screen.
- **Identity:** `welcome.clientId` = routing id; `join.name` = cosmetic display name only.
- **On-air UX:** header **red dot** when `activeAudioId === clientId`; **On air:** text row from `mainTalkerId` (`you` / name / `host`) — no dot on that row.
- **Lost host:** banner on same screen when WS/WebRTC drops while connected; exponential backoff **auto-reconnect** + auto-`join`; local preview stays up; user Disconnect cancels retry.
- **Disconnect:** `leave` + close peer/WS; remain on same screen with local preview.

**Rejected:** Separate connect vs session pages; red dot on “On air:” row; navigate away on leave; manual re-enter name after host restart.

## D19 — Per-stream denoise

**Chosen:** [RNNoise](https://github.com/xiph/rnnoise) for voice isolation.

- Independent of AEC toggle; runs after AEC on enhancement branch.
- Host toggle on stream card; default **off**.
- `denoiseUs` per stream; `enhancementBudgetPct` in header.

**Rejected:** Global `nsLevel` slider; DeepFilterNet in scope; cloud enhancement APIs.

## D21 — WebRTC SDP negotiation direction

**Chosen:** Participant browser creates the SDP offer after `join`; Pion hub answers.

- Signaling on `:1234` WS: client `offer` → hub `answer`; ICE relay both ways.
- Hub is receive-only for participant mic/cam (star topology); host A/V from native capture, not WebRTC.
- Wave 8 participant adapter: `RTCPeerConnection.createOffer()` → apply hub answer.

**Rejected:** Hub-initiated offer as primary path; dual-direction negotiation without a single owner.

## D22 — PulseAudio virtual mic

**Chosen:** `github.com/jfreymuth/pulse` for float32 mono playback to a named PulseAudio sink.

- Teams selects `spidercam_sink` (null sink) as microphone input.
- Proof [P5.6](../experiments/wave5/p5.6-pulse/): 440 Hz tone verified via `spidercam_sink.monitor`.

**Rejected:** `github.com/jfreymuth/pulseaudio` (module does not exist); C shim; `pacat` subprocess per frame.

## D23 — v4l2 camera library

**Chosen:** `github.com/blackjack/webcam` for listing and opening host cameras.

- Device list via `/dev/video*` + sysfs card name.
- Proof [P5.4](../experiments/wave5/p5.4-v4l2-cam/): YUYV 640×480 → PNG from `/dev/video0`.

**Rejected:** Raw ioctl-only for v1; `go4vl` (unnecessary until blackjack limits hit).

## D24 — v4l2loopback pixel format

**Chosen:** Enumerate output formats at open; prefer YUYV then RGB24; convert compositor RGBA before write.

- If `VIDIOC_S_FMT` fails (e.g. requested 1280×720), fall back to `VIDIOC_G_FMT` defaults and write at negotiated size.
- Proof [P5.5](../experiments/wave5/p5.5-loopback/): 300 frames written to loopback.

**Rejected:** Fixed format without enumeration; assuming `S_FMT` always succeeds at 1280×720.

## D25 — PipeWire thread model

**Chosen:** Dedicated `pthread` runs `pw_main_loop`; Go engine tick pulls SPSC ring buffers (480 samples × 8 frames).

- During setup: iterate `pw_loop` while `pw_core_sync` pending — do not block before `pw_main_loop_run`.
- Teardown: `spa_hook_remove`, not deprecated `pw_registry_destroy` arity.
- `capture.Reopen` tears down and relinks streams; proof [P5.9](../experiments/wave5/p5.9-reopen/) reopen **11.82 ms**.

**Rejected:** PipeWire on Go main thread; blocking PW calls from 10 ms engine tick.

## D26 — libx264 in CI

**Chosen:** CI uses `--mock` preview encoder; unit tests cover `PackChunk` / `AnnexBToAVCC` only ([P5.8](../experiments/wave5/p5.8-preview-pack/)).

- Production: libx264 cgo (`ultrafast`, `zerolatency`, baseline) on dev/operator host ([P5.7](../experiments/wave5/p5.7-x264/)).
- Do not install `libx264-dev` in GitHub Actions for v1.

**Rejected:** Full encode tests in CI; macOS/Windows x264 gates.

## D27 — `!cgo` build fallback

**Chosen:** `//go:build !cgo || !linux` stubs for capture, output, and preview encoder.

- `go test ./internal/...` runs without `libpipewire-0.3-dev` or `libx264-dev`.
- Linux production build: `CGO_ENABLED=1 go build ./cmd/spidercamd`.

**Rejected:** cgo required for all `go test` targets.

## D28 — Virtual device provisioning

**Chosen:** Check-only at startup — clear log message and `outputHealthy: false` when virtual devices missing.

- Operator docs: `modprobe v4l2loopback …` and `pactl load-module module-null-sink sink_name=spidercam_sink`.
- No `spidercamd setup-devices` subcommand in v1.

**Rejected:** Auto-create loopback/null sink from daemon; silent failure.

## D29 — Go module path

**Chosen:** `github.com/markus/spidercam` — root `go.mod`, `internal/*` packages.

- Established by [P5.8](../experiments/wave5/p5.8-preview-pack/) preview framing tests.

## D30 — Virtual cam device path

**Chosen:** Resolve v4l2loopback device by sysfs name (`*loopback*`), not a fixed `/dev/video2`.

- `video_nr=2` may not land on `/dev/video2` when that node is already taken (integrated camera metadata).
- Env `SPIDERCAM_VIDEO_DEVICE` overrides auto-discovery when set.
- Proof [P5.5](../experiments/wave5/p5.5-loopback/): loopback at `/dev/video4`, `card_label=spidercam-loopback`.

**Rejected:** Hardcoded `/dev/video2` as the only production path.
