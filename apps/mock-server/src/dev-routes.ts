import type { IncomingMessage, ServerResponse } from "node:http";
import type { RoomModel } from "./room-model.js";
import type { ScenarioEngine } from "./scenario-engine.js";
import { parseMixerState } from "./scenario-engine.js";
import { methodNotAllowed, readJsonBody, sendJson } from "./http-util.js";

export interface DevRouteDeps {
  room: RoomModel;
  engine: ScenarioEngine;
  dropParticipantSockets: () => void;
  notifyParticipants: () => void;
}

export function handleDevRoute(
  req: IncomingMessage,
  res: ServerResponse,
  pathname: string,
  deps: DevRouteDeps,
): boolean {
  if (!pathname.startsWith("/dev/scenario/")) {
    return false;
  }

  if (req.method !== "POST") {
    methodNotAllowed(res);
    return true;
  }

  const routeToMatch = pathname.match(/^\/dev\/scenario\/route-to\/([^/]+)$/);
  if (routeToMatch) {
    const id = decodeURIComponent(routeToMatch[1]!);
    if (!deps.engine.routeTo(id)) {
      sendJson(res, 400, { ok: false, error: "unknown participant id" });
      return true;
    }
    deps.notifyParticipants();
    sendJson(res, 200, { ok: true });
    return true;
  }

  const mixerMatch = pathname.match(/^\/dev\/scenario\/mixer-state\/([^/]+)$/);
  if (mixerMatch) {
    const state = parseMixerState(decodeURIComponent(mixerMatch[1]!));
    if (!state) {
      sendJson(res, 400, { ok: false, error: "invalid mixer state" });
      return true;
    }
    deps.engine.setMixerState(state);
    deps.notifyParticipants();
    sendJson(res, 200, { ok: true });
    return true;
  }

  if (pathname === "/dev/scenario/drop-participant-ws") {
    deps.dropParticipantSockets();
    sendJson(res, 200, { ok: true });
    return true;
  }

  if (pathname === "/dev/scenario/add-participant") {
    void handleAddParticipant(req, res, deps.room, deps.notifyParticipants);
    return true;
  }

  if (pathname === "/dev/scenario/remove-participant") {
    void handleRemoveParticipant(req, res, deps.room, deps.notifyParticipants);
    return true;
  }

  sendJson(res, 404, { ok: false, error: "not found" });
  return true;
}

async function handleAddParticipant(
  req: IncomingMessage,
  res: ServerResponse,
  room: RoomModel,
  notifyParticipants: () => void,
): Promise<void> {
  const body = await readJsonBody<{
    name?: string;
    hasVideo?: boolean;
    hasAudio?: boolean;
  }>(req);
  if (!body) {
    sendJson(res, 400, { ok: false, error: "invalid json" });
    return;
  }
  const participant = room.addMockParticipant(
    body.name ?? "Guest",
    body.hasVideo ?? true,
    body.hasAudio ?? true,
  );
  notifyParticipants();
  sendJson(res, 200, { ok: true, participant });
}

async function handleRemoveParticipant(
  req: IncomingMessage,
  res: ServerResponse,
  room: RoomModel,
  notifyParticipants: () => void,
): Promise<void> {
  const body = await readJsonBody<{ id?: string }>(req);
  if (!body?.id) {
    sendJson(res, 400, { ok: false, error: "id required" });
    return;
  }
  if (!room.removeMockParticipant(body.id)) {
    sendJson(res, 404, { ok: false, error: "participant not found" });
    return;
  }
  notifyParticipants();
  sendJson(res, 200, { ok: true });
}
