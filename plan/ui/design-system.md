# Design system

**Target:** `web/ui-theme/` (shared) · consumed by `web/host` and `web/participant`

Styling is **Tailwind CSS v4** utility-first. No hand-written widget CSS files. Repeated patterns live in SolidJS components (`web/host`, `web/participant`).

Use CSS-variable shorthand in class strings: `bg-(--color-spider-surface)` not `bg-[var(--color-spider-surface)]`.

## Package layout

```
web/ui-theme/
  package.json
  src/
    spidercam.css
    VerticalVuMeter.tsx
    StateTimeline.tsx
    LoopDelayText.tsx
    TransportBlock.tsx
    StreamProcessingRow.tsx
    StreamCard.tsx
```

## web/ui-theme/src/spidercam.css

```css
@theme {
  --color-spider-bg: #0a0a0b;
  --color-spider-surface: #111113;
  --color-spider-border: #2a2a2e;
  --color-spider-text: #e8e8ea;
  --color-spider-muted: #6b6b70;
  --color-spider-accent: #3dd68c;
  --color-spider-hold: #2dd4bf;
  --color-spider-warn: #f5a623;
  --color-spider-error: #ff5f57;
  --color-spider-meter-track: #1a1a1e;

  --font-mono: "JetBrains Mono", "SF Mono", "Fira Code", ui-monospace, monospace;
  --font-sans: system-ui, -apple-system, sans-serif;

  --radius-spider: 3px;
  --radius-spider-sm: 2px;

  --stream-card-w: 168px;
  --stream-card-h: 240px;
}
```

## App entry CSS

Each app imports Tailwind once:

```css
/* web/host/src/index.css  |  web/participant/src/index.css */
@import "tailwindcss";
@import "../../ui-theme/spidercam.css";

@layer base {
  body {
    @apply bg-(--color-spider-bg) text-(--color-spider-text) font-sans antialiased;
  }
}
```

## Layout primitives

Host root (`App.tsx`):

```tsx
<div class="grid h-screen grid-rows-[40px_minmax(0,1fr)_260px] grid-cols-[1fr_360px] gap-2 overflow-hidden p-2">
  <Header class="col-span-2" />
  <OutputPreview />
  <SettingsPanel />
  <StreamGrid class="col-span-2 min-h-0 overflow-y-auto" />
</div>
```

Stream grid:

```tsx
<div class="grid grid-cols-5 gap-2 content-start font-mono">
  <For each={streams}>{(m) => <StreamCard metric={m} selection={selection} />}</For>
</div>
```

Participant shell — single card, centered:

```tsx
// web/participant/src/components/ParticipantShell.tsx
<div class="flex min-h-screen items-center justify-center p-4">
  <div class="w-full max-w-[420px] space-y-3">
    <ParticipantHeader />   {/* display name + routed dot + clientId */}
    <LocalPreview />        {/* video + local VuMeter — always */}
    <DeviceRows />          {/* mic/cam select + toggles — always */}
    <ConnectionToggle />
    <Show when={phase === "connected"}><RoomStatus /></Show>
    <Show when={phase === "reconnecting" || phase === "lost"}><LostHostBanner /></Show>
  </div>
</div>
```

### On-air dot (host vs participant)

**Host stream cards** — red dot in card header when `activeAudioId === streamId`:

```tsx
<span
  class="h-2 w-2 shrink-0 rounded-full bg-(--color-spider-error)"
  classList={{ invisible: !props.onAir }}
  title="On air"
/>
```

**Participant monitor** — red dot in **page header** beside display name when `activeAudioId === clientId` (routed to Teams). The **On air:** status row is text only (`you` / `host` / display name from `mainTalkerId`).

```tsx
// web/participant/src/components/ParticipantHeader.tsx
<div class="flex items-center gap-2">
  <input class="flex-1 …" value={displayName()} onInput={…} />
  <OnAirDot onAir={isRouted(view(), clientId())} title="Routed to Teams" />
</div>
<p class="font-mono text-[10px] text-(--color-spider-muted) truncate">{clientId()}</p>
```

Independent of on-air dot on host cards: a challenger can have a bright score border during HOLD without being routed.

## Widget components

### Vertical VU meter

