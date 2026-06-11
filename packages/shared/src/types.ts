export type ClientRole = "participant" | "host-mixer";

export interface ParticipantInfo {
  id: string;
  name: string;
  seat: number;
  role: ClientRole;
  hasVideo: boolean;
  hasAudio: boolean;
  joinedAt: number;
}

export interface StreamMetrics {
  participantId: string;
  seat: number;
  audioLevel: number;
  videoActive: boolean;
  audioActive: boolean;
  rttMs: number | null;
  packetLoss: number | null;
  jitterMs: number | null;
  bitrateKbps: number | null;
  framesPerSecond: number | null;
  lastUpdated: number;
}

export interface SelectionState {
  activeVideoId: string;
  activeAudioId: string;
  reason: string;
  timestamp: number;
}

export interface RoomState {
  participants: ParticipantInfo[];
  metrics: StreamMetrics[];
  selection: SelectionState | null;
  seatCount: number;
}

export interface HostConfig {
  seatCount: number;
  hostSeat: number;
  defaultVideoId: string;
  defaultAudioId: string;
  audioThreshold: number;
  videoHoldMs: number;
  audioHoldMs: number;
}

export const DEFAULT_HOST_CONFIG: HostConfig = {
  seatCount: 8,
  hostSeat: 0,
  defaultVideoId: "host",
  defaultAudioId: "host",
  audioThreshold: 0.02,
  videoHoldMs: 1500,
  audioHoldMs: 800,
};
