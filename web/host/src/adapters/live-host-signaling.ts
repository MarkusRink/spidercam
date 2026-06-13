import type {
  CaptureDevices,
  CaptureSelection,
  CaptureState,
  HostConfig,
  HostControlMessage,
  HostInboundMessage,
  RoomState,
} from "@spidercam/protocol";
import type { HostSignaling } from "./host-signaling.js";

function wsUrl(path: string): string {
  const proto = location.protocol === "https:" ? "wss:" : "ws:";
  return `${proto}//${location.host}${path}`;
}

export class LiveHostSignaling implements HostSignaling {
  private ws: WebSocket | null = null;
  private stateHandlers = new Set<(state: RoomState) => void>();
  private captureDevicesHandlers = new Set<(devices: CaptureDevices) => void>();
  private captureUpdatedHandlers = new Set<
    (capture: CaptureState, error?: string) => void
  >();
  private participantUrlHandlers = new Set<(url: string) => void>();

  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      const ws = new WebSocket(wsUrl("/api/v1/ws"));
      this.ws = ws;

      ws.addEventListener("open", () => resolve(), { once: true });
      ws.addEventListener(
        "error",
        () => reject(new Error("host signaling websocket failed")),
        { once: true },
      );
      ws.addEventListener("message", (ev) => this.onMessage(ev));
      ws.addEventListener("close", () => {
        if (this.ws === ws) {
          this.ws = null;
        }
      });
    });
  }

  disconnect(): void {
    this.ws?.close();
    this.ws = null;
  }

  onState(handler: (state: RoomState) => void): () => void {
    this.stateHandlers.add(handler);
    return () => this.stateHandlers.delete(handler);
  }

  onCaptureDevices(handler: (devices: CaptureDevices) => void): () => void {
    this.captureDevicesHandlers.add(handler);
    return () => this.captureDevicesHandlers.delete(handler);
  }

  onCaptureUpdated(
    handler: (capture: CaptureState, error?: string) => void,
  ): () => void {
    this.captureUpdatedHandlers.add(handler);
    return () => this.captureUpdatedHandlers.delete(handler);
  }

  onParticipantUrl(handler: (url: string) => void): () => void {
    this.participantUrlHandlers.add(handler);
    return () => this.participantUrlHandlers.delete(handler);
  }

  send(msg: HostInboundMessage): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(msg));
    }
  }

  sendConfig(partial: Partial<HostConfig>): void {
    this.send({ type: "config", config: partial });
  }

  listCaptureDevices(): void {
    this.send({ type: "list-capture-devices" });
  }

  setCaptureDevices(selection: CaptureSelection): void {
    this.send({ type: "set-capture-devices", selection });
  }

  setStreamProcessing(
    participantId: string,
    flags: import("@spidercam/protocol").StreamProcessingFlags,
  ): void {
    this.send({ type: "set-stream-processing", participantId, flags });
  }

  copyParticipantUrl(): void {
    this.send({ type: "copy-participant-url" });
  }

  private onMessage(ev: MessageEvent): void {
    let msg: HostControlMessage;
    try {
      msg = JSON.parse(String(ev.data)) as HostControlMessage;
    } catch {
      return;
    }

    switch (msg.type) {
      case "host-state":
        for (const handler of this.stateHandlers) {
          handler(msg.state);
        }
        break;
      case "capture-devices":
        for (const handler of this.captureDevicesHandlers) {
          handler(msg.devices);
        }
        break;
      case "capture-devices-updated":
        for (const handler of this.captureUpdatedHandlers) {
          handler(msg.capture, msg.error);
        }
        break;
      case "participant-url":
        for (const handler of this.participantUrlHandlers) {
          handler(msg.url);
        }
        break;
    }
  }
}

export async function fetchHostState(): Promise<RoomState> {
  const res = await fetch("/api/v1/host/state");
  if (!res.ok) {
    throw new Error(`host state fetch failed: ${res.status}`);
  }
  return res.json() as Promise<RoomState>;
}

export async function fetchCaptureDevices(): Promise<CaptureDevices> {
  const res = await fetch("/api/v1/capture/devices");
  if (!res.ok) {
    throw new Error(`capture devices fetch failed: ${res.status}`);
  }
  return res.json() as Promise<CaptureDevices>;
}
