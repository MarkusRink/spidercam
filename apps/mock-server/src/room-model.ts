import {
  DefaultHostConfig,
  HostStreamID,
  type HostConfig,
  type ParticipantInfo,
  type ParticipantRoomView,
  type RoomState,
  type SelectionState,
  type StreamMetrics,
  type StreamProcessingFlags,
} from "@spidercam/protocol";
import { loadRoutingState } from "./fixtures.js";

export interface ConnectedParticipant {
  id: string;
  name: string;
  hasVideo: boolean;
  hasAudio: boolean;
  joinedAt: number;
}

export class RoomModel {
  private state: RoomState;
  private config: HostConfig;
  private connected = new Map<string, ConnectedParticipant>();
  private streamProcessing = new Map<string, StreamProcessingFlags>();

  constructor(participantUrl: string) {
    this.state = loadRoutingState();
    this.state.participantUrl = participantUrl;
    this.config = structuredClone(DefaultHostConfig);
    this.streamProcessing.set(HostStreamID, {
      aecEnabled: false,
      denoiseEnabled: false,
    });
    for (const participant of this.state.participants) {
      this.streamProcessing.set(participant.id, {
        aecEnabled: false,
        denoiseEnabled: false,
      });
    }
  }

  getConfig(): HostConfig {
    return this.config;
  }

  getState(): RoomState {
    return this.state;
  }

  setState(state: RoomState): void {
    this.state = state;
  }

  replaceMetrics(metrics: StreamMetrics[]): void {
    this.state.metrics = metrics;
  }

  replaceSelection(selection: SelectionState): void {
    this.state.selection = selection;
  }

  getActiveVideoId(): string {
    return this.state.selection?.activeVideoId ?? "";
  }

  updateConfig(partial: Partial<HostConfig>): string | null {
    const merged = { ...this.config, ...partial };
    if (partial.scoreWeights) {
      merged.scoreWeights = {
        ...this.config.scoreWeights,
        ...partial.scoreWeights,
      };
    }
    if (merged.crossfadeMs < 0 || merged.crossfadeMs > 5000) {
      return "crossfadeMs out of range";
    }
    if (merged.audioHoldMs < 0 || merged.audioHoldMs > 10_000) {
      return "audioHoldMs out of range";
    }
    this.config = merged;
    return null;
  }

  setCaptureSelection(micId: string, cameraId: string, sinkId: string): void {
    const devices = this.state.capture;
    devices.micId = micId;
    devices.cameraId = cameraId;
    devices.sinkId = sinkId;
  }

  setStreamProcessing(
    participantId: string,
    flags: StreamProcessingFlags,
  ): boolean {
    const metric = this.state.metrics.find(
      (m) => m.participantId === participantId,
    );
    if (!metric && participantId !== HostStreamID) {
      return false;
    }
    this.streamProcessing.set(participantId, { ...flags });
    if (metric) {
      metric.aecEnabled = flags.aecEnabled;
      metric.denoiseEnabled = flags.denoiseEnabled;
    }
    return true;
  }

  addMockParticipant(
    name: string,
    hasVideo: boolean,
    hasAudio: boolean,
  ): ParticipantInfo {
    const id = crypto.randomUUID();
    const participant: ParticipantInfo = {
      id,
      name,
      hasVideo,
      hasAudio,
      joinedAt: Date.now(),
    };
    this.state.participants.push(participant);
    const template = this.state.metrics.find((m) => m.role === "participant");
    const metric: StreamMetrics = template
      ? {
          ...structuredClone(template),
          participantId: id,
          name,
          videoActive: hasVideo,
          audioActive: hasAudio,
          lastUpdated: Date.now(),
        }
      : {
          participantId: id,
          name,
          role: "participant",
          rmsDbfs: -40,
          peakDbfs: -35,
          speechLevelDbfs: -42,
          noiseFloorDbfs: -60,
          snrDb: 12,
          vad: false,
          vadHangoverMs: 0,
          score: 0.1,
          scoreSmooth: 0.08,
          scoreComponents: {
            level: 0.05,
            snr: 0.1,
            vad: 0,
            priority: 0,
            echoPenalty: 0,
          },
          rank: this.state.participants.length,
          gateGainDb: 0,
          duckingGainDb: 0,
          calibrationGain: 1,
          calibrationPhase: "idle",
          jitterBufferFrames: 2,
          delayOffsetMs: 0,
          isMainTalker: false,
          videoActive: hasVideo,
          audioActive: hasAudio,
          rttMs: 15,
          packetLoss: 0.1,
          jitterMs: 8,
          bitrateKbps: 300,
          framesPerSecond: 30,
          lastUpdated: Date.now(),
          loopDelay: { ms: null, uncertaintyMs: 0, known: false },
          aecEnabled: false,
          denoiseEnabled: false,
          aecUs: 0,
          denoiseUs: 0,
        };
    this.state.metrics.push(metric);
    this.streamProcessing.set(id, { aecEnabled: false, denoiseEnabled: false });
    return participant;
  }

