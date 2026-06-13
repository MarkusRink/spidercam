# Spidercam Next Iteration — Implementation Plan

Companion to [`spidercam_next_iteration_e83e1b88.plan.md`](./spidercam_next_iteration_e83e1b88.plan.md). This document turns the architecture spec into concrete interfaces, file-level tasks, and code sketches. Snippets are valid TypeScript intended to compile once wired; `// DECISION:` comments mark gaps the planner must close.

---

## Delivery phases

| Phase  | Scope                                         | Unblocks                                  |
| ------ | --------------------------------------------- | ----------------------------------------- |
| **P0** | `apps/electron`, always-on host, seat removal | Real host shell, simplified protocol      |
| **P1** | Host UI v2 layout (stub metrics)              | Visual target, incremental DOM            |
| **P2** | Participant UI v2                             | End-user join flow                        |
| **P3** | Score selector + equal-power crossfade        | Mixer brain widgets with real data        |
| **P4** | Frame audio engine (10 ms loop)               | VAD, SNR, calibration, full stream strips |
| **P5** | AEC + limiter (optional polish)               | Teams echo reduction                      |

Phases P0–P2 can ship UI with **legacy level fields bridged** into new metric names; P3+ replaces routing math.

---

## Canonical audio math (research baseline)

All routing uses **normalized scores in [0, 1]** after feature scaling. Decision loop runs every **20 ms** (50 Hz); score EMA uses **β ≈ 0.1** (~200 ms smoothing).

### Per-frame metrics (10 ms @ 48 kHz, N = 480)

```ts
const EPS = 1e-8;

export function rmsDbfs(samples: Float32Array): number {
  let sum = 0;
  for (let i = 0; i < samples.length; i++) sum += samples[i] * samples[i];
  const rms = Math.sqrt(sum / samples.length);
  return 20 * Math.log10(rms + EPS);
}

export function snrDb(rms: number, noiseRms: number): number {
  return 20 * Math.log10((rms + EPS) / (noiseRms + EPS));
}

export function ema(prev: number, next: number, alpha: number): number {
  return (1 - alpha) * prev + alpha * next;
}
```

### Feature normalization → score components

```ts
export function normLevel(dbfs: number): number {
  return clamp((dbfs + 60) / 40, 0, 1); // [-60, -20] dBFS → [0, 1]
}

export function normSnr(snr: number): number {
  return clamp(snr / 20, 0, 1); // [0, 20] dB → [0, 1]
}

export function frameScore(
  components: ScoreComponents,
  w: ScoreWeights,
): number {
  return clamp(
    w.level * components.level +
      w.snr * components.snr +
      w.vad * components.vad +
      w.priority * components.priority -
      w.echoPenalty * components.echoPenalty,
    0,
    1,
  );
}

// Research defaults (§3.1, §4.3)
export const DEFAULT_SCORE_WEIGHTS: ScoreWeights = {
  level: 0.35,
  snr: 0.35,
  vad: 0.25,
  priority: 0.2,
  echoPenalty: 0.3,
};
```

### Hysteretic selection (§3.2)

Switch from current main $k$ to candidate $j = \arg\max_i \bar{S}_i$ only when:

1. $\bar{S}_j > \bar{S}_k + \Delta_{\text{switch}}$ with $\Delta_{\text{switch}} \approx 1.0$ (normalized), **or** emergency override when $\bar{S}_j \geq 3 \cdot \bar{S}_k$
2. Condition holds for $T_{\text{hold}}$ = 200–500 ms (`audioHoldMs`)
3. After switch, minimum hold 500 ms–1 s before another switch (unless 3× override)

**Silence:** all $\bar{S}_i$ below global threshold → `mixerState: "SILENCE"`, route default/host muted mix.

### Crossfade (§2.4)

Equal-power over 50–150 ms (`crossfadeMs`):

$$g_a(t) = \cos(\frac{\pi}{2} t),\quad g_b(t) = \sin(\frac{\pi}{2} t)$$

### Calibration (§1.1)

- Target speech level: **−20 dBFS** (`speechLevelDbfs` EMA during VAD=1, α_slow ≈ 0.95)
- $G_{\text{cal}} = L_{\text{target}} - \text{speech\_L}$, clamped **[−12, +18] dB**
- Apply $g_{\text{cal}}$ via very slow EMA (α ≈ 0.99)

### VAD (§1.2, §2.3)

- SNR_on ≈ 6–8 dB, SNR_off ≈ 3 dB (hysteresis)
- Band energy 300–3400 Hz + ZCR check
- Hangover **100–200 ms** after energy drop

### Host priority (§3.3)

- $H_{\text{host}} = 1.0$, $H_{\text{participant}} = 0.0$ in score
- When host VAD active and above threshold: **force host** as main (`forceHostWhenVad`) or boost by fixed margin

### Video coherence (§3.5)

Audio-driven: when audio main talker changes to participant A, switch video to A if feed usable and stable **> 100 ms**. Shorter hysteresis than audio (`videoHoldMs` can be lower than `audioHoldMs`).

### MVP enhancement stack (§6, recommended)

Per stream: HPF (~100 Hz) → WebRTC NS (level 1–2) → conservative compressor → analysis. **AEC host-only** where far-end reference exists. Master soft limiter on mixed output. RNNoise/GCC-PHAT deferred.

### v1 policies (planner should encode explicitly)

