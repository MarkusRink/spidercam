# Resolved decisions

| ID | Decision | Rationale | Date |
|----|----------|-----------|------|
| D1 | Dual-port WS split | `:1234` participant-view + WebRTC; `:1235` full host-state. Different channels by construction. | 2026-06-11 |
| D2 | (superseded) | Replaced by Go dual-port — host UI on `:1235` loopback only. | 2026-06-11 |
| D4 | Omit `cpuPercent` | OS-level monitoring sufficient. | 2026-06-11 |
| D7 | Pull-based mixer in Go | Engine ticker @ 10 ms; jitter buffers per stream. | 2026-06-11 |
| D9 | SolidJS for UIs | Host + participant static SPAs. | 2026-06-11 |
| D10 | Tailwind CSS v4 | Shared `web/ui-theme`. | 2026-06-11 |
| D13 | Go + Pion daemon | Single process: native capture, WebRTC hub, mixer, virtual output. No Electron/Node media path. | 2026-06-11 |
| D14 | Playback reference | Host speaker monitor = Teams room audio; echo correlation + host-configurable duck. | 2026-06-11 |
| D5 | Passive loop latency | GCC-PHAT ref↔mic; per-participant bar + global max; `—` when unknown. | 2026-06-11 |
| D8 | No host VAD special handling | Host mic competes on score only; no force-host or host-VAD ducking. | 2026-06-11 |
| D11 | PipeWire + C capture | Thin C shim for PW mic + sink monitor; host UI device pickers. | 2026-06-11 |
| D6 | Testing / CI | Static `/` + API `/api`; Go WS/REST E2E; Playwright+MSW UI; single CI gate. | 2026-06-11 |
| D12 | Reference ducking | echoPenalty + duck slider 0…−12 dB; session RAM. | 2026-06-11 |
| D15 | Host UI layout | Timeline, vertical meters, settings panel, session-only config. | 2026-06-11 |
| D16 | Host preview transport | H.264 @ 15 fps on `/api/v1/ws/preview`; WebCodecs in browser; libx264 cgo in daemon. | 2026-06-11 |
| D17 | Dual-branch audio pipeline | Raw analysis tap (scores, echoPenalty, loop delay) + enhancement branch (AEC, RNNoise) for mixer output. | 2026-06-11 |
| D18 | Per-stream AEC | WebRTC APM AEC3; one instance per mic; `playback-ref` far-end; host toggle per card; default off. | 2026-06-11 |
| D19 | Per-stream denoise | RNNoise via cgo; host toggle per card; `aecUs`/`denoiseUs` + `enhancementBudgetPct`; default off. | 2026-06-11 |
| D20 | Participant single screen | One viewport; clientId UUID + cosmetic name; header routed dot; lost-host auto-reconnect. | 2026-06-13 |

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
