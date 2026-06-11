import WebSocket from "ws";
import type { SignalingMessage } from "@spidercam/shared";
import { createApp, type SpidercamApp } from "../src/create-app.js";
import { MockVirtualDeviceBridge } from "./mock-bridge.js";

export interface TestServer {
  port: number;
  baseUrl: string;
  wsUrl: string;
  bridgeUrl: string;
  app: SpidercamApp;
  bridge: MockVirtualDeviceBridge;
  close(): Promise<void>;
}

export async function startTestServer(): Promise<TestServer> {
  const bridge = new MockVirtualDeviceBridge();
  const app = createApp({ bridge });

  await new Promise<void>((resolve) => {
    app.httpServer.listen(0, "127.0.0.1", () => resolve());
  });

  const addr = app.httpServer.address();
  const port = typeof addr === "object" && addr ? addr.port : 0;

  return {
    port,
    baseUrl: `http://127.0.0.1:${port}`,
    wsUrl: `ws://127.0.0.1:${port}/ws`,
    bridgeUrl: `ws://127.0.0.1:${port}/bridge`,
    app,
    bridge,
    close: () => app.close(),
  };
}

export class SignalingTestClient {
  private ws: WebSocket;
  private queue: SignalingMessage[] = [];
  private waiters: Array<{ type: string; resolve: (m: SignalingMessage) => void; reject: (e: Error) => void }> = [];
  clientId: string | null = null;

  private constructor(ws: WebSocket) {
    this.ws = ws;
    ws.on("message", (raw) => {
      const msg = JSON.parse(raw.toString()) as SignalingMessage;
      if (msg.type === "welcome") this.clientId = msg.clientId;
      const idx = this.waiters.findIndex((w) => w.type === msg.type);
      if (idx >= 0) {
        const [waiter] = this.waiters.splice(idx, 1);
        waiter.resolve(msg);
      } else {
        this.queue.push(msg);
      }
    });
  }

  static async connect(wsUrl: string): Promise<SignalingTestClient> {
    const ws = new WebSocket(wsUrl);
    const client = new SignalingTestClient(ws);
    await new Promise<void>((resolve, reject) => {
      ws.once("open", () => resolve());
      ws.once("error", reject);
    });
    if (client.clientId) return client;
    await client.waitFor("welcome");
    return client;
  }

  send(message: SignalingMessage): void {
    this.ws.send(JSON.stringify(message));
  }

  async join(opts: {
    name: string;
    seat: number;
    role: "participant" | "host-mixer";
    hasVideo?: boolean;
    hasAudio?: boolean;
  }): Promise<SignalingMessage> {
    this.send({
      type: "join",
      name: opts.name,
      seat: opts.seat,
      role: opts.role,
      hasVideo: opts.hasVideo ?? true,
      hasAudio: opts.hasAudio ?? true,
    });
    return this.waitFor("room-update");
  }

  waitFor(type: string, timeoutMs = 5000): Promise<SignalingMessage> {
    const queued = this.queue.find((m) => m.type === type);
    if (queued) {
      this.queue = this.queue.filter((m) => m !== queued);
      return Promise.resolve(queued);
    }
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        this.waiters = this.waiters.filter((w) => w.resolve !== resolve);
        reject(new Error(`timeout waiting for ${type}`));
      }, timeoutMs);
      this.waiters.push({
        type,
        resolve: (m) => {
          clearTimeout(timer);
          resolve(m);
        },
        reject,
      });
    });
  }

  close(): void {
    this.ws.close();
  }
}

export async function connectBridge(
  bridgeUrl: string,
): Promise<{ ws: WebSocket; ready: { type: string } }> {
  return new Promise((resolve, reject) => {
    const ws = new WebSocket(bridgeUrl);
    let ready: { type: string } | null = null;
    const timer = setTimeout(() => reject(new Error("bridge connect timeout")), 5000);

    const finish = (value: { ws: WebSocket; ready: { type: string } }) => {
      clearTimeout(timer);
      resolve(value);
    };

    ws.on("message", (raw) => {
      const msg = JSON.parse(raw.toString()) as { type: string };
      if (msg.type === "bridge-ready" && !ready) {
        ready = msg;
        if (ws.readyState === WebSocket.OPEN) {
          finish({ ws, ready });
        }
      }
    });

    ws.once("open", () => {
      if (ready) finish({ ws, ready });
    });

    ws.once("error", (err) => {
      clearTimeout(timer);
      reject(err);
    });
  });
}