| Policy                      | Research recommendation       | Spidercam v1 default                                     |
| --------------------------- | ----------------------------- | -------------------------------------------------------- |
| Overlapping speech          | Single best vs dual −3 dB mix | **Single best talker**                                   |
| Silence output              | Muted vs low ambience         | **Near-silence** (heavy gate)                            |
| Orphan room speech          | No geometry                   | **Host priority + SNR**; duck participants when host VAD |
| Echo on participant streams | Down-weight correlated        | **echoPenalty** stub 0 until far-end ref exists          |

---

## P0 — Electron shell + always-on host + seat removal

### P0.1 New workspace `apps/electron`

```
apps/electron/
  package.json
  tsconfig.json
  src/
    main.ts
    preload.ts
    server-spawn.ts
    tray.ts
    lan-url.ts
```

**`apps/electron/package.json`**

```json
{
  "name": "@spidercam/electron",
  "version": "0.1.0",
  "private": true,
  "main": "dist/main.js",
  "scripts": {
    "build": "tsc",
    "start": "electron dist/main.js",
    "dev": "tsc && electron dist/main.js"
  },
  "dependencies": {
    "@spidercam/server": "*"
  },
  "devDependencies": {
    "electron": "^35.0.0",
    "typescript": "^5.8.0"
  }
}
```

**`apps/electron/src/server-spawn.ts`** — spawn in-process or child; DECISION: child process is safer for clean quit.

```ts
import { spawn, type ChildProcess } from "node:child_process";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

export interface ServerHandle {
  port: number;
  stop(): Promise<void>;
}

export async function startServer(webDist: string): Promise<ServerHandle> {
  // DECISION: import createApp from @spidercam/server and listen in-process
  // vs fork apps/server/dist/index.js. In-process avoids port races on fast restart.
  const { createApp } = await import("@spidercam/server");
  const port = Number(process.env.SPIDERCAM_PORT ?? 9847);
  const { httpServer } = createApp({ webDist });
  await new Promise<void>((resolve) =>
    httpServer.listen(port, "0.0.0.0", resolve),
  );
  return {
    port,
    stop: () =>
      new Promise((resolve, reject) => {
        httpServer.close((err) => (err ? reject(err) : resolve()));
      }),
  };
}
```

**`apps/electron/src/lan-url.ts`**

```ts
import os from "node:os";

export function participantUrl(port: number): string {
  const ifaces = os.networkInterfaces();
  const ip =
    Object.values(ifaces)
      .flat()
      .find((i) => i && i.family === "IPv4" && !i.internal)?.address ??
    "localhost";
  return `http://${ip}:${port}/`;
}
```

**`apps/electron/src/main.ts`**

```ts
import { app, BrowserWindow, clipboard, shell } from "electron";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { startServer } from "./server-spawn.js";
import { participantUrl } from "./lan-url.js";
import { createTray } from "./tray.js";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
let server: Awaited<ReturnType<typeof startServer>> | null = null;
let mainWindow: BrowserWindow | null = null;

async function createWindow(port: number): Promise<void> {
  mainWindow = new BrowserWindow({
    width: 1920,
    height: 1080,
    backgroundColor: "#0a0a0b",
    webPreferences: {
      preload: path.join(__dirname, "preload.js"),
      contextIsolation: true,
      nodeIntegration: false,
    },
  });

  const isDev = !app.isPackaged;
  const hostUrl = isDev
    ? `http://localhost:${port}/host.html` // vite dev proxy or built dist
    : `file://${path.join(__dirname, "../../web/dist/host.html")}`; // DECISION: file:// breaks WS unless server still HTTP

  // DECISION: always load via http://127.0.0.1:<port>/host.html so WS + getUserMedia share origin
  await mainWindow.loadURL(`http://127.0.0.1:${port}/host.html`);
}

app.whenReady().then(async () => {
  const webDist = path.resolve(__dirname, "../../web/dist");
  server = await startServer(webDist);
  await createWindow(server.port);

  createTray({
    url: participantUrl(server.port),
    onCopyUrl: () => clipboard.writeText(participantUrl(server!.port)),
    onQuit: () => app.quit(),
  });
});

app.on("before-quit", async () => {
  await server?.stop();
});
```

**`apps/electron/src/preload.ts`**

```ts
import { contextBridge, ipcRenderer } from "electron";

contextBridge.exposeInMainWorld("spidercam", {
  isElectronHost: true as const,
  copyParticipantUrl: () => ipcRenderer.invoke("copy-participant-url"),
});
```

**Root `package.json` change**

```json
{
  "scripts": {
    "start": "npm run build && npm run start -w @spidercam/electron"
  }
}
```

**Host route hardening** — `apps/server/src/create-app.ts`:

```ts
app.get("/host.html", (_req, res, next) => {
  // DECISION: 404 for non-loopback in production, or always serve for dev?
  const remote = req.socket.remoteAddress;
  const local =
    remote === "127.0.0.1" || remote === "::1" || remote === "::ffff:127.0.0.1";
  if (!local && process.env.SPIDERCAM_PUBLIC_HOST !== "1") {
    res.status(404).send("host dashboard is electron-only");
    return;
  }
  next();
});
```

### P0.2 Domain model — remove seats

**`packages/shared/src/types.ts`** (target)

```ts
export type ClientRole = "participant" | "host-mixer";
export type MixerState = "LOCKED" | "HOLD" | "SWITCH" | "SILENCE";
export type CalibrationPhase = "idle" | "measuring" | "applying" | "done";

