# Host console

**Target:** `web/host/` — SolidJS + Tailwind, served on `:1235` (loopback).

Opened by **`spidercamd`** via system browser (`xdg-open`). Operator may re-open `http://127.0.0.1:1235/` manually.

## File layout

```
web/host/
  package.json
  vite.config.ts
  index.html
  src/
    main.tsx
    App.tsx
    signaling.ts
    preview-stream.ts
    stores/session-store.ts
    components/
      Header.tsx
      OutputPreview.tsx
      SettingsPanel.tsx
      StateTimeline.tsx
      StreamGrid.tsx
      StreamCard.tsx
      TransportBlock.tsx
    index.css
```

## Signaling

```ts
export class HostSignaling {
  private ws: WebSocket;

  connect(): Promise<void> {
    this.ws = new WebSocket(`ws://${location.host}/api/v1/ws`);
  }

  onState(handler: (state: RoomState) => void): void {
    /* host-state */
  }

  sendConfig(partial: Partial<HostConfig>): void {
    this.ws.send(JSON.stringify({ type: "config", config: partial }));
  }

  listCaptureDevices(): void {
    this.ws.send(JSON.stringify({ type: "list-capture-devices" }));
  }

  setCaptureDevices(sel: CaptureSelection): void {
    this.ws.send(
      JSON.stringify({ type: "set-capture-devices", selection: sel }),
    );
  }

  setStreamProcessing(
    participantId: string,
    flags: StreamProcessingFlags,
  ): void {
    this.ws.send(
      JSON.stringify({
        type: "set-stream-processing",
        participantId,
        flags,
      }),
    );
  }
}
```

Handle inbound: `capture-devices`, `capture-devices-updated`, `host-state`.

On mount: `listCaptureDevices()` once for dropdown options.

## Session store

```ts
export interface SessionStore {
  state: RoomState | null;
  captureDevices: CaptureDevices | null;
  timeline: MixerState[]; // rolling 45 s of mixerState samples
}
```

SolidJS `createStore` — update on each `host-state` (~20 Hz). Interpolate meter levels on rAF between snapshots. Append `state.selection?.mixerState` to `timeline` each tick; trim to 45 s window.

## App layout

```tsx
<div class="grid h-screen grid-rows-[40px_minmax(0,1fr)_260px] grid-cols-[1fr_360px] gap-2 overflow-hidden p-2">
  <Header class="col-span-2" />
  <OutputPreview />
  <SettingsPanel />
  <StreamGrid class="col-span-2 min-h-0 overflow-y-auto" />
</div>
```

Right column is **SettingsPanel** (always visible). No transport footer, no debug drawer.

## Output preview

Two WebSockets on mount:

| Socket  | Path                 | Content                                        |
| ------- | -------------------- | ---------------------------------------------- |
| Control | `/api/v1/ws`         | `host-state` @ 50 Hz — meters, timeline, cards |
| Preview | `/api/v1/ws/preview` | H.264 @ 15 fps — WebCodecs → `<canvas>`        |

`PreviewStream` ([architecture/preview.md](../architecture/preview.md)) decodes binary chunks; on `preview-cut` clears canvas until next keyframe. Video is **hard-cut** in the daemon compositor (IDR forced on `activeVideoId` change).

Beside the preview canvas:

| Meter | Source                                    | Label                   |
| ----- | ----------------------------------------- | ----------------------- |
| OUT   | `outLevelDbfs`, `outPeakDbfs`             | Teams virtual mic level |
| REF   | `reference.rmsDbfs`, `reference.peakDbfs` | Playback reference      |

Both use **VerticalVuMeter** with numeric dBFS below. Clip LED at top when peak ≥ −1 dBFS.

Below preview + meters: **StateTimeline** (45 s, see [UI.md](../UI.md)).

## Header

- `outputHealthy` dot
- Stream count
- `globalLatencyMs` — single integer or `—`
- `enhancementBudgetPct` — when any stream has AEC or NS on (e.g. `DSP 4%`); green / amber / red tiers
- Participant URL copy button

No OUT/REF dBFS in header (moved to preview panel).

## Settings panel

Always visible in right column. Changes apply immediately via WS; **session RAM only** — no disk persistence (D15).

### Devices

| Control         | Binds to                    | Note                                         |
| --------------- | --------------------------- | -------------------------------------------- |
| Microphone      | `CaptureSelection.micId`    | PW sources                                   |
| Webcam          | `CaptureSelection.cameraId` | v4l2 paths                                   |
| Playback output | `CaptureSelection.sinkId`   | PW sinks — Teams speaker; drives REF capture |

On change: `setCaptureDevices()` → daemon `capture.Reopen`.

Hint under speaker: “Teams meeting audio should play to this device.”

### Mixer

| Control       | Key               | Range                  |
| ------------- | ----------------- | ---------------------- |
| Hold time     | `audioHoldMs`     | 200–800 ms             |
| Crossfade     | `crossfadeMs`     | 50–200 ms (audio only) |
| Ducking       | `referenceDuckDb` | 0 … −12 dB (`0` = off) |
| Switch margin | `switchMargin`    | 0.5–2.0                |

AEC and RNNoise are **per-stream** on cards — not in this panel.

### Score weights

| Control      | Key                        | Range |
| ------------ | -------------------------- | ----- |
| Level        | `scoreWeights.level`       | 0–1   |
| SNR          | `scoreWeights.snr`         | 0–1   |
| VAD          | `scoreWeights.vad`         | 0–1   |
| Priority     | `scoreWeights.priority`    | 0–1   |
| Echo penalty | `scoreWeights.echoPenalty` | 0–1   |

Sliders debounced ~150 ms → `sendConfig(partial)`.

No preset buttons.

## Stream grid

Fixed-layout grid of **StreamCard** components. Order: **host** first, then participants by `joinedAt`. **No `playback-ref` card.**

```tsx
<div class="grid grid-cols-5 gap-2 content-start font-mono">
  <For each={orderedStreams()}>{(m) => <StreamCard metric={m} />}</For>
