# Participant monitor

**Target:** `web/participant/` — SolidJS + Tailwind, served on `:1234` (LAN).

Single viewport — **no connect vs session navigation**. Connection is a toggle on the same screen; local A/V preview and device pickers are always available.

## File layout

```
web/participant/
  src/
    main.tsx
    App.tsx
    signaling.ts
    webrtc.ts
    stores/
      participant-store.ts   # connection phase, local media, persisted clientId
    components/
      ParticipantShell.tsx
      LocalPreview.tsx
      DeviceRow.tsx
      RoomStatus.tsx         # on-air, loop delay, SNR — when connected
      LostHostBanner.tsx
    index.css
```

## Layout (one screen, max-width 420 px)

```
┌──────────────────────────────────────┐
│  [display name input]            ●   │  ← red dot when activeAudioId === clientId
│  a3f7c2e1-…                         │  ← clientId (UUID), muted mono
│  connected · WebRTC active           │  ← or “disconnected · local preview only”
├──────────────────────────────────────┤
│  [ local video preview 16:9 ]        │  ← always; dim when camera off
│           ▮ RMS meter                │  ← AnalyserNode on local stream
├──────────────────────────────────────┤
│  Microphone ▾              [toggle]  │
│  Camera ▾                  [toggle]  │
│  [ Connect ] / [ Disconnect ]        │
├──────────────────────────────────────┤
│  On air: you                         │  ← text only; no dot on this row
│  Loop delay · ~118 ms                │  ← hidden or “—” when disconnected
│  SNR · 14 dB · room · 5              │
│  transport 2×3 grid                  │
└──────────────────────────────────────┘
```

When **disconnected**, hide room metrics (on-air, loop delay, SNR, transport) or show `—` so layout does not jump. Local preview + meters stay live.

## Identity

| Field | Source | UI |
|-------|--------|-----|
| **`clientId`** | Server `welcome` (UUID) | Muted mono under display name; routing key (`activeAudioId`, `mainTalkerId`) |
| **Display name** | Client default `client-{random}`; user-editable | Header text input; cosmetic only — sent on `join` |

On first page load, generate default display name `client-{4–6 digit random}` if none stored. Persist display name + `clientId` in `sessionStorage`.

Server assigns **`clientId`** on every WebSocket connect (`welcome`). After host restart the id may change; display name is re-sent on `join`.

## Connection toggle

| State | WS | WebRTC | UI |
|-------|-----|--------|-----|
| **Disconnected** | closed | stopped | Local `getUserMedia` + preview; Connect button |
| **Connected** | open | active | `join` sent; `participant-view` drives room block; Disconnect button |

**Connect:** open WS → `welcome` → `getUserMedia` (if not already) → `join { name, hasVideo, hasAudio }` → offer/answer/ICE.

**Disconnect (user):** `leave` → close peer → close WS → stay on same screen; local preview continues.

Device dropdowns and toggles work in both states. Changing device while connected swaps tracks on the `RTCPeerConnection` (or brief reconnect).

## Routing display

```ts
function deriveOnAirLabel(view: ParticipantRoomView, myId: string): string {
  const main = view.selection?.mainTalkerId;
  if (!main) return "—";
  if (main === myId) return "you";
  if (main === HostStreamID) return "host";
  const p = view.participants.find((x) => x.id === main);
  return p?.name ?? "—";
}

function isRouted(view: ParticipantRoomView, myId: string): boolean {
  return view.selection?.activeAudioId === myId;
}
```

| UI element | Binding |
|------------|---------|
| Header **red dot** | `activeAudioId === clientId` — routed to Teams output |
| **On air:** line | `mainTalkerId` → `you` / `host` / participant display name; **no dot** on this row |

During audio crossfade, `mainTalkerId` and `activeAudioId` can differ briefly; dot follows routing, label follows selected talker.

## Self metrics

`selfMetric` from `participant-view` (daemon inbound RTP analysis @ low rate).

- **Loop delay** — `LoopDelayText` on `selfMetric.loopDelay` (connected only)
- **SNR / VAD** — from `selfMetric`
- **Local RMS** — `AnalyserNode` on own `MediaStream` for responsive meter (always, including disconnected)

Optional footnote under loop delay: “Updates when remote speaks in Teams”.

## Lost host connection

Triggered when the participant was **connected** and the host becomes unreachable:

- WebSocket closes abnormally or health probe fails
- WebRTC `connectionState` → `failed` / `disconnected` (debounced)
- HTTP `GET /api/health` fails while WS is down (optional secondary check)

### UI — same screen, banner state

Replace room metrics with a **Lost host connection** block (not a separate route):

```
┌──────────────────────────────────────┐
│  … header, preview, devices unchanged … │
├──────────────────────────────────────┤
│  Lost host connection                 │
│  Retrying in 4 s… (attempt 3)         │
│  [ Retry now ]  [ Disconnect ]        │
└──────────────────────────────────────┘
```

Local preview and device controls **stay active**. User can **Disconnect** to cancel auto-retry and return to idle disconnected state.

### Auto-reconnect

When connection was lost unintentionally (not user Disconnect):

1. Enter `reconnecting` phase; show banner with backoff countdown.
2. Retry WebSocket to `ws://${location.host}/api/v1/ws` with exponential backoff (e.g. 1 s → 2 s → 4 s → 8 s cap 30 s).
3. On WS open: `welcome` → store new `clientId` if changed → **auto `join`** with last display name + AV flags → re-run WebRTC negotiation.
4. On success: clear banner; restore room metrics.
5. On user **Disconnect** during retry: abort timer, close WS, `idle`.

Persist `wasConnected` + last join payload in memory only for the retry loop; do not auto-connect on cold page load unless user had toggled Connect in this session.

Optional `join.clientId` hint (see [domain/messages.md](../domain/messages.md)): server may ignore unknown ids after host restart.

## Signaling — `/api/v1/ws`

```ts
export class ParticipantSignaling {
  connect(): Promise<void> {
    this.ws = new WebSocket(`ws://${location.host}/api/v1/ws`);
  }

  onWelcome(handler: (clientId: string, view: ParticipantRoomView) => void): void;
  onView(handler: (view: ParticipantRoomView) => void): void;
  onClose(handler: (wasClean: boolean) => void): void;

  sendJoin(name: string, hasVideo: boolean, hasAudio: boolean, clientId?: string): void;
  sendLeave(): void;
  relaySDP(msg: Offer | Answer | ICE): void;
}
```

## WebRTC → Pion

```ts
export class ParticipantPeer {
  async start(stream: MediaStream): Promise<void> {
    this.pc = new RTCPeerConnection({ iceServers: [...] });
    stream.getTracks().forEach((t) => this.pc!.addTrack(t, stream));
    // offer → signaling → Go hub
  }

  async replaceTrack(kind: "audio" | "video", track: MediaStreamTrack | null): Promise<void>;
  close(): void;
}
```

## Store sketch

```ts
type ConnectionPhase = "idle" | "connecting" | "connected" | "reconnecting" | "lost";

export interface ParticipantStore {
  phase: ConnectionPhase;
  clientId: string | null;
  displayName: string;
  localStream: MediaStream | null;
  view: ParticipantRoomView | null;
  reconnectAttempt: number;
  reconnectDelayMs: number;
}
```
