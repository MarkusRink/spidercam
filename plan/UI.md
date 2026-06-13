# UI specification

Design principle: **every pixel shows mixer state** — dark panels, mono labels, green accent (`#3dd68c`). **Tailwind CSS v4** + shared theme ([ui/design-system.md](./ui/design-system.md)).

Both UIs are **SolidJS static SPAs** embedded in `spidercamd`. No media processing in the browser except participant WebRTC capture.

## Screens

| Screen       | URL                      | WS                                                        | Media                                        |
| ------------ | ------------------------ | --------------------------------------------------------- | -------------------------------------------- |
| Host console | `http://127.0.0.1:1235/` | `/api/v1/ws` — `host-state`; `/api/v1/ws/preview` — H.264 | Preview via WebCodecs (daemon pushes stream) |
| Participant  | `http://<lan>:1234/`     | `/api/v1/ws` — `participant-view`                         | `getUserMedia` + WebRTC → Pion               |

## Host console (1080p, no page scroll)

```
┌─────────────────────────────────────────────────────────────────────────┐
│ HEADER: spidercam/host · output ● · N streams · 142ms · DSP 4% · [copy] │
├──────────────────────────────────────────┬──────────────────────────────┤
│ OUTPUT PREVIEW (hard-cut video)          │ SETTINGS PANEL               │
│  OUT ▮  REF ▮  (−12.3 / −18.1 dBFS)     │ Mic / Webcam / Speaker ▾     │
│  clip LED top of each vertical meter     │ Hold · Crossfade · Ducking   │
│ [___LLLLHSSSSSLLLL________________]     │ Switch margin · weights      │
│  45 s state timeline                     │                              │
├──────────────────────────────────────────┴──────────────────────────────┤
│ STREAM GRID (fixed 168×240 cards, grid-cols-5, host + participants)     │
└─────────────────────────────────────────────────────────────────────────┘
```

No transport footer. No debug drawer. REF level only in preview panel (not on stream cards).

### State timeline (45 s)

Rolling strip driven by `selection.mixerState` (~20 Hz samples, bucketed for display). Each cell is one color:

| Symbol | `mixerState` | Color                                     |
| ------ | ------------ | ----------------------------------------- |
| `_`    | `SILENCE`    | `--color-spider-meter-track` (background) |
| `L`    | `LOCKED`     | `--color-spider-accent` (green)           |
| `H`    | `HOLD`       | `--color-spider-hold` (teal)              |
| `S`    | `SWITCH`     | `--color-spider-warn` (yellow)            |

Example: `[___LLLLHSSSSSLLLL__]`. Audio crossfade duration appears as a run of `S` cells.

### Host update loops

| Data                                        | Rate                                          | Source                                 |
| ------------------------------------------- | --------------------------------------------- | -------------------------------------- |
| Vertical meters (OUT, REF, stream cards)    | WS `host-state` ~20 Hz + rAF interpolate      | Daemon                                 |
| Card border opacity (`scoreSmooth`)         | same `host-state`                             | Daemon                                 |
| State timeline                              | append `mixerState` on each `host-state` tick | UI buffer (45 s)                       |
| Preview video                               | H.264 on `/api/v1/ws/preview` @ 15 fps        | Daemon compositor + `internal/preview` |
| Transport cells on cards                    | `host-state` @ 1 Hz (Pion stats)              | Daemon                                 |
| Loop delay text, global latency             | `host-state` @ ~0.3 Hz                        | Daemon passive loop                    |
| `enhancementBudgetPct`, `aecUs`/`denoiseUs` | `host-state` ~20 Hz                           | Daemon enhancement branch              |

SolidJS store subscribes to WS.

## Participant monitor (one viewport)

Single screen — no connect vs session routes. **Connect / Disconnect** is one toggle; local video preview, device pickers, and RMS meter are always visible.

| Zone    | Always                                                                       | When connected                                                             |
| ------- | ---------------------------------------------------------------------------- | -------------------------------------------------------------------------- |
| Header  | Editable display name (default `client-{random}`), muted **clientId** (UUID) | Red dot beside name when `activeAudioId === clientId`                      |
| Preview | Local camera + **AnalyserNode** meter                                        | Same                                                                       |
| Devices | Mic / camera dropdown + on/off toggles                                       | Same; track swap on change                                                 |
| Room    | —                                                                            | On air: `you` / `host` / name (text only); loop delay; SNR; transport grid |

**Lost host connection:** banner on the same screen when WS/WebRTC drops; exponential backoff **auto-reconnect** with auto-`join`; local preview stays up. Detail: [ui/participant-monitor.md](./ui/participant-monitor.md).

## Widget catalog

See [ui/design-system.md](./ui/design-system.md):

- **VerticalVuMeter** — current level bar + peak hold; red clip segment at top when `peakDbfs ≥ −1`
- **StateTimeline** — 45 s mixer-state history strip
- **LoopDelayText** — approximate ms on stream cards and participant session card
- **TransportBlock** — 2×3 mono grid: rtt, loss, jitter, buf, fps, A/V
- **On-air dot** — red (`--color-spider-error`); **host stream cards** when `activeAudioId === streamId`; **participant header** when `activeAudioId === clientId` (not on the “On air:” text row)
- **Score border** — card border opacity ∝ `scoreSmooth` (activity / energy)
- **StreamProcessingRow** — per-card AEC / NS toggles + timing when enabled

## Settings panel (host, always visible)

Right column — not an overlay. Devices + mixer tuning; session RAM only (see [domain/host-config.md](./domain/host-config.md)). WS `config` / `set-capture-devices` on change.