export interface ParticipantInfo {
  id: string;
  name: string;
  role: ClientRole;
  hasVideo: boolean;
  hasAudio: boolean;
  deviceLabel?: string;
  joinedAt: number;
}

export interface ScoreComponents {
  level: number;
  snr: number;
  vad: number;
  priority: number;
  echoPenalty: number;
}

export interface StreamMetrics {
  participantId: string;
  name: string;
  role: "host" | "participant";

  rmsDbfs: number;
  peakDbfs: number;
  speechLevelDbfs: number;
  noiseFloorDbfs: number;
  snrDb: number;

  vad: boolean;
  vadHangoverMs: number;
  score: number;
  scoreComponents: ScoreComponents;
  rank: number;
  gateGainDb: number;
  duckingGainDb: number;
  calibrationGain: number;
  calibrationPhase: CalibrationPhase;
  jitterBufferFrames: number;
  delayOffsetMs: number;
  isMainTalker: boolean;

  videoActive: boolean;
  audioActive: boolean;
  rttMs: number | null;
  packetLoss: number | null;
  jitterMs: number | null;
  bitrateKbps: number | null;
  framesPerSecond: number | null;
  lastUpdated: number;

  /** @deprecated bridge during P1–P2; remove after frame engine */
  audioLevel?: number;
}

export interface CrossfadeState {
  fromId: string;
  toId: string;
  t: number; // 0..1
}

export interface SwitchEvent {
  at: number;
  fromId: string;
  toId: string;
}

export interface SelectionState {
  activeVideoId: string;
  activeAudioId: string;
  mainTalkerId: string;
  mixerState: MixerState;
  holdRemainingMs: number;
  crossfade: CrossfadeState | null;
  switchEvents: SwitchEvent[];
  reason: string;
  timestamp: number;
}

export interface RoomState {
  participants: ParticipantInfo[];
  metrics: StreamMetrics[];
  selection: SelectionState | null;
  hostOnline: boolean;
  globalLatencyMs: number;
  cpuPercent: number | null;
  outLevelDbfs: number;
}

export interface ScoreWeights {
  level: number;
  snr: number;
  vad: number;
  priority: number;
  echoPenalty: number;
}

export interface HostConfig {
  defaultVideoId: string;
  defaultAudioId: string;
  silenceScoreThreshold: number; // normalized; all S̄ below → SILENCE
  videoHoldMs: number; // audio-driven video; can be < audioHoldMs (§3.5)
  audioHoldMs: number; // T_hold 200–500 ms
  minHoldAfterSwitchMs: number; // 500–1000 ms before next switch
  crossfadeMs: number; // 50–150 ms
  switchMargin: number; // Δ_switch ≈ 1.0 normalized (NOT dB)
  emergencyScoreRatio: number; // 3× override (§3.2)
  scoreSmoothingAlpha: number; // β ≈ 0.1 for S̄ EMA
  scoreWeights: ScoreWeights;
  hostPriority: number; // 1.0 host, 0.0 participants in norm(priority)
  forceHostWhenVad: boolean;
  targetSpeechDbfs: number; // −20
  calibrationGainClampDb: [number, number]; // [-12, +18]
}

export const DEFAULT_HOST_CONFIG: HostConfig = {
  defaultVideoId: "host",
  defaultAudioId: "host",
  silenceScoreThreshold: 0.15,
  videoHoldMs: 300,
  audioHoldMs: 400,
  minHoldAfterSwitchMs: 600,
  crossfadeMs: 100,
  switchMargin: 1.0,
  emergencyScoreRatio: 3.0,
  scoreSmoothingAlpha: 0.1,
  scoreWeights: DEFAULT_SCORE_WEIGHTS,
  hostPriority: 1.0,
  forceHostWhenVad: true,
  targetSpeechDbfs: -20,
  calibrationGainClampDb: [-12, 18],
};

/** Stable id for host mixer stream in selector + mixer maps */
export const HOST_STREAM_ID = "host" as const;
```

**`packages/shared/src/messages.ts`** (target)

```ts
export type SignalingMessage =
  | { type: "welcome"; clientId: string; room: RoomState }
  | {
      type: "join";
      name: string;
      role: "participant" | "host-mixer";
      hasVideo: boolean;
      hasAudio: boolean;
      deviceLabel?: string;
    }
  | { type: "leave" }
  | { type: "room-update"; room: RoomState }
  | { type: "offer"; from: string; to: string; sdp: SessionDescription }
  | { type: "answer"; from: string; to: string; sdp: SessionDescription }
  | {
      type: "ice-candidate";
      from: string;
      to: string;
      candidate: IceCandidate | null;
    }
  | { type: "metrics"; from: string; metrics: Partial<StreamMetrics> }
  | { type: "selection"; selection: SelectionState }
  | { type: "config"; config: Partial<HostConfig> }
  | { type: "error"; message: string }
  | { type: "ping"; ts: number }
  | { type: "pong"; ts: number; serverTs: number };
```

**`apps/server/src/room.ts`** — drop `seat` from `updateMetrics` preserve logic; add room-level fields:

```ts
getState(): RoomState {
  return {
    participants: [...this.clients.values()].map((c) => c.info),
    metrics: [...this.metrics.values()],
    selection: this.selection,
    hostOnline: [...this.clients.values()].some((c) => c.info.role === "host-mixer"),
    globalLatencyMs: 0, // DECISION: max RTT or bridge-reported?
    cpuPercent: null,   // DECISION: Electron main only?
    outLevelDbfs: -60,
  };
}

