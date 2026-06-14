export type PeerConnectionState =
  | "new"
  | "connecting"
  | "connected"
  | "disconnected"
  | "failed"
  | "closed";

export interface ParticipantPeer {
  start(stream: MediaStream): Promise<void>;
  replaceTrack(
    kind: "audio" | "video",
    track: MediaStreamTrack | null,
  ): Promise<void>;
  close(): void;
  onConnectionStateChange(handler: (state: PeerConnectionState) => void): void;
}
