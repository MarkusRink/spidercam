import type { IncomingMessage, ServerResponse } from "node:http";
import type { WebSocket } from "ws";
import type { ParticipantInboundMessage } from "@spidercam/protocol";
import type { RoomModel } from "./room-model.js";
import { sendJson } from "./http-util.js";
import { MockWebRTCHub } from "./webrtc-mock.js";

const VIEW_THROTTLE_MS = 100;
const webrtcHub = new MockWebRTCHub();

export class ParticipantHub {
  private clients = new Map<WebSocket, string>();
  private lastBroadcast = 0;
  private pendingBroadcast: ReturnType<typeof setTimeout> | null = null;

  add(ws: WebSocket, clientId: string, room: RoomModel): void {
    this.clients.set(ws, clientId);
    this.send(ws, {
      type: "welcome",
      clientId,
      view: room.viewFor(clientId),
    });
    ws.on("close", () => {
      this.clients.delete(ws);
      webrtcHub.remove(clientId);
    });
  }

  dropAll(): void {
    for (const ws of this.clients.keys()) {
      ws.close(1001, "dev drop");
    }
    this.clients.clear();
  }

  scheduleBroadcast(room: RoomModel): void {
    const now = Date.now();
    const elapsed = now - this.lastBroadcast;
    if (elapsed >= VIEW_THROTTLE_MS) {
      this.broadcastViews(room);
      return;
    }
    if (this.pendingBroadcast) {
      return;
    }
    this.pendingBroadcast = setTimeout(() => {
      this.pendingBroadcast = null;
      this.broadcastViews(room);
    }, VIEW_THROTTLE_MS - elapsed);
  }

  async handleMessage(
    ws: WebSocket,
    data: Buffer,
    clientId: string,
    room: RoomModel,
  ): Promise<void> {
    let msg: ParticipantInboundMessage;
    try {
      msg = JSON.parse(data.toString("utf8")) as ParticipantInboundMessage;
    } catch {
      this.send(ws, { type: "error", message: "invalid json" });
      return;
    }

    switch (msg.type) {
      case "join": {
        if (!msg.name?.trim()) {
          this.send(ws, { type: "error", message: "name required" });
          return;
        }
        room.join(clientId, msg.name.trim(), msg.hasVideo, msg.hasAudio);
        this.send(ws, {
          type: "participant-view",
          view: room.viewFor(clientId),
        });
        this.scheduleBroadcast(room);
        break;
      }
      case "leave":
        room.leave(clientId);
        webrtcHub.remove(clientId);
        this.scheduleBroadcast(room);
        break;
      case "offer":
        await webrtcHub
          .getOrCreate(
            clientId,
            (socket, message) => this.send(socket, message),
            ws,
          )
          .handleOffer(msg.sdp);
        break;
      case "answer":
        break;
      case "ice-candidate":
        await webrtcHub
          .getOrCreate(
            clientId,
            (socket, message) => this.send(socket, message),
            ws,
          )
          .handleIce(msg.candidate);
        break;
    }
  }

  private broadcastViews(room: RoomModel): void {
    this.lastBroadcast = Date.now();
    for (const [ws, clientId] of this.clients) {
      this.send(ws, {
        type: "participant-view",
        view: room.viewFor(clientId),
      });
    }
  }

  private send(ws: WebSocket, message: unknown): void {
    if (ws.readyState === ws.OPEN) {
      ws.send(JSON.stringify(message));
    }
  }
}

export function handleParticipantRest(
  req: IncomingMessage,
  res: ServerResponse,
  pathname: string,
): boolean {
  if (pathname === "/api/health" && req.method === "GET") {
    sendJson(res, 200, { ok: true });
    return true;
  }
  return false;
}
