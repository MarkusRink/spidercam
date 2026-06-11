import type { StreamMetrics } from "@spidercam/shared";

export async function collectStats(
  pc: RTCPeerConnection,
  participantId: string,
  seat: number,
  audioLevel: number,
): Promise<Partial<StreamMetrics>> {
  const report = await pc.getStats();
  let rttMs: number | null = null;
  let packetLoss: number | null = null;
  let jitterMs: number | null = null;
  let bitrateKbps: number | null = null;
  let framesPerSecond: number | null = null;

  for (const stat of report.values()) {
    if (stat.type === "candidate-pair" && stat.state === "succeeded") {
      rttMs = stat.currentRoundTripTime != null ? Math.round(stat.currentRoundTripTime * 1000) : rttMs;
    }
    if (stat.type === "inbound-rtp" && stat.kind === "audio") {
      if (stat.packetsLost != null && stat.packetsReceived != null) {
        const total = stat.packetsLost + stat.packetsReceived;
        packetLoss = total > 0 ? Math.round((stat.packetsLost / total) * 1000) / 10 : 0;
      }
      jitterMs = stat.jitter != null ? Math.round(stat.jitter * 1000) : jitterMs;
      if (stat.bytesReceived != null && stat.timestamp != null) {
        bitrateKbps = Math.round((stat.bytesReceived * 8) / 1000);
      }
    }
    if (stat.type === "inbound-rtp" && stat.kind === "video") {
      framesPerSecond = stat.framesPerSecond ?? framesPerSecond;
    }
  }

  return {
    participantId,
    seat,
    audioLevel,
    audioActive: audioLevel > 0.02,
    rttMs,
    packetLoss,
    jitterMs,
    bitrateKbps,
    framesPerSecond,
  };
}

export function createAudioLevelMonitor(
  stream: MediaStream,
  onLevel: (level: number) => void,
): () => void {
  const ctx = new AudioContext();
  const source = ctx.createMediaStreamSource(stream);
  const analyser = ctx.createAnalyser();
  analyser.fftSize = 256;
  source.connect(analyser);

  const data = new Uint8Array(analyser.frequencyBinCount);
  let raf = 0;

  const tick = () => {
    analyser.getByteFrequencyData(data);
    let sum = 0;
    for (let i = 0; i < data.length; i++) sum += data[i];
    const level = sum / (data.length * 255);
    onLevel(level);
    raf = requestAnimationFrame(tick);
  };
  tick();

  return () => {
    cancelAnimationFrame(raf);
    source.disconnect();
    void ctx.close();
  };
}