addClient(id: string, ws: WebSocket, info: ParticipantInfo): void {
  this.metrics.set(id, createEmptyMetrics(id, info));
}

function createEmptyMetrics(id: string, info: ParticipantInfo): StreamMetrics {
  return {
    participantId: id,
    name: info.name,
    role: info.role === "host-mixer" ? "host" : "participant",
    rmsDbfs: -60,
    peakDbfs: -60,
    speechLevelDbfs: -60,
    noiseFloorDbfs: -60,
    snrDb: 0,
    vad: false,
    vadHangoverMs: 0,
    score: 0,
    scoreComponents: { level: 0, snr: 0, vad: 0, priority: 0, echoPenalty: 0 },
    rank: 0,
    gateGainDb: 0,
    duckingGainDb: 0,
    calibrationGain: 1,
    calibrationPhase: "idle",
    jitterBufferFrames: 0,
    delayOffsetMs: 0,
    isMainTalker: false,
    videoActive: info.hasVideo,
    audioActive: info.hasAudio,
    rttMs: null,
    packetLoss: null,
    jitterMs: null,
    bitrateKbps: null,
    framesPerSecond: null,
    lastUpdated: Date.now(),
  };
}
```

### P0.3 Always-on host — `dashboard.ts`

Delete `renderStart()`. On `DOMContentLoaded`, call `startHost()` immediately.

```ts
// apps/web/src/host/dashboard.ts
document.addEventListener("DOMContentLoaded", () => {
  void startHost();
});

async function startHost(): Promise<void> {
  config = { ...DEFAULT_HOST_CONFIG };
  hostStream = await navigator.mediaDevices.getUserMedia({
    video: true,
    audio: true,
  });
  // ... same pipeline, no seat/config form ...
  signaling.send({
    type: "join",
    name: "host",
    role: "host-mixer",
    hasVideo: true,
    hasAudio: true,
    deviceLabel: hostStream.getAudioTracks()[0]?.label,
  });
  // remove: signaling.send({ type: "config", config: { seatCount, hostSeat } });
}
```

### P0.4 Participant connect — remove seat UI

```ts
// apps/web/src/participant.ts — join payload
signaling.send({
  type: "join",
  name,
  role: "participant",
  hasVideo: useVideo,
  hasAudio: useAudio,
  deviceLabel: localStream?.getAudioTracks()[0]?.label,
});

// reportStats — drop seat arg
const metrics = peer
  ? await collectStats(peer.pc, signaling.clientId, audioLevel)
  : {
      participantId: signaling.clientId,
      audioLevel,
      audioActive: audioLevel > 0.02,
    };
```

### P0.5 Migration shim for metrics (P0–P2)

Until P4 ships, bridge legacy `audioLevel` (0..1 linear) into `rmsDbfs`:

```ts
// packages/shared/src/audio-math.ts
export function linearToDbfs(level: number): number {
  const v = Math.max(level, 1e-8);
  return 20 * Math.log10(v);
}

export function enrichLegacyMetrics(
  partial: Partial<StreamMetrics> & { audioLevel?: number },
): Partial<StreamMetrics> {
  if (partial.audioLevel != null && partial.rmsDbfs == null) {
    partial.rmsDbfs = linearToDbfs(partial.audioLevel);
  }
  return partial;
}
```

Use in `room.updateMetrics` and `collectStats`.

### P0.6 Files to touch (seat removal checklist)

| File                                   | Change                                                     |
| -------------------------------------- | ---------------------------------------------------------- |
| `packages/shared/src/types.ts`         | New shapes above                                           |
| `packages/shared/src/messages.ts`      | `join` without `seat`                                      |
| `packages/shared/src/selector.ts`      | Delete `seatDistance`, `findNearestConnected` (P3 rewrite) |
| `packages/shared/src/selector.test.ts` | Remove seat tests; stub until P3                           |
| `packages/shared/src/messages.test.ts` | Update join fixture                                        |
| `apps/server/src/signaling.ts`         | `join` handler                                             |
| `apps/server/src/room.ts`              | `getState`, `addClient`, `updateMetrics`                   |
| `apps/server/test/room.test.ts`        | No `seatCount`                                             |
| `apps/web/src/participant.ts`          | UI + join                                                  |
| `apps/web/src/host/dashboard.ts`       | Auto-start                                                 |
| `apps/web/src/webrtc/stats.ts`         | Drop `seat` param                                          |
| `e2e/*.spec.ts`                        | No start screen / seat                                     |
| `README.md`                            | Electron start, no seats                                   |

---

## P1 — Host UI v2 (layout + incremental DOM)

### P1.1 CSS grid zones

Add to `apps/web/src/styles/global.css`:

```css
.grid-host-v2 {
  display: grid;
  grid-template-rows: 40px 1fr 88px 120px;
  grid-template-columns: 1fr 360px;
  height: 100vh;
  overflow: hidden;
  gap: 8px;
  padding: 8px;
}

.host-header {
  grid-column: 1 / -1;
}
.host-output {
  grid-row: 2;
  grid-column: 1;
  min-height: 0;
}
.host-brain {
  grid-row: 2;
  grid-column: 2;
  min-height: 0;
  overflow: auto;
}
.host-rail {
  grid-column: 1 / -1;
  display: flex;
  gap: 6px;
  overflow-x: auto;
}
.host-footer {
  grid-column: 1 / -1;
  position: relative;
}

.stream-strip {
  flex: 0 0 72px;
  height: 72px;
  /* ... */
}