Typical desk-mixer meter: fill from bottom (current RMS), horizontal tick for peak hold, **red clip segment at top** when `peakDbfs ≥ −1`.

```tsx
// web/ui-theme/src/VerticalVuMeter.tsx
export function VerticalVuMeter(props: {
  rmsDbfs: number;
  peakDbfs: number;
  label?: string;
  compact?: boolean;
}) {
  const levelPct = (db: number) => clamp((db + 60) / 60, 0, 1) * 100;
  const clipped = props.peakDbfs >= -1;

  return (
    <div class="flex flex-col items-center gap-1">
      <div
        class="relative w-3 rounded-(--radius-spider-sm) bg-(--color-spider-meter-track) overflow-hidden h-14"
      >
        {clipped && (
          <div class="absolute top-0 inset-x-0 h-1 bg-(--color-spider-error)" />
        )}
        <div
          class="absolute bottom-0 inset-x-0 bg-(--color-spider-accent) transition-[height] duration-[16ms] linear"
          style={{ height: `${levelPct(props.rmsDbfs)}%` }}
        />
        <div
          class="absolute inset-x-0 h-px bg-(--color-spider-warn)"
          style={{ bottom: `${levelPct(props.peakDbfs)}%` }}
        />
      </div>
      <span class="font-mono text-[10px] tabular-nums text-(--color-spider-muted)">
        {props.label ?? `${props.rmsDbfs.toFixed(1)}`}
      </span>
    </div>
  );
}
```

Host preview: taller variant (`h-24`) for OUT + REF side by side.

### State timeline (45 s)

```tsx
// web/ui-theme/src/StateTimeline.tsx
const TIMELINE_COLORS: Record<MixerState, string> = {
  SILENCE: "var(--color-spider-meter-track)",
  LOCKED: "var(--color-spider-accent)",
  HOLD: "var(--color-spider-hold)",
  SWITCH: "var(--color-spider-warn)",
};

export function StateTimeline(props: { samples: MixerState[] }) {
  return (
    <div class="flex h-2 w-full gap-px rounded-(--radius-spider-sm) overflow-hidden" role="img">
      <For each={props.samples}>
        {(s) => (
          <div class="flex-1 min-w-0" style={{ "background-color": TIMELINE_COLORS[s] }} />
        )}
      </For>
    </div>
  );
}
```

### On-air dot

```tsx
<span
  class="h-2 w-2 shrink-0 rounded-full bg-(--color-spider-error)"
  classList={{ invisible: !props.onAir }}
  title="On air"
/>
```

Shown when `activeAudioId === streamId` on **host stream cards**. Red — routed to Teams, not top rank.

### Score border (activity)

Card border opacity reflects `scoreSmooth` (0…1):

```tsx
const opacity = 0.15 + 0.85 * clamp(props.scoreSmooth, 0, 1);

<div
  class="rounded-(--radius-spider) border bg-(--color-spider-surface)"
  style={{
    width: "var(--stream-card-w)",
    height: "var(--stream-card-h)",
    "border-color": `color-mix(in srgb, var(--color-spider-accent) ${opacity * 100}%, var(--color-spider-border))`,
  }}
/>
```

Independent of on-air dot: a challenger can have a bright border during HOLD without being routed.

### Loop delay text

```tsx
// web/ui-theme/src/LoopDelayText.tsx
export function LoopDelayText(props: { estimate: LoopDelayEstimate }) {
  const known = props.estimate.known && props.estimate.ms != null;
  const ms = props.estimate.ms ?? 0;
  const color =
    !known ? "var(--color-spider-muted)"
    : ms < 100 ? "var(--color-spider-accent)"
    : ms <= 150 ? "var(--color-spider-warn)"
    : "var(--color-spider-error)";

  return (
    <span class="font-mono text-[10px] tabular-nums" style={{ color }}>
      {known ? `~${ms} ms` : "—"}
    </span>
  );
}
```

No uncertainty displayed.

### Transport block

Fixed 2×3 mono grid inside each stream card. Participant: live WebRTC stats. Host: `—` for rtt/loss/jitter/buf.

