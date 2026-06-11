import type { RoomState, SignalingMessage } from "@spidercam/shared";

type Handler = (msg: SignalingMessage) => void;

export class SignalingClient {
  private ws: WebSocket | null = null;
  private handlers = new Set<Handler>();
  clientId: string | null = null;
  room: RoomState | null = null;

  connect(url?: string): Promise<string> {
    const wsUrl = url ?? `${location.protocol === "https:" ? "wss" : "ws"}://${location.host}/ws`;

    return new Promise((resolve, reject) => {
      this.ws = new WebSocket(wsUrl);

      this.ws.onopen = () => resolve(this.clientId ?? "");

      this.ws.onerror = () => reject(new Error("WebSocket connection failed"));

      this.ws.onmessage = (ev) => {
        try {
          const msg = JSON.parse(ev.data as string) as SignalingMessage;
          if (msg.type === "welcome") {
            this.clientId = msg.clientId;
            this.room = msg.room;
          }
          if (msg.type === "room-update") {
            this.room = msg.room;
          }
          for (const h of this.handlers) h(msg);
        } catch {
          return;
        }
      };
    });
  }

  onMessage(handler: Handler): () => void {
    this.handlers.add(handler);
    return () => this.handlers.delete(handler);
  }

  send(message: SignalingMessage): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    }
  }

  close(): void {
    this.ws?.close();
    this.ws = null;
  }

  measureLatency(): Promise<number> {
    return new Promise((resolve) => {
      const ts = Date.now();
      const unsub = this.onMessage((msg) => {
        if (msg.type === "pong" && msg.ts === ts) {
          unsub();
          resolve(Date.now() - ts);
        }
      });
      this.send({ type: "ping", ts });
      setTimeout(() => {
        unsub();
        resolve(-1);
      }, 3000);
    });
  }
}