.vad-pill {
  /* active/inactive */
}
.score-stack {
  /* micro-bars */
}
.hold-badge {
  /* LOCKED | HOLD 420ms */
}
```

### P1.2 Dashboard controller split

```
apps/web/src/host/
  dashboard.ts          # bootstrap, signaling, peers
  dashboard-view.ts     # DOM shell + incremental updates
  dashboard-widgets.ts  # meter, sparkline, timeline helpers
```

**`dashboard-view.ts` interface**

```ts
import type {
  RoomState,
  SelectionState,
  StreamMetrics,
} from "@spidercam/shared";

export interface HostViewHandles {
  header: {
    setBridge(ok: boolean): void;
    setStreamCount(n: number): void;
    setOutLevel(dbfs: number): void;
    setLatency(ms: number): void;
  };
  output: {
    setCanvas(el: HTMLCanvasElement): void;
    setCrossfadeOverlay(active: boolean, t: number): void;
    pushOutSample(dbfs: number): void;
  };
  brain: {
    setSelection(sel: SelectionState): void;
    setMainTalker(id: string, name: string): void;
  };
  rail: {
    syncStreams(
      streams: StreamMetrics[],
      selection: SelectionState | null,
    ): void;
    expandStream(id: string | null): void;
  };
  footer: {
    syncTransportTable(metrics: StreamMetrics[]): void;
    setDebugOpen(open: boolean): void;
  };
}

export function mountHostView(root: HTMLElement): HostViewHandles {
  // Build static skeleton once; return handles that mutate textContent/styles
  // DECISION: no innerHTML on room-update — only patch changed nodes
  throw new Error("implement");
}
```

**Update loops** (from plan):

```ts
function startUiLoops(
  handles: HostViewHandles,
  getState: () => RoomState,
): void {
  const raf = () => {
    // meters, VAD pills, crossfade overlay
    requestAnimationFrame(raf);
  };
  raf();

  setInterval(() => {
    const room = getState();
    handles.brain.setSelection(room.selection!);
    handles.rail.syncStreams(room.metrics, room.selection);
  }, 100);

  setInterval(() => {
    // cpuPercent, globalLatencyMs — DECISION: source
  }, 500);
}
```

### P1.3 Stub extended metrics for strips

Until P4, populate from shim:

```ts
function stubStreamMetrics(m: StreamMetrics): StreamMetrics {
  return {
    ...m,
    snrDb: m.snrDb || 12,
    vad: m.rmsDbfs > linearToDbfs(DEFAULT_HOST_CONFIG.audioThreshold),
    score: m.score || Math.max(0, (m.rmsDbfs + 60) / 60),
    isMainTalker: false,
    calibrationPhase: "done",
    calibrationGain: 1,
  };
}
```

---

## P2 — Participant UI v2

### P2.1 Layout + on-air indicator

```ts
// apps/web/src/participant-view.ts
export interface ParticipantView {
  showConnect(): void;
  showSession(props: {
    name: string;
    roomSize: number;
    rmsDbfs: number;
    snrDb: number;
    vad: boolean;
    signalMs: number;
    onAirName: string | null;
    routed: boolean;
    calibrating: boolean;
  }): void;
}

function routingLabel(
  room: RoomState,
  myId: string,
): { onAir: string | null; routed: boolean } {
  const sel = room.selection;
  if (!sel) return { onAir: null, routed: false };
  const main = room.participants.find((p) => p.id === sel.mainTalkerId);
  const onAir =
    main?.name ?? (sel.mainTalkerId === HOST_STREAM_ID ? "host" : null);
  const routed = sel.activeAudioId === myId;
  return { onAir, routed };
}
```

**DECISION:** Participant sees `mainTalkerId` (new) vs `activeAudioId` (existing). Plan UI says "On air: Alice" — align on `mainTalkerId` for display, `activeAudioId` for "You: routed".

### P2.2 Connect screen

Remove `#seat`. Keep name, server URL, device toggles. Mobile `max-width: 420px` on `.grid-participant`.

---

## P3 — Score-based selector + equal-power crossfade

### P3.1 Selector rewrite

**`packages/shared/src/selector.ts`**

