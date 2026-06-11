import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { startTestServer, SignalingTestClient } from "./helpers.js";

describe("signaling integration", () => {
  it("welcomes clients with a client id", async () => {
    const server = await startTestServer();
    try {
      const client = await SignalingTestClient.connect(server.wsUrl);
      assert.ok(client.clientId);
      client.close();
    } finally {
      await server.close();
    }
  });

  it("broadcasts room updates when participants join", async () => {
    const server = await startTestServer();
    try {
      const host = await SignalingTestClient.connect(server.wsUrl);
      const update = await host.join({ name: "host", seat: 0, role: "host-mixer" });
      assert.equal(update.type, "room-update");
      if (update.type !== "room-update") return;
      assert.equal(update.room.participants.length, 1);

      const participant = await SignalingTestClient.connect(server.wsUrl);
      const pUpdate = await participant.join({ name: "bob", seat: 3, role: "participant" });
      if (pUpdate.type !== "room-update") return;
      assert.equal(pUpdate.room.participants.length, 2);

      const hostUpdate = await host.waitFor("room-update");
      if (hostUpdate.type !== "room-update") return;
      assert.equal(hostUpdate.room.participants.length, 2);

      host.close();
      participant.close();
    } finally {
      await server.close();
    }
  });

  it("relays WebRTC offers between clients", async () => {
    const server = await startTestServer();
    try {
      const a = await SignalingTestClient.connect(server.wsUrl);
      const b = await SignalingTestClient.connect(server.wsUrl);
      await a.join({ name: "a", seat: 1, role: "participant" });
      await b.join({ name: "b", seat: 2, role: "participant" });

      a.send({
        type: "offer",
        from: a.clientId!,
        to: b.clientId!,
        sdp: { type: "offer", sdp: "v=0" },
      });

      const received = await b.waitFor("offer");
      assert.equal(received.type, "offer");
      if (received.type !== "offer") return;
      assert.equal(received.from, a.clientId);
      assert.equal(received.sdp.sdp, "v=0");

      a.close();
      b.close();
    } finally {
      await server.close();
    }
  });

  it("responds to ping with pong", async () => {
    const server = await startTestServer();
    try {
      const client = await SignalingTestClient.connect(server.wsUrl);
      const ts = Date.now();
      client.send({ type: "ping", ts });
      const pong = await client.waitFor("pong");
      assert.equal(pong.type, "pong");
      if (pong.type !== "pong") return;
      assert.equal(pong.ts, ts);
      assert.ok(pong.serverTs >= ts);
      client.close();
    } finally {
      await server.close();
    }
  });

  it("propagates metrics and selection updates", async () => {
    const server = await startTestServer();
    try {
      const host = await SignalingTestClient.connect(server.wsUrl);
      const participant = await SignalingTestClient.connect(server.wsUrl);
      await host.join({ name: "host", seat: 0, role: "host-mixer" });
      await participant.join({ name: "p1", seat: 4, role: "participant" });

      participant.send({
        type: "metrics",
        from: participant.clientId!,
        metrics: { audioLevel: 0.5, rttMs: 20 },
      });

      let m: { audioLevel?: number; rttMs?: number | null } | undefined;
      for (let i = 0; i < 5; i++) {
        const metricsUpdate = await host.waitFor("room-update");
        if (metricsUpdate.type !== "room-update") continue;
        m = metricsUpdate.room.metrics.find((x) => x.participantId === participant.clientId);
        if (m?.audioLevel === 0.5) break;
      }
      assert.equal(m?.audioLevel, 0.5);
      assert.equal(m?.rttMs, 20);

      host.send({
        type: "selection",
        selection: {
          activeVideoId: "host",
          activeAudioId: participant.clientId!,
          reason: "test",
          timestamp: Date.now(),
        },
      });

      const selUpdate = await participant.waitFor("room-update");
      if (selUpdate.type !== "room-update") return;
      assert.equal(selUpdate.room.selection?.activeAudioId, participant.clientId);

      host.close();
      participant.close();
    } finally {
      await server.close();
    }
  });
});
