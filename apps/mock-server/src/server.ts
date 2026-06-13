import {
  createServer,
  type IncomingMessage,
  type ServerResponse,
} from "node:http";
import { WebSocketServer } from "ws";
import {
  HOST_HOST,
  HOST_PORT,
  PARTICIPANT_HOST,
  PARTICIPANT_PORT,
} from "./config.js";
import { handleDevRoute } from "./dev-routes.js";
import { HostControlHub, handleHostRest, PreviewHub } from "./host-handlers.js";
import { methodNotAllowed, notFound, serveSpaStub } from "./http-util.js";
import {
  ParticipantHub,
  handleParticipantRest,
} from "./participant-handlers.js";
import { PreviewStream } from "./preview-stream.js";
import { RoomModel } from "./room-model.js";
import { ScenarioEngine } from "./scenario-engine.js";

export interface MockServer {
  stop: () => void;
  room: RoomModel;
  engine: ScenarioEngine;
  participantHub: ParticipantHub;
}

export function createMockServer(): MockServer {
  const participantUrl = `http://127.0.0.1:${PARTICIPANT_PORT}/`;
  const room = new RoomModel(participantUrl);
  const engine = new ScenarioEngine(room);
  const preview = new PreviewStream();
  const hostControl = new HostControlHub();
  const previewHub = new PreviewHub();
  const participantHub = new ParticipantHub();

  engine.onState((state) => {
    hostControl.broadcastState(state);
  });

  engine.onActiveVideoChange((activeVideoId, seq) => {
    preview.notifyCut(activeVideoId, seq);
  });

  const notifyParticipants = () => participantHub.scheduleBroadcast(room);
  engine.onSelectionChange(notifyParticipants);

  engine.start();
  preview.start();

  const hostHttp = createServer((req, res) => {
    void handleHostRequest(req, res, room, engine, participantHub);
  });

  const participantHttp = createServer((req, res) => {
    void handleParticipantRequest(req, res);
  });

  const hostWs = new WebSocketServer({ noServer: true });
  const previewWs = new WebSocketServer({ noServer: true });
  const participantWs = new WebSocketServer({ noServer: true });

  hostHttp.on("upgrade", (req, socket, head) => {
    const pathname = new URL(req.url ?? "/", "http://localhost").pathname;
    if (pathname === "/api/v1/ws") {
      hostWs.handleUpgrade(req, socket, head, (ws) => {
        hostWs.emit("connection", ws, req);
      });
      return;
    }
    if (pathname === "/api/v1/ws/preview") {
      previewWs.handleUpgrade(req, socket, head, (ws) => {
        previewWs.emit("connection", ws, req);
      });
      return;
    }
    socket.destroy();
  });

  participantHttp.on("upgrade", (req, socket, head) => {
    const pathname = new URL(req.url ?? "/", "http://localhost").pathname;
    if (pathname === "/api/v1/ws") {
      participantWs.handleUpgrade(req, socket, head, (ws) => {
        participantWs.emit("connection", ws, req);
      });
      return;
    }
    socket.destroy();
  });

  hostWs.on("connection", (ws) => {
    hostControl.add(ws);
    ws.on("message", (data) => {
      void hostControl.handleMessage(ws, Buffer.from(data as Buffer), room);
    });
  });

  previewWs.on("connection", (ws) => {
    previewHub.add(ws, preview);
  });

  participantWs.on("connection", (ws) => {
    const clientId = crypto.randomUUID();
    participantHub.add(ws, clientId, room);
    ws.on("message", (data) => {
      void participantHub.handleMessage(
        ws,
        Buffer.from(data as Buffer),
        clientId,
        room,
      );
    });
  });

  hostHttp.listen(HOST_PORT, HOST_HOST);
  participantHttp.listen(PARTICIPANT_PORT, PARTICIPANT_HOST);

  console.log(`host mock server on http://${HOST_HOST}:${HOST_PORT}`);
  console.log(
    `participant mock server on http://${PARTICIPANT_HOST}:${PARTICIPANT_PORT}`,
  );

  return {
    room,
    engine,
    participantHub,
    stop: () => {
      engine.stop();
      preview.stop();
      hostWs.close();
      previewWs.close();
      participantWs.close();
      hostHttp.close();
      participantHttp.close();
    },
  };
}

async function handleHostRequest(
  req: IncomingMessage,
  res: ServerResponse,
  room: RoomModel,
  engine: ScenarioEngine,
  participantHub: ParticipantHub,
): Promise<void> {
  const pathname = new URL(req.url ?? "/", "http://localhost").pathname;
  const notifyParticipants = () => participantHub.scheduleBroadcast(room);

  if (
    handleDevRoute(req, res, pathname, {
      room,
      engine,
      dropParticipantSockets: () => participantHub.dropAll(),
      notifyParticipants,
    })
  ) {
    return;
  }

  if (await handleHostRest(req, res, pathname, room)) {
    return;
  }

  if (pathname.startsWith("/api/")) {
    notFound(res);
    return;
  }

  if (req.method === "GET") {
    serveSpaStub(res);
    return;
  }

  methodNotAllowed(res);
}

async function handleParticipantRequest(
  req: IncomingMessage,
  res: ServerResponse,
): Promise<void> {
  const pathname = new URL(req.url ?? "/", "http://localhost").pathname;

  if (handleParticipantRest(req, res, pathname)) {
    return;
  }

  if (pathname.startsWith("/api/")) {
    notFound(res);
    return;
  }

  if (req.method === "GET") {
    serveSpaStub(res);
    return;
  }

  methodNotAllowed(res);
}