```ts
import type {
  HostConfig,
  SelectionState,
  StreamMetrics,
  MixerState,
} from "./types.js";
import { HOST_STREAM_ID } from "./types.js";

export interface SelectorState {
  lastAudioId: string;
  lastVideoId: string;
  lastAudioSwitch: number;
  lastVideoSwitch: number;
  holdUntil: number;
  switchLog: SelectionState["switchEvents"];
  crossfade: SelectionState["crossfade"];
}

export interface SelectorInput {
  config: HostConfig;
  streams: StreamMetrics[];
  now?: number;
  frameDtMs?: number;
}

export function createSelectorState(): SelectorState {
  return {
    lastAudioId: HOST_STREAM_ID,
    lastVideoId: HOST_STREAM_ID,
    lastAudioSwitch: 0,
    lastVideoSwitch: 0,
    holdUntil: 0,
    switchLog: [],
    crossfade: null,
  };
}

function normalizeScore(
  c: HostConfig["scoreWeights"],
  s: StreamMetrics,
): number {
  const w = c.scoreWeights;
  const comp = s.scoreComponents;
  const raw =
    w.level * comp.level +
    w.snr * comp.snr +
    w.vad * comp.vad +
    w.priority * comp.priority -
    w.echoPenalty * comp.echoPenalty;
  return Math.max(0, Math.min(1, raw));
}

function ranked(streams: StreamMetrics[], config: HostConfig): StreamMetrics[] {
  return [...streams]
    .map((s) => ({ ...s, score: normalizeScore(config, s) }))
    .sort((a, b) => b.score - a.score)
    .map((s, i) => ({ ...s, rank: i + 1 }));
}

export function selectSources(
  input: SelectorInput,
  state: SelectorState,
): SelectionState {
  const now = input.now ?? Date.now();
  const { config, streams } = input;
  const rankedStreams = ranked(streams, config);
  const top = rankedStreams[0];
  const current =
    rankedStreams.find((s) => s.participantId === state.lastAudioId) ??
    rankedStreams[0];

  let mixerState: MixerState = "SILENCE";
  let audioId = state.lastAudioId;
  let reason = "hold";

  const topScore = top?.score ?? 0;
  const currentScore = current?.score ?? 0;
  const aboveThreshold =
    (top?.rmsDbfs ?? -60) > linearToDbfs(config.audioThreshold);

  if (!aboveThreshold) {
    mixerState = "SILENCE";
    audioId = config.defaultAudioId;
    reason = "silence";
  } else if (top.participantId === state.lastAudioId) {
    mixerState = "LOCKED";
    reason = "locked";
  } else if (top.score - currentScore < config.switchMarginDb / 20) {
    // DECISION: switchMarginDb is dB-like but score is 0..1 — need unified units
    mixerState = "HOLD";
    reason = "margin";
  } else if (now < state.holdUntil) {
    mixerState = "HOLD";
    reason = "hold-timer";
  } else {
    mixerState = "SWITCH";
    audioId = top.participantId;
    state.holdUntil = now + config.audioHoldMs;
    state.lastAudioSwitch = now;
    state.switchLog.push({ at: now, fromId: state.lastAudioId, toId: audioId });
    if (state.switchLog.length > 50) state.switchLog.shift();
    state.crossfade = { fromId: state.lastAudioId, toId: audioId, t: 0 };
    reason = `switch to ${audioId}`;
  }

  // Video: DECISION — follow audio main talker, or independent videoHoldMs logic?
  const videoId =
    top && aboveThreshold && !config.forceHostWhenVad
      ? top.participantId
      : config.defaultVideoId;

  const holdRemainingMs = Math.max(0, state.holdUntil - now);
  const crossfade = advanceCrossfade(
    state,
    input.frameDtMs ?? 0,
    config.crossfadeMs,
  );

  state.lastAudioId = audioId;
  state.lastVideoId = videoId;

  return {
    activeVideoId: videoId,
    activeAudioId: audioId,
    mainTalkerId: top?.participantId ?? config.defaultAudioId,
    mixerState,
    holdRemainingMs,
    crossfade,
    switchEvents: [...state.switchLog],
    reason,
    timestamp: now,
  };
}

function advanceCrossfade(
  state: SelectorState,
  dtMs: number,
  crossfadeMs: number,
): SelectionState["crossfade"] {
  if (!state.crossfade) return null;
  const step = dtMs / crossfadeMs;
  state.crossfade.t = Math.min(1, state.crossfade.t + step);
  if (state.crossfade.t >= 1) {
    const done = state.crossfade;
    state.crossfade = null;
    return done;
  }
  return { ...state.crossfade };
}
```

**Unit tests to add** (`packages/shared/src/selector.test.ts`):

```ts
it("holds switch until margin and timer satisfied", () => {
  /* ... */
});
it("enters SILENCE when all below threshold", () => {
  /* ... */
});
it("advances crossfade t over frames", () => {
  /* ... */
});
```

### P3.2 Equal-power crossfade in mixer

**`apps/web/src/host/mixer.ts`**

```ts
export interface MixerSelection {
  audioId: string;
  crossfade: { fromId: string; toId: string; t: number } | null;
}

export class StreamMixer {
  private targetAudioId = "host";
  private crossfade: MixerSelection["crossfade"] = null;

  setSelection(videoId: string, selection: MixerSelection): void {
    this.videoId = videoId;
    this.targetAudioId = selection.audioId;
    this.crossfade = selection.crossfade;
    this.updateAudioGains();
  }

  private equalPowerGains(t: number): { a: number; b: number } {
    const theta = (t * Math.PI) / 2;
    return { a: Math.cos(theta), b: Math.sin(theta) };
  }

  private updateAudioGains(): void {
    const { crossfade, targetAudioId } = this;
    if (crossfade && crossfade.t < 1) {
      const { a, b } = this.equalPowerGains(crossfade.t);
      for (const [id, gain] of this.gainNodes) {
        if (id === crossfade.fromId) gain.gain.value = a;
        else if (id === crossfade.toId) gain.gain.value = b;
        else gain.gain.value = 0;
      }
      return;
    }
    for (const [id, gain] of this.gainNodes) {
      gain.gain.value = id === targetAudioId ? 1 : 0;
    }
  }
}
```

**DECISION:** `gain.gain.value` jumps are clicky; use `gain.gain.setTargetAtTime` in AudioContext clock for P3.2b.

### P3.3 Selector tick rate

