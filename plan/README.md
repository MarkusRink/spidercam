# Spidercam — Planning index

Target-state specification for the Go daemon iteration. **Do not treat the existing Node/Electron codebase as the spec** — these documents define what we build.

## High level

| Document | Purpose |
|----------|---------|
| [USE_CASE.md](./USE_CASE.md) | Actors, goals, success criteria, user actions |
| [ARCHITECTURE.md](./ARCHITECTURE.md) | Go daemon topology, ports, data flow |
| [API.md](./API.md) | REST + WebSocket routes (both ports) |
| [UI.md](./UI.md) | Host + participant screens, update loops |
| [DECISIONS.md](./DECISIONS.md) | Open items before implementation |
| [GLOSSAR.md](./GLOSSAR.md) | Term definitions |

## Architecture drill-down

| Document | Target |
|----------|--------|
| [architecture/daemon.md](./architecture/daemon.md) | CLI, dual HTTP, browser launch |
| [architecture/capture.md](./architecture/capture.md) | PipeWire C shim, mic, cam, sink monitor |
| [architecture/output.md](./architecture/output.md) | Virtual cam + mic |
| [architecture/signaling.md](./architecture/signaling.md) | Participant + host WebSocket handlers |
| [architecture/preview.md](./architecture/preview.md) | H.264 host preview stream subsystem |
| [architecture/webrtc.md](./architecture/webrtc.md) | Pion star hub |

## Domain

| Document | Target |
|----------|--------|
| [domain/types.md](./domain/types.md) | Go structs + JSON protocol |
| [domain/messages.md](./domain/messages.md) | WS message variants |
| [domain/host-config.md](./domain/host-config.md) | Tuning defaults |

## Audio pipeline

| Document | Target |
|----------|--------|
| [audio/overview.md](./audio/overview.md) | Dual-branch graph (analysis + enhancement) |
| [audio/reference-loopback.md](./audio/reference-loopback.md) | Playback reference + correlation/ducking |
| [audio/echo-cancellation.md](./audio/echo-cancellation.md) | WebRTC APM AEC3 per stream |
| [audio/enhancement.md](./audio/enhancement.md) | RNNoise per stream |
| [audio/math.md](./audio/math.md) | `internal/audio/math` |
| [audio/frame-engine.md](./audio/frame-engine.md) | `internal/audio/engine` |
| [audio/stream-processor.md](./audio/stream-processor.md) | Per-stream dual branch |
| [audio/selector.md](./audio/selector.md) | `internal/selector` |
| [audio/mixer.md](./audio/mixer.md) | `internal/audio/mixer` |

## UI drill-down

| Document | Target |
|----------|--------|
| [ui/design-system.md](./ui/design-system.md) | Tailwind theme |
| [ui/host-console.md](./ui/host-console.md) | `web/host/` → `:1235` |
| [ui/participant-monitor.md](./ui/participant-monitor.md) | `web/participant/` → `:1234` |

## Quality

| Document | Purpose |
|----------|---------|
| [testing.md](./testing.md) | Go unit + API E2E; Playwright+MSW UI; single CI |

## Reading order

1. USE_CASE → ARCHITECTURE → UI  
2. Resolve [DECISIONS.md](./DECISIONS.md)  
3. Domain + audio (Go interfaces are canonical)  
4. Implement per [ARCHITECTURE.md](./ARCHITECTURE.md) delivery sequence
