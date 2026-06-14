import type {
  ParticipantPeer,
  PeerConnectionState,
} from "./participant-peer.js";

export type { PeerConnectionState } from "./participant-peer.js";

export class FakeParticipantPeer implements ParticipantPeer {
  private state: PeerConnectionState = "new";
  private stateHandler: ((state: PeerConnectionState) => void) | null = null;

  async start(stream: MediaStream): Promise<void> {
    void stream;
    this.setState("connecting");
    this.setState("connected");
  }

  async replaceTrack(
    kind: "audio" | "video",
    track: MediaStreamTrack | null,
  ): Promise<void> {
    void kind;
    void track;
  }

  close(): void {
    this.setState("closed");
  }

  onConnectionStateChange(handler: (state: PeerConnectionState) => void): void {
    this.stateHandler = handler;
  }

  private setState(state: PeerConnectionState): void {
    this.state = state;
    this.stateHandler?.(state);
  }
}