Plan says 20 Hz selection broadcast; today `setInterval(..., 200)` = 5 Hz.

```ts
selectorInterval = setInterval(() => runSelector(), 50); // 20 Hz
```

Pass `frameDtMs: 50` into selector for crossfade progression.

---

## P4 — Frame audio engine (10 ms @ 48 kHz)

### P4.1 Module layout

```
apps/web/src/host/audio/
  audio-engine.ts       # owns AudioContext graph, fan-out to processors
  stream-processor.ts   # per-stream state machine
  vad.ts
  calibration.ts
  noise-floor.ts
  score.ts
  ring-buffer.ts
  audio-worklet-processor.ts  # optional; start with ScriptProcessorNode deprecated path
```

**Constants**

```ts
export const SAMPLE_RATE = 48_000;
export const FRAME_SAMPLES = 480; // 10 ms
export const FRAME_MS = 10;
```

### P4.2 Per-stream processor interface

```ts
// apps/web/src/host/audio/stream-processor.ts
export interface StreamProcessorConfig {
  targetSpeechDbfs: number; // -20
  noiseAlpha: number; // EMA for noise floor
  vadHangoverMs: number; // 200
  hostPriority: number;
}

export interface StreamFrameInput {
  participantId: string;
  name: string;
  role: "host" | "participant";
  pcm: Float32Array; // length FRAME_SAMPLES
  timestampMs: number;
}

export class StreamProcessor {
  constructor(private cfg: StreamProcessorConfig) {}

  processFrame(input: StreamFrameInput): StreamMetrics {
    const rms = rmsDbfs(input.pcm);
    const peak = peakDbfs(input.pcm);
    // noise floor, vad, snr, calibration, scoreComponents...
    return {
      participantId: input.participantId,
      name: input.name,
      role: input.role,
      rmsDbfs: rms,
      peakDbfs: peak,
      // ...
      lastUpdated: input.timestampMs,
    };
  }
}

function rmsDbfs(samples: Float32Array): number {
  let sum = 0;
  for (let i = 0; i < samples.length; i++) sum += samples[i] * samples[i];
  const rms = Math.sqrt(sum / samples.length);
  return 20 * Math.log10(Math.max(rms, 1e-8));
}
```

### P4.3 Audio engine orchestration

```ts
// apps/web/src/host/audio/audio-engine.ts
export interface AudioEngineCallbacks {
  onMetrics(streams: StreamMetrics[]): void;
  onOutLevel(dbfs: number): void;
}

export class AudioEngine {
  private processors = new Map<string, StreamProcessor>();

  attachStream(
    id: string,
    stream: MediaStream,
    meta: Pick<StreamMetrics, "name" | "role">,
  ): void {
    // MediaStreamSource -> ScriptProcessorNode(480, 1) -> silent destination
    // DECISION: AudioWorklet preferred; ScriptProcessor for MVP with eslint waiver
  }

  detachStream(id: string): void {
    this.processors.delete(id);
  }

  start(callbacks: AudioEngineCallbacks): void {
    // 10 ms callbacks aggregate all processors -> onMetrics at 100 Hz
    // DECISION: throttle broadcast to 20 Hz for WS, UI at 100 Hz local
  }

  stop(): void {}
}
```

### P4.4 Dashboard integration

Replace `createAudioLevelMonitor` + `peerAudioLevels` map with `AudioEngine`:

```ts
audioEngine = new AudioEngine(config);
audioEngine.attachStream(HOST_STREAM_ID, hostStream, {
  name: "host",
  role: "host",
});
audioEngine.start({
  onMetrics: (streams) => {
    hostMetrics = streams;
    runSelector(streams);
  },
  onOutLevel: (dbfs) => {
    outLevelDbfs = dbfs;
    view.header.setOutLevel(dbfs);
  },
});
```

### P4.5 VAD sketch (underspecified in plan)

```ts
// apps/web/src/host/audio/vad.ts
export interface VadState {
  active: boolean;
  hangoverRemainingMs: number;
}

export function updateVad(
  state: VadState,
  snrDb: number,
  bandEnergy: number,
  dtMs: number,
  opts: { snrOn: number; snrOff: number; hangoverMs: number },
): VadState {
  const wantOn = snrDb >= opts.snrOn && bandEnergy > 0; // DECISION: bandEnergy definition
  if (wantOn) return { active: true, hangoverRemainingMs: opts.hangoverMs };
  if (state.active) {
    const next = state.hangoverRemainingMs - dtMs;
    return next > 0
      ? { active: true, hangoverRemainingMs: next }
      : { active: false, hangoverRemainingMs: 0 };
  }
  return { active: false, hangoverRemainingMs: 0 };
}
```

---

## P5 — AEC + master limiter (deferred polish)

```ts
// apps/web/src/host/audio/limiter.ts
export function createSoftLimiter(ctx: AudioContext): DynamicsCompressorNode {
  const comp = ctx.createDynamicsCompressor();
  comp.threshold.value = -6;
  comp.knee.value = 12;
  comp.ratio.value = 8;
  comp.attack.value = 0.003;
  comp.release.value = 0.25;
  return comp;
}

// AEC: DECISION — WebRTC insertable streams vs playback reference tap only on host
```

Wire limiter between mixer destination and `BridgeClient.attachAudio`.

---

## Signaling + broadcast strategy

