import {
  HostStreamID,
  MixerHold,
  MixerLocked,
  MixerSilence,
  MixerSwitch,
  type MixerState,
  type RoomState,
  type StreamMetrics,
} from "@spidercam/protocol";
import { LOOP_DELAY_PUBLISH_MS, TICK_MS } from "./config.js";
import type { RoomModel } from "./room-model.js";

export type StateListener = (state: RoomState) => void;
export type ActiveVideoListener = (activeVideoId: string, seq: number) => void;

export type SelectionListener = () => void;

export class ScenarioEngine {
  private readonly room: RoomModel;
  private tickTimer: ReturnType<typeof setInterval> | null = null;
  private tickCount = 0;
  private previewSeq = 0;
  private lastLoopDelayPublish = 0;
  private stateListeners = new Set<StateListener>();
  private activeVideoListeners = new Set<ActiveVideoListener>();
  private selectionListeners = new Set<SelectionListener>();
  private lastActiveVideoId = "";

  constructor(room: RoomModel) {
    this.room = room;
    this.lastActiveVideoId = room.getActiveVideoId();
  }

  start(): void {
    if (this.tickTimer) {
      return;
    }
    this.tickTimer = setInterval(() => this.tick(), TICK_MS);
  }

  stop(): void {
    if (this.tickTimer) {
      clearInterval(this.tickTimer);
      this.tickTimer = null;
    }
  }

  onState(listener: StateListener): () => void {
    this.stateListeners.add(listener);
    return () => this.stateListeners.delete(listener);
  }

  onActiveVideoChange(listener: ActiveVideoListener): () => void {
    this.activeVideoListeners.add(listener);
    return () => this.activeVideoListeners.delete(listener);
  }

  onSelectionChange(listener: SelectionListener): () => void {
    this.selectionListeners.add(listener);
    return () => this.selectionListeners.delete(listener);
  }

  routeTo(id: string): boolean {
    const state = this.room.getState();
    if (!state.selection) {
      return false;
    }
    const valid =
      id === HostStreamID || state.participants.some((p) => p.id === id);
    if (!valid) {
      return false;
    }
    const prev = state.selection.activeVideoId;
    state.selection.activeVideoId = id;
    state.selection.activeAudioId = id;
    state.selection.mainTalkerId = id;
    state.selection.timestamp = Date.now();
    if (prev !== id) {
      this.emitActiveVideoChange(id);
    }
    this.emitSelectionChange();
    this.emitState();
    return true;
  }

  setMixerState(mixerState: MixerState): void {
    const state = this.room.getState();
    if (!state.selection) {
      return;
    }
    state.selection.mixerState = mixerState;
    state.selection.timestamp = Date.now();
    this.emitSelectionChange();
    this.emitState();
  }

  private tick(): void {
    this.tickCount += 1;
    const now = Date.now();
    const t = this.tickCount * 0.05;
    const state = this.room.getState();

    for (const metric of state.metrics) {
      this.animateMetric(metric, t);
      metric.lastUpdated = now;
    }

    state.reference.rmsDbfs = -42 + Math.sin(t * 0.7) * 4;
    state.reference.peakDbfs = state.reference.rmsDbfs + 5;
    state.reference.vad = state.reference.rmsDbfs > -40;
    state.reference.active = state.reference.vad;

    state.outLevelDbfs = -24 + Math.sin(t * 1.1) * 6;
    state.outPeakDbfs = state.outLevelDbfs + 4;
    state.enhancementBudgetPct = 3 + Math.sin(t * 0.3) * 1.5;

    if (state.selection) {
      state.selection.timestamp = now;
      if (state.selection.holdRemainingMs > 0) {
        state.selection.holdRemainingMs = Math.max(
          0,
          state.selection.holdRemainingMs - TICK_MS,
        );
      }
    }

    if (now - this.lastLoopDelayPublish >= LOOP_DELAY_PUBLISH_MS) {
      this.lastLoopDelayPublish = now;
      state.globalLatencyMs = 110 + Math.round(Math.sin(t) * 8);
      for (const metric of state.metrics) {
        if (metric.role === "participant" && metric.loopDelay.known) {
          metric.loopDelay = {
            ms: 95 + Math.round(Math.sin(t + 1) * 10),
            uncertaintyMs: 12,
            known: true,
          };
        }
      }
    }

    const activeVideoId = this.room.getActiveVideoId();
    if (activeVideoId !== this.lastActiveVideoId) {
      this.lastActiveVideoId = activeVideoId;
      this.emitActiveVideoChange(activeVideoId);
    }

    this.emitState();
  }

  private animateMetric(metric: StreamMetrics, t: number): void {
    const phase = metric.participantId.length + t;
    const base = metric.isMainTalker ? -22 : -38;
    metric.rmsDbfs =
      base + Math.sin(phase * 1.3) * (metric.isMainTalker ? 4 : 2);
    metric.peakDbfs = metric.rmsDbfs + 4 + Math.sin(phase * 2.1) * 1.5;
    metric.speechLevelDbfs = metric.rmsDbfs - 2;
    metric.snrDb = 10 + Math.sin(phase * 0.9) * 6;
    metric.vad = metric.rmsDbfs > -30;
    metric.score = Math.max(0, Math.min(1, 0.3 + Math.sin(phase) * 0.2));
    metric.scoreSmooth = metric.score * 0.9;
    if (metric.role === "participant") {
      metric.rttMs = 12 + Math.sin(phase) * 3;
      metric.packetLoss = Math.max(0, Math.sin(phase * 0.5) * 0.5);
      metric.jitterMs = 8 + Math.sin(phase * 1.7) * 3;
      metric.bitrateKbps = 300 + Math.sin(phase) * 40;
      metric.framesPerSecond = 29 + Math.sin(phase * 0.4);
    }
  }

  private emitState(): void {
    const snapshot = structuredClone(this.room.getState());
    for (const listener of this.stateListeners) {
      listener(snapshot);
    }
  }

  private emitActiveVideoChange(activeVideoId: string): void {
    this.previewSeq += 1;
    for (const listener of this.activeVideoListeners) {
      listener(activeVideoId, this.previewSeq);
    }
  }

  private emitSelectionChange(): void {
    for (const listener of this.selectionListeners) {
      listener();
    }
  }
}

export function parseMixerState(value: string): MixerState | null {
  switch (value.toUpperCase()) {
    case MixerLocked:
      return MixerLocked;
    case MixerHold:
      return MixerHold;
    case MixerSwitch:
      return MixerSwitch;
    case MixerSilence:
      return MixerSilence;
    default:
      return null;
  }
}
