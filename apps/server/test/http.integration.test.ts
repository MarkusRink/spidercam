import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { startTestServer } from "./helpers.js";

describe("http integration", () => {
  it("serves api info", async () => {
    const server = await startTestServer();
    try {
      const res = await fetch(`${server.baseUrl}/api/info`);
      assert.equal(res.status, 200);
      const body = (await res.json()) as { name: string; port: number };
      assert.equal(body.name, "spidercam");
      assert.equal(body.port, server.port);
    } finally {
      await server.close();
    }
  });

  it("serves participant and host pages", async () => {
    const server = await startTestServer();
    try {
      const index = await fetch(`${server.baseUrl}/`);
      assert.equal(index.status, 200);
      assert.match(await index.text(), /spidercam/i);

      const host = await fetch(`${server.baseUrl}/host.html`);
      assert.equal(host.status, 200);
      assert.match(await host.text(), /host/i);
    } finally {
      await server.close();
    }
  });
});
