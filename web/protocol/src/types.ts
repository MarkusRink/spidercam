import type { MixerState } from "./constants.js";

export type ClientRole = "participant";

export type StreamRole = "host" | "participant" | "reference";

export interface LoopDelayEstimate {
  ms: number | null;
  uncertaintyMs: number;
  known: boolean;
}

export interface ParticipantInfo {
  id: string;
  name: string;
  hasVideo: boolean;
  hasAudio: boolean;
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
  role: StreamRole;
  rmsDbfs: number;
  peakDbfs: number;
  speechLevelDbfs: number;
  noiseFloorDbfs: number;
  snrDb: number;
  vad: boolean;
  vadHangoverMs: number;
  score: number;
  scoreSmooth: number;
  scoreComponents: ScoreComponents;
  rank: number;
  gateGainDb: number;
  duckingGainDb: number;
  calibrationGain: number;
  calibrationPhase: string;
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
  loopDelay: LoopDelayEstimate;
  aecEnabled: boolean;
  denoiseEnabled: boolean;
  aecUs: number;
  denoiseUs: number;
}

export interface StreamProcessingFlags {
  aecEnabled: boolean;
  denoiseEnabled: boolean;
}

export interface ReferenceMetrics {
  streamId: string;
  rmsDbfs: number;
  peakDbfs: number;
  vad: boolean;
  active: boolean;
}

export interface CrossfadeState {
  fromId: string;
  toId: string;
  t: number;
}

export interface SwitchEvent {
  at: number;
  fromId: string;
  toId: string;
  reason: string;
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

export interface CaptureState {
  micId: string;
  micLabel: string;
  cameraId: string;
  cameraLabel: string;
  sinkId: string;
  sinkLabel: string;
}

export interface RoomState {
  participants: ParticipantInfo[];
  metrics: StreamMetrics[];
  reference: ReferenceMetrics;
  selection: SelectionState | null;
  capture: CaptureState;
  outputHealthy: boolean;
  globalLatencyMs: number | null;
  outLevelDbfs: number;
  outPeakDbfs: number;
  enhancementBudgetPct: number;
  participantUrl: string;
}

export interface DeviceInfo {
  id: string;
  label: string;
}

export interface CaptureDevices {
  mics: DeviceInfo[];
  cameras: DeviceInfo[];
  sinks: DeviceInfo[];
}

export interface CaptureSelection {
  micId?: string;
  cameraId?: string;
  sinkId?: string;
}

export interface SelfMetric {
  rmsDbfs: number;
  snrDb: number;
  vad: boolean;
  calibrationPhase: string;
  loopDelay: LoopDelayEstimate;
}

export interface ParticipantRoomView {
  participants: ParticipantInfo[];
  selection: SelectionState | null;
  selfMetric: SelfMetric;
}