```tsx
// web/ui-theme/src/TransportBlock.tsx
type TransportCell = { text: string; tone?: "warn" | "error" };

function toneClass(tone?: string) {
  if (tone === "error") return "text-(--color-spider-error)";
  if (tone === "warn") return "text-(--color-spider-warn)";
  return "text-(--color-spider-muted)";
}

export function TransportBlock(props: { cells: TransportCell[] }) {
  return (
    <div class="grid grid-cols-3 gap-x-1 gap-y-0.5 text-[9px] tabular-nums">
      <For each={props.cells}>
        {(c) => <span class={toneClass(c.tone)}>{c.text}</span>}
      </For>
    </div>
  );
}
```

**Thresholds** (participant):

| Metric | Warn | Error |
|--------|------|-------|
| `packetLoss` | &gt; 1% | &gt; 3% |
| `jitterMs` | &gt; 20 ms | &gt; 40 ms |
| `jitterBufferFrames` | &gt; 5 | &gt; 10 |
| `framesPerSecond` | &lt; 20 | &lt; 10 or missing |

### Stream processing row

```tsx
// web/ui-theme/src/StreamProcessingRow.tsx
export function StreamProcessingRow(props: {
  metric: StreamMetrics;
  onChange?: (flags: StreamProcessingFlags) => void;
}) {
  return (
    <div class="flex flex-col gap-0.5 text-[9px]">
      <div class="flex gap-2">
        <label class="flex items-center gap-1">
          <input type="checkbox" checked={props.metric.aecEnabled}
            onChange={(e) => props.onChange?.({ ...flags, aecEnabled: e.currentTarget.checked })} />
          AEC
        </label>
        <label class="flex items-center gap-1">
          <input type="checkbox" checked={props.metric.denoiseEnabled}
            onChange={(e) => props.onChange?.({ ...flags, denoiseEnabled: e.currentTarget.checked })} />
          NS
        </label>
      </div>
      <Show when={props.metric.aecEnabled}>
        <span class="text-(--color-spider-muted)">AEC · {formatMs(props.metric.aecUs)}</span>
      </Show>
      <Show when={props.metric.denoiseEnabled}>
        <span class="text-(--color-spider-muted)">NS · {formatMs(props.metric.denoiseUs)}</span>
      </Show>
    </div>
  );
}
```

### Stream card

```tsx
// web/ui-theme/src/StreamCard.tsx — fixed 168×240, always expanded
export function StreamCard(props: {
  metric: StreamMetrics;
  isOnAir: boolean;
  isHost: boolean;
  onProcessingChange?: (id: string, flags: StreamProcessingFlags) => void;
}) {
  return (
    <div
      class="flex flex-col gap-1 p-2 font-mono shrink-0 grow-0"
      style={{
        width: "var(--stream-card-w)",
        height: "var(--stream-card-h)",
        /* score border via opacity — see above */
      }}
    >
      <div class="flex items-center gap-1 min-w-0">
        <OnAirDot onAir={props.isOnAir} />
        <span class="truncate text-[10px]">{props.metric.name}</span>
      </div>
      <VerticalVuMeter rmsDbfs={props.metric.rmsDbfs} peakDbfs={props.metric.peakDbfs} compact />
      <LoopDelayText estimate={props.metric.loopDelay} />
      <StreamProcessingRow
        metric={props.metric}
        onChange={(flags) => props.onProcessingChange?.(props.metric.participantId, flags)}
      />
      <TransportBlock cells={transportCells(props.metric, props.isHost)} />
    </div>
  );
}
```

No VAD pill. No score stack. No expand/collapse.

## Typography utilities

| Pattern | Classes |
|---------|---------|
| Mono label | `font-mono text-[10px] text-(--color-spider-muted)` |
| Metric value | `font-mono text-xs tabular-nums text-(--color-spider-text)` |
| Section title | `text-xs font-medium uppercase tracking-wide text-(--color-spider-muted)` |

## Panel / surface

```tsx
<div class="rounded-(--radius-spider) border border-(--color-spider-border) bg-(--color-spider-surface) p-2" />
```

## What stays non-Tailwind

- Preview video — WebCodecs `VideoDecoder` → `<canvas>` from `/api/v1/ws/preview`
- State timeline cell colors (`style` background from `MixerState`)
- Vertical meter fill height and peak tick position
- Score border `color-mix` / opacity from `scoreSmooth`
