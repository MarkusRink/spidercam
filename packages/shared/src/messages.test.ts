import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { isSignalingMessage } from "./messages.js";

describe("isSignalingMessage", () => {
  it("accepts valid signaling messages", () => {
    assert.equal(isSignalingMessage({ type: "ping", ts: 1 }), true);
    assert.equal(isSignalingMessage({ type: "join", name: "a", seat: 0, role: "participant", hasVideo: true, hasAudio: true }), true);
    assert.equal(isSignalingMessage({ type: "leave" }), true);
  });

  it("rejects invalid payloads", () => {
    assert.equal(isSignalingMessage(null), false);
    assert.equal(isSignalingMessage("ping"), false);
    assert.equal(isSignalingMessage({}), false);
    assert.equal(isSignalingMessage({ notype: true }), false);
  });
});
