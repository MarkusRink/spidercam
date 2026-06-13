import type { IncomingMessage, ServerResponse } from "node:http";
import type { WebSocket } from "ws";
import type {
  CaptureSelection,
  HostConfig,
  HostInboundMessage,
  RoomState,
  StreamProcessingFlags,
} from "@spidercam/protocol";
import { loadCaptureDevices } from "./fixtures.js";
import type { RoomModel } from "./room-model.js";
import { readJsonBody, sendJson } from "./http-util.js";

export class HostControlHub {
  private clients = new Set<WebSocket>();

  add(ws: WebSocket): void {
    this.clients.add(ws);
    ws.on("close", () => this.clients.delete(ws));
  }

  broadcastState(state: RoomState): void {
    const payload = JSON.stringify({ type: "host-state", state });
    for (const client of this.clients) {
      if (client.readyState === client.OPEN) {
        client.send(payload);
      }
    }
  }

  send(ws: WebSocket, message: unknown): void {
    if (ws.readyState === ws.OPEN) {
      ws.send(JSON.stringify(message));
    }
  }

  async handleMessage(
    ws: WebSocket,
    data: Buffer,
    room: RoomModel,
  ): Promise<void> {
    let msg: HostInboundMessage;
    try {
      msg = JSON.parse(data.toString("utf8")) as HostInboundMessage;
    } catch {
      return;
    }

    switch (msg.type) {
      case "config": {
        room.updateConfig(msg.config as Partial<HostConfig>);
        break;
      }
      case "list-capture-devices":
        this.send(ws, {
          type: "capture-devices",
          devices: loadCaptureDevices(),
        });
        break;
      case "set-capture-devices":
        this.handleSetCaptureDevices(ws, msg.selection, room);
        break;
      case "set-stream-processing":
        room.setStreamProcessing(msg.participantId, msg.flags);
        break;
      case "copy-participant-url":
        this.send(ws, {
          type: "participant-url",
          url: room.getState().participantUrl,
        });
        break;
    }
  }

  private handleSetCaptureDevices(
    ws: WebSocket,
    selection: CaptureSelection,
    room: RoomModel,
  ): void {
    const devices = loadCaptureDevices();
    const mic = devices.mics.find((d) => d.id === selection.micId);
    const camera = devices.cameras.find((d) => d.id === selection.cameraId);
    const sink = devices.sinks.find((d) => d.id === selection.sinkId);
    if (!mic || !camera || !sink) {
      this.send(ws, {
        type: "capture-devices-updated",
        capture: room.getState().capture,
        error: "unknown device id",
      });
      return;
    }
    room.setCaptureSelection(
      selection.micId,
      selection.cameraId,
      selection.sinkId,
    );
    const capture = room.getState().capture;
    capture.micLabel = mic.label;
    capture.cameraLabel = camera.label;
    capture.sinkLabel = sink.label;
    this.send(ws, {
      type: "capture-devices-updated",
      capture,
    });
  }
}

export class PreviewHub {
  private clients = new Map<
    WebSocket,
    { unsubChunk: () => void; unsubCut: () => void }
  >();

  add(
    ws: WebSocket,
    preview: {
      initMessage: () => unknown;
      onChunk: (cb: (chunk: Buffer) => void) => () => void;
      onCut: (cb: (activeVideoId: string, seq: number) => void) => () => void;
      notifyCut: (activeVideoId: string, seq: number) => void;
    },
  ): void {
    this.sendJson(ws, preview.initMessage());
    const unsubChunk = preview.onChunk((chunk) => {
      if (ws.readyState === ws.OPEN) {
        ws.send(chunk);
      }
    });
    const unsubCut = preview.onCut((activeVideoId, seq) => {
      this.sendJson(ws, {
        type: "preview-cut",
        activeVideoId,
        seq,
      });
    });
    this.clients.set(ws, { unsubChunk, unsubCut });
    ws.on("close", () => {
      const subs = this.clients.get(ws);
      subs?.unsubChunk();
      subs?.unsubCut();
      this.clients.delete(ws);
    });
  }

  private sendJson(ws: WebSocket, message: unknown): void {
    if (ws.readyState === ws.OPEN) {
      ws.send(JSON.stringify(message));
    }
  }
}

export async function handleHostRest(
  req: IncomingMessage,
  res: ServerResponse,
  pathname: string,
  room: RoomModel,
): Promise<boolean> {
  if (pathname === "/api/health" && req.method === "GET") {
    sendJson(res, 200, { ok: true });
    return true;
  }

  if (pathname === "/api/v1/host/state" && req.method === "GET") {
    sendJson(res, 200, room.getState());
    return true;
  }

  if (pathname === "/api/v1/host/config" && req.method === "POST") {
    const body = await readJsonBody<Partial<HostConfig>>(req);
    if (!body) {
      sendJson(res, 400, { ok: false, error: "invalid json" });
      return true;
    }
    const error = room.updateConfig(body);
    if (error) {
      sendJson(res, 400, { ok: false, error });
      return true;
    }
    sendJson(res, 200, { ok: true });
    return true;
  }

  if (pathname === "/api/v1/capture/devices" && req.method === "GET") {
    sendJson(res, 200, loadCaptureDevices());
    return true;
  }

  if (pathname === "/api/v1/capture/selection" && req.method === "POST") {
    const body = await readJsonBody<CaptureSelection>(req);
    if (!body) {
      sendJson(res, 400, { error: "invalid json" });
      return true;
    }
    const devices = loadCaptureDevices();
    const mic = devices.mics.find((d) => d.id === body.micId);
    const camera = devices.cameras.find((d) => d.id === body.cameraId);
    const sink = devices.sinks.find((d) => d.id === body.sinkId);
    if (!mic || !camera || !sink) {
      sendJson(res, 500, { error: "unknown device id" });
      return true;
    }
    room.setCaptureSelection(body.micId, body.cameraId, body.sinkId);
    const capture = room.getState().capture;
    capture.micLabel = mic.label;
    capture.cameraLabel = camera.label;
    capture.sinkLabel = sink.label;
    sendJson(res, 200, capture);
    return true;
  }

  if (pathname === "/api/v1/host/stream-processing" && req.method === "POST") {
    const body = await readJsonBody<{
      participantId?: string;
      flags?: StreamProcessingFlags;
    }>(req);
    if (!body?.participantId || !body.flags) {
      sendJson(res, 400, { ok: false, error: "invalid json" });
      return true;
    }
    if (!room.setStreamProcessing(body.participantId, body.flags)) {
      sendJson(res, 400, { ok: false, error: "unknown participant" });
      return true;
    }
    sendJson(res, 200, { ok: true });
    return true;
  }

  return false;
}
