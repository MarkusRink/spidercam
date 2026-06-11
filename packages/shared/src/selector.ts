import type { HostConfig, SelectionState, StreamMetrics } from "./types.js";

export function seatDistance(seatCount: number, a: number, b: number): number {
  const diff = Math.abs(a - b);
  return Math.min(diff, seatCount - diff);
}

export interface SelectorInput {
  config: HostConfig;
  metrics: StreamMetrics[];
  hostAudioLevel: number;
  hostVideoActive: boolean;
  now?: number;
}

let lastVideoId = "host";
let lastAudioId = "host";
let lastVideoSwitch = 0;
let lastAudioSwitch = 0;

export function resetSelectorState(): void {
  lastVideoId = "host";
  lastAudioId = "host";
  lastVideoSwitch = 0;
  lastAudioSwitch = 0;
}

export function selectSources(input: SelectorInput): SelectionState {
  const now = input.now ?? Date.now();
  const { config, metrics, hostAudioLevel } = input;
  const threshold = config.audioThreshold;

  const hostSpeaking = hostAudioLevel > threshold;
  const hostMetric: StreamMetrics = {
    participantId: "host",
    seat: config.hostSeat,
    audioLevel: hostAudioLevel,
    videoActive: input.hostVideoActive,
    audioActive: hostSpeaking,
    rttMs: 0,
    packetLoss: 0,
    jitterMs: 0,
    bitrateKbps: null,
    framesPerSecond: null,
    lastUpdated: now,
  };

  const allWithHost = [hostMetric, ...metrics];
  const connectedLoudest = [...metrics]
    .filter((m) => m.audioLevel > threshold)
    .sort((a, b) => b.audioLevel - a.audioLevel)[0];

  const hostDominant =
    hostSpeaking && (!connectedLoudest || hostAudioLevel > connectedLoudest.audioLevel * 1.2);

  let videoId = lastVideoId;
  let audioId = lastAudioId;
  let reason = "hold";

  if (connectedLoudest && !hostDominant) {
    const candidateVideoId = connectedLoudest.participantId;
    if (candidateVideoId !== lastVideoId) {
      if (now - lastVideoSwitch >= config.videoHoldMs || connectedLoudest.audioLevel > threshold * 3) {
        videoId = candidateVideoId;
        lastVideoSwitch = now;
        reason = `video: connected speaker ${candidateVideoId}`;
      }
    } else {
      videoId = candidateVideoId;
    }
  } else if (hostSpeaking) {
    videoId = "host";
    reason = "video: host (ambient/unconnected speech)";
  } else {
    videoId = config.defaultVideoId;
    reason = "video: default (silence)";
  }

  if (connectedLoudest && connectedLoudest.audioLevel > hostAudioLevel * 1.2) {
    if (connectedLoudest.participantId !== lastAudioId) {
      if (now - lastAudioSwitch >= config.audioHoldMs || connectedLoudest.audioLevel > threshold * 2) {
        audioId = connectedLoudest.participantId;
        lastAudioSwitch = now;
        reason = `audio: connected speaker ${connectedLoudest.participantId}`;
      }
    } else {
      audioId = connectedLoudest.participantId;
    }
  } else if (videoId === "host" && hostSpeaking) {
    const ambientThreshold = threshold * 0.2;
    const connected = metrics.filter((m) => m.audioLevel > ambientThreshold);
    if (connected.length > 0) {
      const oppositeSeat = hostSeatOpposite(config);
      const nearest = findNearestConnected(metrics, oppositeSeat, config, ambientThreshold);
      const pick = nearest ?? connected.sort((a, b) => b.audioLevel - a.audioLevel)[0];
      audioId = pick.participantId;
      reason = nearest
        ? `audio: host video, nearest connected seat ${nearest.seat}`
        : `audio: host video, connected pickup`;
    } else {
      audioId = "host";
      reason = "audio: host (no connected mics)";
    }
  } else if (hostSpeaking) {
    audioId = "host";
    reason = "audio: host";
  } else {
    audioId = config.defaultAudioId;
    reason = "audio: default (silence)";
  }

  lastVideoId = videoId;
  lastAudioId = audioId;

  return { activeVideoId: videoId, activeAudioId: audioId, reason, timestamp: now };
}

function hostSeatOpposite(config: HostConfig): number {
  return (config.hostSeat + Math.floor(config.seatCount / 2)) % config.seatCount;
}

function findNearestConnected(
  metrics: StreamMetrics[],
  targetSeat: number,
  config: HostConfig,
  threshold: number,
): StreamMetrics | null {
  const connected = metrics.filter((m) => m.participantId !== "host" && m.audioLevel > threshold);
  if (connected.length === 0) return null;

  let best: StreamMetrics | null = null;
  let bestDist = Infinity;

  for (const m of connected) {
    const dist = seatDistance(config.seatCount, targetSeat, m.seat);
    if (dist < bestDist || (dist === bestDist && best && m.audioLevel > best.audioLevel)) {
      bestDist = dist;
      best = m;
    }
  }

  return best;
}
