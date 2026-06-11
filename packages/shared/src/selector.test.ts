import { describe, it, beforeEach } from "node:test";
import assert from "node:assert/strict";
import { selectSources, resetSelectorState, seatDistance } from "./selector.js";
import { DEFAULT_HOST_CONFIG, type StreamMetrics } from "./types.js";

function metric(id: string, seat: number, level: number): StreamMetrics {
  return {
    participantId: id,
    seat,
    audioLevel: level,
    videoActive: true,
    audioActive: level > 0.02,
    rttMs: 10,
    packetLoss: 0,
    jitterMs: 1,
    bitrateKbps: 64,
    framesPerSecond: 30,
    lastUpdated: Date.now(),
  };
}

describe("seatDistance", () => {
  it("measures circular distance around table", () => {
    assert.equal(seatDistance(8, 0, 4), 4);
    assert.equal(seatDistance(8, 0, 7), 1);
    assert.equal(seatDistance(8, 3, 4), 1);
  });
});

describe("selectSources", () => {
  beforeEach(() => resetSelectorState());

  it("defaults to host when silent", () => {
    const result = selectSources({
      config: DEFAULT_HOST_CONFIG,
      metrics: [],
      hostAudioLevel: 0,
      hostVideoActive: true,
    });
    assert.equal(result.activeVideoId, "host");
    assert.equal(result.activeAudioId, "host");
  });

  it("selects loudest connected speaker", () => {
    const result = selectSources({
      config: DEFAULT_HOST_CONFIG,
      metrics: [metric("p1", 4, 0.5)],
      hostAudioLevel: 0,
      hostVideoActive: true,
      now: 10000,
    });
    assert.equal(result.activeVideoId, "p1");
    assert.equal(result.activeAudioId, "p1");
  });

  it("routes audio to nearest connected seat for host-video orphan speech", () => {
    resetSelectorState();
    const result = selectSources({
      config: { ...DEFAULT_HOST_CONFIG, seatCount: 8, hostSeat: 0 },
      metrics: [
        metric("p-near", 3, 0.08),
        metric("p-far", 6, 0.05),
      ],
      hostAudioLevel: 0.15,
      hostVideoActive: true,
      now: 20000,
    });
    assert.equal(result.activeVideoId, "host");
    assert.equal(result.activeAudioId, "p-near");
  });
});
