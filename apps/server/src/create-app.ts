import express from "express";
import { createServer, type Server } from "node:http";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { WebSocketServer } from "ws";
import { attachSignaling } from "./signaling.js";
import { attachBridge } from "./bridge.js";
import { Room } from "./room.js";
import { VirtualDeviceBridge, type VirtualDeviceBridgeLike } from "./virtual-devices.js";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

export interface SpidercamApp {
  app: express.Express;
  httpServer: Server;
  room: Room;
  bridge: VirtualDeviceBridgeLike;
  webDist: string;
  close(): Promise<void>;
}

export interface CreateAppOptions {
  webDist?: string;
  bridge?: VirtualDeviceBridgeLike;
  videoDevice?: string;
  audioSink?: string;
}

export function createApp(options: CreateAppOptions = {}): SpidercamApp {
  const webDist = options.webDist ?? path.resolve(__dirname, "../../web/dist");
  const bridge =
    options.bridge ??
    new VirtualDeviceBridge({
      videoDevice: options.videoDevice ?? process.env.SPIDERCAM_VIDEO_DEVICE ?? "/dev/video2",
      audioSink: options.audioSink ?? process.env.SPIDERCAM_AUDIO_SINK ?? "spidercam_sink",
    });

  const app = express();
  const httpServer = createServer(app);
  const room = new Room();

  app.use(express.static(webDist));

  app.get("/api/info", (_req, res) => {
    const addr = httpServer.address();
    const port = typeof addr === "object" && addr ? addr.port : 9847;
    res.json({
      name: "spidercam",
      version: "0.1.0",
      port,
      paths: {
        participant: "/",
        host: "/host.html",
        bridge: "/bridge",
      },
    });
  });

  const signalingWss = new WebSocketServer({ noServer: true });
  attachSignaling(signalingWss, room);

  const bridgeWss = new WebSocketServer({ noServer: true });
  attachBridge(bridgeWss, bridge);

  httpServer.on("upgrade", (request, socket, head) => {
    const pathname = new URL(request.url ?? "/", `http://${request.headers.host ?? "localhost"}`).pathname;
    if (pathname === "/ws") {
      signalingWss.handleUpgrade(request, socket, head, (ws) => {
        signalingWss.emit("connection", ws, request);
      });
      return;
    }
    if (pathname === "/bridge") {
      bridgeWss.handleUpgrade(request, socket, head, (ws) => {
        bridgeWss.emit("connection", ws, request);
      });
      return;
    }
    socket.destroy();
  });

  return {
    app,
    httpServer,
    room,
    bridge,
    webDist,
    close: () =>
      new Promise((resolve, reject) => {
        bridge.stop();
        for (const client of room.getAllClients()) {
          client.ws.close();
        }
        signalingWss.close();
        bridgeWss.close();
        const server = httpServer as Server & {
          closeAllConnections?: () => void;
          closeIdleConnections?: () => void;
        };
        server.closeAllConnections?.();
        server.closeIdleConnections?.();
        httpServer.close((err) => (err ? reject(err) : resolve()));
      }),
  };
}
