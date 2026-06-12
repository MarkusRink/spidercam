# Participant monitor

**Target:** `web/participant/` — SolidJS + Tailwind, served on `:1234` (LAN).

## File layout

```
web/participant/
  src/
    main.tsx
    App.tsx
    signaling.ts
    webrtc.ts
    components/
    index.css
```

## Signaling — `/api/v1/ws` only

```ts
export class ParticipantSignaling {
  connect(): Promise<void> {
    this.ws = new WebSocket(`ws://${location.host}/api/v1/ws`);
  }

  onView(handler: (view: ParticipantRoomView) => void): void {
    // participant-view messages only — never RoomState
  }

  sendJoin(name: string, hasVideo: boolean, hasAudio: boolean): void;
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
}
```

## Routing display

Same `deriveRouting(view, myId)` logic — uses `participant-view.selection` only.

## Self metrics

`selfMetric` from `participant-view` (daemon computes from inbound RTP analysis @ low rate).

Includes **`loopDelay`** — passive loop estimate. Render **LoopDelayText** in session panel ([design-system.md](./design-system.md)).

Local RMS meter optional: `AnalyserNode` on own stream for responsive UI; SNR/VAD from server view.

## Session card layout

Below preview / meter table:

- **Loop delay** — `LoopDelayText` bound to `selfMetric.loopDelay`
- Muted footnote optional: “Updates when remote speaks in Teams”

## Connect view

Tailwind form — name, AV toggles, connect. No seat field. Server URL implicit (`location.host`).
