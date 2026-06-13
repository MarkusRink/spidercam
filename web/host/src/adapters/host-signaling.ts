import type { HostConfig } from "@spidercam/protocol";
import type {
  CaptureDevices,
  CaptureSelection,
  CaptureState,
  RoomState,
  StreamProcessingFlags,
} from "@spidercam/protocol";

export interface HostSignaling {
  connect(): Promise<void>;
  disconnect(): void;
  onState(handler: (state: RoomState) => void): () => void;
  onCaptureDevices(handler: (devices: CaptureDevices) => void): () => void;
  onCaptureUpdated(
    handler: (capture: CaptureState, error?: string) => void,
  ): () => void;
  onParticipantUrl(handler: (url: string) => void): () => void;
  sendConfig(partial: Partial<HostConfig>): void;
  listCaptureDevices(): void;
  setCaptureDevices(selection: CaptureSelection): void;
  setStreamProcessing(
    participantId: string,
    flags: StreamProcessingFlags,
  ): void;
  copyParticipantUrl(): void;
}