</div>
```

- **5 columns** hard-coded (`grid-cols-5`); each card **168×240 px** fixed (`w-[168px] h-[240px] shrink-0`).
- More than five streams → wraps to additional rows; **stream zone scrolls** (`overflow-y-auto`) without changing card size.
- Alternative acceptable: `flex flex-wrap` with the same fixed `w-[168px] h-[240px] shrink-0 grow-0` per card.

### StreamCard (always full layout)

All cards use the same template — no expand/collapse.

| Zone       | Content                                                                                |
| ---------- | -------------------------------------------------------------------------------------- |
| Header row | **On-air dot** (red when `activeAudioId === participantId`) + truncated **name**       |
| Meter      | **VerticalVuMeter** (compact) + RMS dBFS                                               |
| Loop       | **LoopDelayText** — `~118 ms` or `—` (no uncertainty)                                  |
| Processing | **AEC** / **NS** toggles + timing lines when on (`AEC · 0.4ms`, `NS · 0.2ms`)          |
| Transport  | **TransportBlock** 2×3 grid (see below)                                                |
| Chrome     | Border opacity ∝ `scoreSmooth` (0.15…1.0) — activity/energy, independent of on-air dot |

Typography: **mono only** inside the card (`font-mono`, `tabular-nums`).

### Stream processing toggles

Small labeled toggles on each card (host + participants):

| Toggle | WS field               | Default |
| ------ | ---------------------- | ------- |
| AEC    | `flags.aecEnabled`     | off     |
| NS     | `flags.denoiseEnabled` | off     |

On change: `setStreamProcessing(participantId, flags)`. Card meters (`rmsDbfs`) show **post-enhancement** level.

When enabled, show EMA timing below toggles from `aecUs` / `denoiseUs`. When off, hide timing line (not `0ms`).

### TransportBlock — participant

|       |        |          |
| ----- | ------ | -------- |
| `rtt` | `loss` | `jitter` |
| `buf` | `fps`  | `A/V`    |

Formats: `42ms`, `0.2%`, `8ms`, `3`, `29fps`, `AV` / `A-` / `-V` / `--`.

Warn/error coloring on individual values (see [design-system.md](./design-system.md)).

Updates @ **1 Hz** (Pion transport stats).

### TransportBlock — host (local capture)

Same 2×3 cell layout; WebRTC fields not applicable:

|     |       |       |
| --- | ----- | ----- |
| `—` | `—`   | `—`   |
| `—` | `fps` | `A/V` |

`fps` from v4l2 when available; `A/V` from `audioActive` / `videoActive`.

## Latency UX

Loop delay text updates on passive-loop publish cadence (~3 s), not with 50 Hz meters.

## Diagnosis without debug drawer

Routing history: **45 s state timeline** only. Per-stream health: **cards** (meter, transport, loop text, score border). Deep logs: **daemon terminal** — no in-UI JSON drawer (D15).
