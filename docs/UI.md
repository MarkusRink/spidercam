# UI specification

Design principle: **every pixel shows mixer state** — dark panels, mono labels, green accent (`#3dd68c`). **Tailwind CSS v4** + shared theme in `web/ui-theme/`.

Both UIs are **SolidJS static SPAs** embedded in `spidercamd`. No media processing in the browser except participant WebRTC capture.

## Screens

| Screen | URL | WS | Media |
| ------ | --- | -- | ----- |
| Host console | `http://127.0.0.1:1235/` | `/api/v1/ws` — `host-state`; `/api/v1/ws/preview` — H.264 | Preview via WebCodecs (daemon pushes stream) |
| Participant | `http://<lan>:1234/` | `/api/v1/ws` — `participant-view` | `getUserMedia` + WebRTC → Pion |

## Host console (1080p, no page scroll)

Layout in `web/host/src/App.tsx`:

```
┌─────────────────────────────────────────────────────────────────────────┐
│ HEADER: spidercam/host · output ● · N streams · latency · DSP · [copy] │
├──────────────────────────────────────────┬──────────────────────────────┤
│ OUTPUT PREVIEW (hard-cut video)          │ SETTINGS PANEL               │
│  OUT ▮  REF ▮  (dBFS meters)             │ Mic / Webcam / Speaker ▾     │
│  clip LED top of each vertical meter     │ Hold · Crossfade · Ducking   │
│ [___LLLLHSSSSSLLLL________________]     │ Switch margin · weights      │
│  45 s state timeline                     │                              │
├──────────────────────────────────────────┴──────────────────────────────┤
│ STREAM GRID (fixed 168×240 cards, grid-cols-5, host + participants)     │
└─────────────────────────────────────────────────────────────────────────┘
```

Grid: `grid-rows-[40px_minmax(0,1fr)_260px] grid-cols-[1fr_360px]`. No transport footer. No debug drawer. REF level only in preview panel (not on stream cards).

### State timeline (45 s)

Rolling strip driven by `selection.mixerState` (~20 Hz samples, bucketed for display). Implemented in `web/ui-theme/src/StateTimeline.tsx`. Each cell is one color:

| Symbol | `mixerState` | Color |
| ------ | ------------ | ----- |
| `_` | `SILENCE` | `--color-spider-meter-track` (background) |
| `L` | `LOCKED` | `--color-spider-accent` (green) |
| `H` | `HOLD` | `--color-spider-hold` (teal) |
| `S` | `SWITCH` | `--color-spider-warn` (yellow) |

Example: `[___LLLLHSSSSSLLLL__]`. Audio crossfade duration appears as a run of `S` cells.

Timeline buffer: `TIMELINE_SECONDS * HOST_STATE_HZ` (45 × 50) in `web/host/src/stores/session-store.tsx`.

### Host update loops

| Data | Rate | Source |
| ---- | ---- | ------ |
| Vertical meters (OUT, REF, stream cards) | WS `host-state` ~20 Hz + rAF interpolate | Daemon |
| Card border opacity (`scoreSmooth`) | same `host-state` | Daemon |
| State timeline | append `mixerState` on each `host-state` tick | UI buffer (45 s) |
| Preview video | H.264 on `/api/v1/ws/preview` @ 15 fps | Daemon compositor + `internal/preview` |
| Transport cells on cards | `host-state` @ 1 Hz | Daemon |
| Loop delay text, global latency | `host-state` @ ~0.3 Hz | Daemon passive loop |
| `enhancementBudgetPct`, `aecUs`/`denoiseUs` | `host-state` ~20 Hz | Daemon enhancement branch |

SolidJS store (`session-store.tsx`) subscribes via `LiveHostSignaling`.

## Participant monitor (one viewport)

Single screen — no connect vs session routes. **Connect / Disconnect** is one toggle; local video preview, device pickers, and RMS meter are always visible. Layout in `web/participant/src/components/ParticipantShell.tsx`.

| Zone | Always | When connected |
| ---- | ------ | -------------- |
| Header | Editable display name (default `client-{random}`), muted **clientId** (UUID) | Red dot beside name when `activeAudioId === clientId` |
| Preview | Local camera + **AnalyserNode** meter | Same |
| Devices | Mic / camera dropdown + on/off toggles | Same; track swap on change |
| Room | — | On air: `you` / `host` / name (text only); loop delay; SNR; transport grid |

**Lost host connection:** banner on the same screen when WS/WebRTC drops; exponential backoff **auto-reconnect** with auto-`join`; local preview stays up. Implemented in `web/participant/src/stores/participant-store.ts`.

Display name and clientId persist in `sessionStorage` (`spidercam.displayName`, `spidercam.clientId`).

## Widget catalog

Shared components in `web/ui-theme/`:

| Component | File | Purpose |
| --------- | ---- | ------- |
| **VerticalVuMeter** | `VerticalVuMeter.tsx` | Current level bar + peak hold; red clip segment at top when `peakDbfs ≥ −1` |
| **StateTimeline** | `StateTimeline.tsx` | 45 s mixer-state history strip |
| **LoopDelayText** | `LoopDelayText.tsx` | Approximate ms on stream cards and participant session card |
| **TransportBlock** | `TransportBlock.tsx` | 2×3 mono grid: rtt, loss, jitter, buf, fps, A/V |
| **OnAirDot** | `OnAirDot.tsx` | Red (`--color-spider-error`); host stream cards when `activeAudioId === streamId`; participant header when `activeAudioId === clientId` |
| **StreamCard** | `StreamCard.tsx` | Score border opacity ∝ `scoreSmooth` |
| **StreamProcessingRow** | `StreamProcessingRow.tsx` | Per-card AEC / NS toggles + timing when enabled |

Theme tokens in `web/ui-theme/src/spidercam.css`: `--stream-card-w: 168px`, `--stream-card-h: 240px`.

## Settings panel (host, always visible)

Right column — not an overlay. Devices + mixer tuning; session RAM only. WS `config` / `set-capture-devices` on change. Implemented in `web/host/src/components/SettingsPanel.tsx`.

Default config values: `DefaultHostConfig` in `web/protocol/src/config.ts` (mirrors Go `internal/protocol/config.go`).

## Dev workflow

| Target | Command |
| ------ | ------- |
| Host UI against mock server | `npm run dev -w web/host` (proxies to `apps/mock-server`) |
| Participant UI against mock server | `npm run dev -w web/participant` |
| Live daemon | Build + run `spidercamd`; UIs served embedded at `/` |

Playwright tests use MSW to mock `/api` — no daemon required. See `e2e/`.
