import type { SignalingMessage } from "@spidercam/shared";
import { isSignalingMessage } from "@spidercam/shared";
import type { WebSocket, WebSocketServer } from "ws";
import { v4 as uuid } from "uuid";
import { Room } from "./room.js";

function send(ws: WebSocket, message: SignalingMessage): void {
  if (ws.readyState === ws.OPEN) {
    ws.send(JSON.stringify(message));
  }
}

function broadcast(room: Room, message: SignalingMessage, except?: string): void {
  for (const client of room.getAllClients()) {
    if (client.id !== except) {
      send(client.ws, message);
    }
  }
}

export function attachSignaling(wss: WebSocketServer, room: Room): void {
  wss.on("connection", (ws) => {
    const clientId = uuid();
    let joined = false;

    send(ws, { type: "welcome", clientId, room: room.getState() });

    ws.on("message", (raw) => {
      let message: SignalingMessage;
      try {
        const parsed: unknown = JSON.parse(raw.toString());
        if (!isSignalingMessage(parsed)) return;
        message = parsed;
      } catch {
        return;
      }

      if (message.type === "ping") {
        send(ws, { type: "pong", ts: message.ts, serverTs: Date.now() });
        return;
      }

      if (message.type === "join") {
        const existing = room.getClient(clientId);
        if (existing) return;

        room.addClient(clientId, ws, {
          id: clientId,
          name: message.name,
          seat: message.seat,
          role: message.role,
          hasVideo: message.hasVideo,
          hasAudio: message.hasAudio,
          joinedAt: Date.now(),
        });
        joined = true;

        broadcast(room, { type: "room-update", room: room.getState() });
        return;
      }

      if (!joined) return;

      switch (message.type) {
        case "leave":
          room.removeClient(clientId);
          broadcast(room, { type: "room-update", room: room.getState() });
          ws.close();
          break;

        case "offer":
        case "answer":
        case "ice-candidate": {
          const target = room.getClient(message.to);
          if (target) {
            send(target.ws, message);
          }
          break;
        }

        case "metrics":
          room.updateMetrics(message.from, message.metrics);
          broadcast(room, { type: "room-update", room: room.getState() }, message.from);
          break;

        case "selection":
          room.setSelection(message.selection);
          broadcast(room, { type: "room-update", room: room.getState() });
          break;

        case "config":
          room.updateConfig(message.config);
          broadcast(room, { type: "room-update", room: room.getState() });
          break;

        default:
          break;
      }
    });

    ws.on("close", () => {
      if (joined) {
        room.removeClient(clientId);
        broadcast(room, { type: "room-update", room: room.getState() });
      }
    });
  });
}
