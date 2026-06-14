import { HostStreamID } from "@spidercam/protocol";
import type {
  LoopDelayEstimate,
  ParticipantRoomView,
  StreamMetrics,
} from "@spidercam/protocol";

export type TransportCell = { text: string; tone?: "warn" | "error" };

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}

export function formatDbfs(db: number): string {
  if (!Number.isFinite(db)) {
    return "-∞";
  }
  return db.toFixed(1);
}

export function levelPct(db: number): number {
  return clamp((db + 60) / 60, 0, 1) * 100;
}

export function scoreBorderOpacity(scoreSmooth: number): number {
  return 0.15 + 0.85 * clamp(scoreSmooth, 0, 1);
}

export function deriveOnAirLabel(
  view: ParticipantRoomView,
  myId: string,
): string {
  const main = view.selection?.mainTalkerId;
  if (!main) return "—";
  if (main === myId) return "you";
  if (main === HostStreamID) return "host";
  const p = view.participants.find((x) => x.id === main);
  return p?.name ?? "—";
}

export function isRouted(view: ParticipantRoomView, myId: string): boolean {
  return view.selection?.activeAudioId === myId;
}

export function loopDelayColor(
  ms: number | null,
  known: boolean,
): "muted" | "accent" | "warn" | "error" {
  if (!known || ms == null) return "muted";
  if (ms < 100) return "accent";
  if (ms <= 150) return "warn";
  return "error";
}

export function formatLoopDelayText(estimate: LoopDelayEstimate): string {
  return estimate.known && estimate.ms != null ? `~${estimate.ms} ms` : "—";
}

export function loopDelayKnown(estimate: LoopDelayEstimate): boolean {
  return estimate.known && estimate.ms != null;
}

function avLabel(audioActive: boolean, videoActive: boolean): string {
  if (audioActive && videoActive) return "AV";
  if (audioActive) return "A-";
  if (videoActive) return "-V";
  return "--";
}

function lossTone(packetLoss: number | null): TransportCell["tone"] {
  if (packetLoss == null) return undefined;
  if (packetLoss > 3) return "error";
  if (packetLoss > 1) return "warn";
  return undefined;
}

function jitterTone(jitterMs: number | null): TransportCell["tone"] {
  if (jitterMs == null) return undefined;
  if (jitterMs > 40) return "error";
  if (jitterMs > 20) return "warn";
  return undefined;
}

function bufTone(frames: number): TransportCell["tone"] {
  if (frames > 10) return "error";
  if (frames > 5) return "warn";
  return undefined;
}

function fpsTone(fps: number | null): TransportCell["tone"] {
  if (fps == null || fps < 10) return "error";
  if (fps < 20) return "warn";
  return undefined;
}

export function transportCells(
  metric: StreamMetrics,
  isHost: boolean,
): TransportCell[] {
  if (isHost) {
    return [
      { text: `${formatDbfs(metric.snrDb)} SNR` },
      { text: metric.vad ? "VAD" : "idle" },
      { text: `${Math.round(metric.scoreSmooth * 100)}%` },
      { text: `buf ${metric.jitterBufferFrames}` },
      { text: formatDbfs(metric.peakDbfs) },
      { text: avLabel(metric.audioActive, metric.videoActive) },
    ];
  }

  const rtt = metric.rttMs != null ? `${Math.round(metric.rttMs)}ms` : "—";
  const loss =
    metric.packetLoss != null ? `${metric.packetLoss.toFixed(1)}%` : "—";
  const jitter =
    metric.jitterMs != null ? `${Math.round(metric.jitterMs)}ms` : "—";
  const buf = String(metric.jitterBufferFrames);
  const fps =
    metric.framesPerSecond != null
      ? `${Math.round(metric.framesPerSecond)}fps`
      : "—";

  return [
    { text: rtt },
    { text: loss, tone: lossTone(metric.packetLoss) },
    { text: jitter, tone: jitterTone(metric.jitterMs) },
    { text: buf, tone: bufTone(metric.jitterBufferFrames) },
    { text: fps, tone: fpsTone(metric.framesPerSecond) },
    { text: avLabel(metric.audioActive, metric.videoActive) },
  ];
}
