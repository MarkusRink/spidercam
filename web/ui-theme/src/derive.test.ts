import { describe, expect, it } from "vitest";
import { HostStreamID } from "@spidercam/protocol";
import type { ParticipantRoomView, StreamMetrics } from "@spidercam/protocol";
import {
  deriveOnAirLabel,
  formatLoopDelayText,
  isRouted,
  levelPct,
  loopDelayColor,
  loopDelayKnown,
  scoreBorderOpacity,
  transportCells,
} from "./derive.js";

function baseMetric(overrides: Partial<StreamMetrics> = {}): StreamMetrics {
  return {
    participantId: "p1",
    name: "Alice",
    role: "participant",
    rmsDbfs: -30,
    peakDbfs: -20,
    speechLevelDbfs: -32,
    noiseFloorDbfs: -58,
    snrDb: 14,
    vad: true,
    vadHangoverMs: 0,
    score: 0.5,
    scoreSmooth: 0.5,
    scoreComponents: {
      level: 0.5,
      snr: 0.5,
      vad: 1,
      priority: 0,
      echoPenalty: 0,
    },
    rank: 1,
    gateGainDb: 0,
    duckingGainDb: 0,
    calibrationGain: 1,
    calibrationPhase: "idle",
    jitterBufferFrames: 3,
    delayOffsetMs: 0,
    isMainTalker: true,
    videoActive: true,
    audioActive: true,
    rttMs: 42,
    packetLoss: 0.2,
    jitterMs: 8,
    bitrateKbps: 300,
    framesPerSecond: 29,
    lastUpdated: 0,
    loopDelay: { ms: 100, uncertaintyMs: 0, known: true },
    aecEnabled: false,
    denoiseEnabled: false,
    aecUs: 0,
    denoiseUs: 0,
    ...overrides,
  };
}

describe("levelPct", () => {
  it("maps -60 dBFS to 0%", () => {
    expect(levelPct(-60)).toBe(0);
  });

  it("maps 0 dBFS to 100%", () => {
    expect(levelPct(0)).toBe(100);
  });

  it("maps -30 dBFS to 50%", () => {
    expect(levelPct(-30)).toBe(50);
  });

  it("clamps below -60 dBFS", () => {
    expect(levelPct(-80)).toBe(0);
  });

  it("clamps above 0 dBFS", () => {
    expect(levelPct(6)).toBe(100);
  });
});

describe("scoreBorderOpacity", () => {
  it("returns 0.15 at score 0", () => {
    expect(scoreBorderOpacity(0)).toBeCloseTo(0.15);
  });

  it("returns 1.0 at score 1", () => {
    expect(scoreBorderOpacity(1)).toBeCloseTo(1);
  });

  it("interpolates mid-range scores", () => {
    expect(scoreBorderOpacity(0.5)).toBeCloseTo(0.575);
  });

  it("clamps out-of-range scores", () => {
    expect(scoreBorderOpacity(-1)).toBeCloseTo(0.15);
    expect(scoreBorderOpacity(2)).toBeCloseTo(1);
  });
});

describe("deriveOnAirLabel", () => {
  const view = (
    selection: ParticipantRoomView["selection"],
  ): ParticipantRoomView => ({
    participants: [
      { id: "p1", name: "Alice", hasVideo: true, hasAudio: true, joinedAt: 0 },
      { id: "p2", name: "Bob", hasVideo: true, hasAudio: true, joinedAt: 0 },
    ],
    selection,
    selfMetric: {
      rmsDbfs: -30,
      snrDb: 10,
      vad: false,
      calibrationPhase: "idle",
      loopDelay: { ms: null, uncertaintyMs: 0, known: false },
    },
  });

  it('returns "—" when no main talker', () => {
    expect(deriveOnAirLabel(view(null), "p1")).toBe("—");
    expect(deriveOnAirLabel(view({ mainTalkerId: "" } as never), "p1")).toBe(
      "—",
    );
  });

  it('returns "you" when main talker is self', () => {
    expect(
      deriveOnAirLabel(
        view({ mainTalkerId: "p1" } as ParticipantRoomView["selection"]),
        "p1",
      ),
    ).toBe("you");
  });

  it('returns "host" when main talker is host', () => {
    expect(
      deriveOnAirLabel(
        view({
          mainTalkerId: HostStreamID,
        } as ParticipantRoomView["selection"]),
        "p1",
      ),
    ).toBe("host");
  });

  it("returns participant display name", () => {
    expect(
      deriveOnAirLabel(
        view({ mainTalkerId: "p2" } as ParticipantRoomView["selection"]),
        "p1",
      ),
    ).toBe("Bob");
  });

  it('returns "—" for unknown participant id', () => {
    expect(
      deriveOnAirLabel(
        view({ mainTalkerId: "missing" } as ParticipantRoomView["selection"]),
        "p1",
      ),
    ).toBe("—");
  });
});