| Data                   | Rate      | Channel                                   |
| ---------------------- | --------- | ----------------------------------------- |
| Full `StreamMetrics[]` | 20 Hz     | host → `metrics` (self) + local UI 100 Hz |
| `SelectionState`       | 20 Hz     | host → `selection`                        |
| Transport stats (RTT…) | 1 Hz      | participants → `metrics` (unchanged)      |
| `RoomState`            | on change | server `room-update`                      |

**DECISION:** Today every `metrics` message triggers `room-update` to all clients. At 20 Hz × N streams this is heavy. Options:

1. New `mixer-state` message with compact payload for host-only fields.
2. Server-side throttle of `room-update` (max 10/s).
3. Participants subscribe to reduced `ParticipantRoomView` without full metrics array.

```ts
// Option 3 — packages/shared/src/types.ts
export interface ParticipantRoomView {
  participants: Pick<
    ParticipantInfo,
    "id" | "name" | "hasVideo" | "hasAudio"
  >[];
  selection: SelectionState | null;
  selfMetric: Pick<
    StreamMetrics,
    "rmsDbfs" | "snrDb" | "vad" | "calibrationPhase"
  >;
}
```

---

## Testing plan

### Unit

| File                       | Cases                                                 |
| -------------------------- | ----------------------------------------------------- |
| `selector.test.ts`         | hysteresis, silence, crossfade advance, host priority |
| `audio-math.test.ts`       | linearToDbfs, equalPowerGains                         |
| `stream-processor.test.ts` | noise floor only updates when !vad                    |
| `messages.test.ts`         | join without seat                                     |

### Integration

| File                            | Cases                      |
| ------------------------------- | -------------------------- |
| `room.test.ts`                  | `hostOnline`, no seatCount |
| `signaling.integration.test.ts` | join/leave without seat    |

### E2E

```ts
// e2e/host.spec.ts — Electron skipped in CI; test host.html auto-start in browser
test("auto-starts dashboard", async ({ page }) => {
  await page.goto("/host.html");
  await expect(page.getByText("spidercam / host")).toBeVisible({
    timeout: 15_000,
  });
  await expect(page.locator(".host-header")).toBeVisible();
  // no #startBtn
});

// e2e/participant.spec.ts
test("connects without seat", async ({ page }) => {
  await page.locator("#name").fill("e2e-user");
  await page.locator("#connectBtn").click();
  await expect(page.getByText(/on air/i)).toBeVisible();
});
```

**DECISION:** CI does not run Electron today. Add `e2e/electron.spec.ts` with `test.skip(!process.env.ELECTRON_E2E)` or smoke-test `apps/electron` via Playwright `_electron` launcher.

---

## Root scripts & build order

```json
{
  "scripts": {
    "build": "npm run build --workspaces --if-present",
    "start": "npm run build && npm run start -w @spidercam/electron",
    "dev": "npm run dev -w @spidercam/server",
    "dev:web": "npm run dev -w @spidercam/web",
    "dev:electron": "npm run build -w @spidercam/web && npm run dev -w @spidercam/electron"
  }
}
```

Build order: `shared` → `server` → `web` → `electron`.

---

## Open decisions register

| ID  | Question                                 | Status / default                                                                 |
| --- | ---------------------------------------- | -------------------------------------------------------------------------------- |
| D1  | Host stream id: `"host"` vs UUID         | **Resolved:** `HOST_STREAM_ID = "host"`                                          |
| D2  | Switch margin units                      | **Resolved:** `switchMargin` normalized (≈1.0), not dB                           |
| D3  | Video vs audio coupling                  | **Resolved:** audio-driven; video stable >100 ms (§3.5)                          |
| D4  | `mainTalkerId` during crossfade          | **Resolved:** `mainTalkerId` = selected target; mixer crossfades `activeAudioId` |
| D5  | `cpuPercent` source                      | Open — Electron IPC                                                              |
| D6  | `globalLatencyMs`                        | Open — jitter buffer + max RTT; target E2E 80–150 ms                             |
| D7  | Host page over LAN                       | Open — 404 non-loopback                                                          |
| D8  | `config` WS message                      | **Resolved:** keep for runtime tuning                                            |
| D9  | ScriptProcessor vs AudioWorklet          | Open — Worklet preferred; ScriptProcessor MVP                                    |
| D10 | Orphan / room speech                     | **Resolved:** host priority + ducking; no seat proxy                             |
| D11 | Signaling at 20 Hz full metrics          | Open — throttle or `ParticipantRoomView`                                         |
| D12 | Pull-based frame graph vs push analyzers | Open — research wants pull; current code is push                                 |
| D13 | `echoPenalty` without far-end ref        | Open — stub 0 in MVP; host AEC in P5                                             |
| D14 | Gate/ducking in mixer vs score only      | Open — research has both; plan UI shows `gateGainDb` / `duckingGainDb`           |
| D15 | CI Electron vs browser host fallback     | Open                                                                             |

---

## Suggested PR sequence

1. `feat/electron-shell` — P0.1, root start script
2. `refactor/remove-seats` — P0.2–P0.6, all tests green
3. `feat/always-on-host` — dashboard auto-start
4. `feat/host-ui-v2` — layout + incremental DOM (stub metrics)
5. `feat/participant-ui-v2` — connect + on-air
6. `feat/score-selector-crossfade` — P3 + mixer
7. `feat/frame-audio-engine` — P4
8. `feat/limiter-aec` — P5 optional

Each PR runs `npm run lint`, `npm run format`, `npm run check`.
