import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { startTestServer, connectBridge } from "./helpers.js";

describe("bridge integration", () => {
  it("sends bridge-ready on connect", async () => {
    const server = await startTestServer();
    try {
      const { ws, ready } = await connectBridge(server.bridgeUrl);
      assert.equal(ready.type, "bridge-ready");
      ws.terminate();
    } finally {
      await server.close();
    }
  });

  it("accepts video frames and audio chunks after config", async () => {
    const server = await startTestServer();
    try {
      const { ws } = await connectBridge(server.bridgeUrl);

      ws.send(
        JSON.stringify({
          type: "bridge-config",
          width: 64,
          height: 48,
          sampleRate: 48000,
          channels: 1,
        }),
      );

      await new Promise((r) => setTimeout(r, 50));
      assert.equal(server.bridge.started, true);

      const frame = Buffer.alloc(64 * 48 * 4, 128);
      ws.send(frame);
      await new Promise((r) => setTimeout(r, 50));
      assert.equal(server.bridge.videoFrames.length, 1);
      assert.equal(server.bridge.videoFrames[0].length, frame.length);

      const pcm = Buffer.alloc(256, 0);
      ws.send(
        JSON.stringify({
          type: "audio-chunk",
          data: pcm.toString("base64"),
        }),
      );
      await new Promise((r) => setTimeout(r, 50));
      assert.equal(server.bridge.audioChunks.length, 1);

      ws.terminate();
    } finally {
      await server.close();
    }
  });
});