describe("isRouted", () => {
  const view = (activeAudioId: string | undefined): ParticipantRoomView => ({
    participants: [],
    selection:
      activeAudioId != null
        ? ({ activeAudioId } as ParticipantRoomView["selection"])
        : null,
    selfMetric: {
      rmsDbfs: -30,
      snrDb: 10,
      vad: false,
      calibrationPhase: "idle",
      loopDelay: { ms: null, uncertaintyMs: 0, known: false },
    },
  });

  it("is true when active audio matches client id", () => {
    expect(isRouted(view("p1"), "p1")).toBe(true);
  });

  it("is false when active audio differs or missing", () => {
    expect(isRouted(view("p2"), "p1")).toBe(false);
    expect(isRouted(view(undefined), "p1")).toBe(false);
  });
});

describe("formatLoopDelayText", () => {
  it('returns "—" when unknown', () => {
    expect(
      formatLoopDelayText({ ms: null, uncertaintyMs: 0, known: false }),
    ).toBe("—");
    expect(
      formatLoopDelayText({ ms: 100, uncertaintyMs: 0, known: false }),
    ).toBe("—");
  });

  it("formats known delay in milliseconds", () => {
    expect(
      formatLoopDelayText({ ms: 118, uncertaintyMs: 12, known: true }),
    ).toBe("~118 ms");
  });
});

describe("loopDelayKnown", () => {
  it("is false when estimate is incomplete", () => {
    expect(loopDelayKnown({ ms: null, uncertaintyMs: 0, known: false })).toBe(
      false,
    );
    expect(loopDelayKnown({ ms: 100, uncertaintyMs: 0, known: false })).toBe(
      false,
    );
  });

  it("is true when ms and known are set", () => {
    expect(loopDelayKnown({ ms: 95, uncertaintyMs: 10, known: true })).toBe(
      true,
    );
  });
});

describe("loopDelayColor", () => {
  it("returns muted when unknown", () => {
    expect(loopDelayColor(null, false)).toBe("muted");
    expect(loopDelayColor(100, false)).toBe("muted");
  });

  it("returns accent below 100 ms", () => {
    expect(loopDelayColor(99, true)).toBe("accent");
  });

  it("returns warn between 100 and 150 ms", () => {
    expect(loopDelayColor(100, true)).toBe("warn");
    expect(loopDelayColor(150, true)).toBe("warn");
  });

  it("returns error above 150 ms", () => {
    expect(loopDelayColor(151, true)).toBe("error");
  });
});

describe("transportCells", () => {
  it("returns host capture stats instead of WebRTC slots", () => {
    const cells = transportCells(
      baseMetric({
        role: "host",
        participantId: HostStreamID,
        snrDb: 14,
        vad: true,
        scoreSmooth: 0.5,
        jitterBufferFrames: 3,
        peakDbfs: -20,
        audioActive: true,
        videoActive: false,
      }),
      true,
    );
    expect(cells.map((c) => c.text)).toEqual([
      "14.0 SNR",
      "VAD",
      "50%",
      "buf 3",
      "-20.0",
      "A-",
    ]);
  });

  it("formats participant transport metrics", () => {
    const cells = transportCells(baseMetric(), false);
    expect(cells.map((c) => c.text)).toEqual([
      "42ms",
      "0.2%",
      "8ms",
      "3",
      "29fps",
      "AV",
    ]);
    expect(cells.every((c) => c.tone == null)).toBe(true);
  });

  it("applies warn and error tones", () => {
    const cells = transportCells(
      baseMetric({
        packetLoss: 2,
        jitterMs: 25,
        jitterBufferFrames: 6,
        framesPerSecond: 15,
      }),
      false,
    );
    expect(cells[1]?.tone).toBe("warn");
    expect(cells[2]?.tone).toBe("warn");
    expect(cells[3]?.tone).toBe("warn");
    expect(cells[4]?.tone).toBe("warn");
  });

  it("flags severe transport issues as error", () => {
    const cells = transportCells(
      baseMetric({
        packetLoss: 4,
        jitterMs: 50,
        jitterBufferFrames: 12,
        framesPerSecond: null,
      }),
      false,
    );
    expect(cells[1]?.tone).toBe("error");
    expect(cells[2]?.tone).toBe("error");
    expect(cells[3]?.tone).toBe("error");
    expect(cells[4]?.tone).toBe("error");
  });
});
