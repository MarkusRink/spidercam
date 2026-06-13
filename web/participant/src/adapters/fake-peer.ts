export type PeerConnectionState =
  | "new"
  | "connecting"
  | "connected"
  | "disconnected"
  | "failed"
  | "closed";

type ConnectionStateHandler = (state: PeerConnectionState) => void;

export class FakeParticipantPeer {
  private state: PeerConnectionState = "new";
  private stateHandler: ConnectionStateHandler | null = null;

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

  onConnectionStateChange(handler: ConnectionStateHandler): void {
    this.stateHandler = handler;
  }

  private setState(state: PeerConnectionState): void {
    this.state = state;
    this.stateHandler?.(state);
  }
}