  removeMockParticipant(id: string): boolean {
    const before = this.state.participants.length;
    this.state.participants = this.state.participants.filter(
      (p) => p.id !== id,
    );
    this.state.metrics = this.state.metrics.filter(
      (m) => m.participantId !== id,
    );
    this.connected.delete(id);
    this.streamProcessing.delete(id);
    if (this.state.selection) {
      if (this.state.selection.activeVideoId === id) {
        this.state.selection.activeVideoId = HostStreamID;
      }
      if (this.state.selection.activeAudioId === id) {
        this.state.selection.activeAudioId = HostStreamID;
      }
      if (this.state.selection.mainTalkerId === id) {
        this.state.selection.mainTalkerId = HostStreamID;
      }
    }
    return this.state.participants.length < before;
  }

  join(
    clientId: string,
    name: string,
    hasVideo: boolean,
    hasAudio: boolean,
  ): ConnectedParticipant {
    const existing = this.connected.get(clientId);
    if (existing) {
      existing.name = name;
      existing.hasVideo = hasVideo;
      existing.hasAudio = hasAudio;
      const info = this.state.participants.find((p) => p.id === clientId);
      if (info) {
        info.name = name;
        info.hasVideo = hasVideo;
        info.hasAudio = hasAudio;
      } else {
        this.state.participants.push({
          id: clientId,
          name,
          hasVideo,
          hasAudio,
          joinedAt: Date.now(),
        });
        this.addMetricFor(clientId, name, hasVideo, hasAudio);
      }
      return existing;
    }

    let info = this.state.participants.find((p) => p.id === clientId);
    if (!info) {
      info = {
        id: clientId,
        name,
        hasVideo,
        hasAudio,
        joinedAt: Date.now(),
      };
      this.state.participants.push(info);
      this.addMetricFor(clientId, name, hasVideo, hasAudio);
    } else {
      info.name = name;
      info.hasVideo = hasVideo;
      info.hasAudio = hasAudio;
    }

    const participant: ConnectedParticipant = {
      id: clientId,
      name,
      hasVideo,
      hasAudio,
      joinedAt: info.joinedAt,
    };
    this.connected.set(clientId, participant);
    return participant;
  }

  leave(clientId: string): void {
    this.connected.delete(clientId);
    this.state.participants = this.state.participants.filter(
      (p) => p.id !== clientId,
    );
    this.state.metrics = this.state.metrics.filter(
      (m) => m.participantId !== clientId,
    );
    this.streamProcessing.delete(clientId);
  }

  viewFor(clientId: string): ParticipantRoomView {
    const metric = this.state.metrics.find((m) => m.participantId === clientId);
    return {
      participants: [...this.state.participants],
      selection: this.state.selection ? { ...this.state.selection } : null,
      selfMetric: {
        rmsDbfs: metric?.rmsDbfs ?? -60,
        snrDb: metric?.snrDb ?? 0,
        vad: metric?.vad ?? false,
        calibrationPhase: metric?.calibrationPhase ?? "idle",
        loopDelay: metric?.loopDelay ?? {
          ms: null,
          uncertaintyMs: 0,
          known: false,
        },
      },
    };
  }

  connectedIds(): string[] {
    return [...this.connected.keys()];
  }

  private addMetricFor(
    clientId: string,
    name: string,
    hasVideo: boolean,
    hasAudio: boolean,
  ): void {
    if (this.state.metrics.some((m) => m.participantId === clientId)) {
      return;
    }
    const template = this.state.metrics.find((m) => m.role === "participant");
    const metric: StreamMetrics = template
      ? {
          ...structuredClone(template),
          participantId: clientId,
          name,
          videoActive: hasVideo,
          audioActive: hasAudio,
          lastUpdated: Date.now(),
        }
      : {
          participantId: clientId,
          name,
          role: "participant",
          rmsDbfs: -40,
          peakDbfs: -35,
          speechLevelDbfs: -42,
          noiseFloorDbfs: -60,
          snrDb: 12,
          vad: false,
          vadHangoverMs: 0,
          score: 0.1,
          scoreSmooth: 0.08,
          scoreComponents: {
            level: 0.05,
            snr: 0.1,
            vad: 0,
            priority: 0,
            echoPenalty: 0,
          },
          rank: this.state.participants.length,
          gateGainDb: 0,
          duckingGainDb: 0,
          calibrationGain: 1,
          calibrationPhase: "idle",
          jitterBufferFrames: 2,
          delayOffsetMs: 0,
          isMainTalker: false,
          videoActive: hasVideo,
          audioActive: hasAudio,
          rttMs: 15,
          packetLoss: 0.1,
          jitterMs: 8,
          bitrateKbps: 300,
          framesPerSecond: 30,
          lastUpdated: Date.now(),
          loopDelay: { ms: null, uncertaintyMs: 0, known: false },
          aecEnabled: false,
          denoiseEnabled: false,
          aecUs: 0,
          denoiseUs: 0,
        };
    this.state.metrics.push(metric);
    this.streamProcessing.set(clientId, {
      aecEnabled: false,
      denoiseEnabled: false,
    });
  }
}
