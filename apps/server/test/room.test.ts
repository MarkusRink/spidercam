import { describe, it, beforeEach } from "node:test";
import assert from "node:assert/strict";
import { EventEmitter } from "node:events";
import { Room } from "../src/room.js";
import type { SelectionState } from "@spidercam/shared";

function mockWs(): EventEmitter & { send: () => void; readyState: number; OPEN: number } {
  const ws = new EventEmitter() as EventEmitter & {
    send: () => void;
    readyState: number;
    OPEN: number;
  };
  ws.send = () => {};
  ws.readyState = 1;
  ws.OPEN = 1;
  return ws;
}

describe("Room", () => {
  let room: Room;

  beforeEach(() => {
    room = new Room();
  });

  it("starts empty", () => {
    const state = room.getState();
    assert.equal(state.participants.length, 0);
    assert.equal(state.metrics.length, 0);
    assert.equal(state.selection, null);
    assert.equal(state.seatCount, 8);
  });

  it("tracks participants and metrics on add", () => {
    room.addClient("a", mockWs() as never, {
      id: "a",
      name: "Alice",
      seat: 2,
      role: "participant",
      hasVideo: true,
      hasAudio: true,
      joinedAt: Date.now(),
    });

    const state = room.getState();
    assert.equal(state.participants.length, 1);
    assert.equal(state.participants[0].name, "Alice");
    assert.equal(state.metrics[0].seat, 2);
    assert.equal(state.metrics[0].audioLevel, 0);
  });

  it("removes client and metrics", () => {
    room.addClient("a", mockWs() as never, {
      id: "a",
      name: "Alice",
      seat: 0,
      role: "participant",
      hasVideo: false,
      hasAudio: true,
      joinedAt: Date.now(),
    });
    room.removeClient("a");
    assert.equal(room.getState().participants.length, 0);
    assert.equal(room.getMetrics().length, 0);
  });

  it("updates metrics partially", () => {
    room.addClient("a", mockWs() as never, {
      id: "a",
      name: "Alice",
      seat: 3,
      role: "participant",
      hasVideo: true,
      hasAudio: true,
      joinedAt: Date.now(),
    });
    room.updateMetrics("a", { audioLevel: 0.42, rttMs: 12 });
    const m = room.getMetrics()[0];
    assert.equal(m.audioLevel, 0.42);
    assert.equal(m.rttMs, 12);
    assert.equal(m.seat, 3);
  });

  it("stores selection and config", () => {
    const selection: SelectionState = {
      activeVideoId: "host",
      activeAudioId: "a",
      reason: "test",
      timestamp: Date.now(),
    };
    room.setSelection(selection);
    room.updateConfig({ seatCount: 12, hostSeat: 1 });
    const state = room.getState();
    assert.deepEqual(state.selection, selection);
    assert.equal(room.getConfig().seatCount, 12);
    assert.equal(room.getConfig().hostSeat, 1);
  });
});
