import express from "express";
import { createServer } from "node:http";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { networkInterfaces } from "node:os";
import { WebSocketServer } from "ws";
import { attachSignaling } from "./signaling.js";
import { attachBridge } from "./bridge.js";
import { Room } from "./room.js";
import { VirtualDeviceBridge } from "./virtual-devices.js";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const PORT = Number(process.env.SPIDERCAM_PORT ?? 9847);
const HOST = process.env.SPIDERCAM_HOST ?? "0.0.0.0";

const app = express();
const httpServer = createServer(app);

const webDist = path.resolve(__dirname, "../../web/dist");
app.use(express.static(webDist));

app.get("/api/info", (_req, res) => {
  res.json({
    name: "spidercam",
    version: "0.1.0",
    addresses: getLocalAddresses(),
    port: PORT,
    paths: {
      participant: "/",
      host: "/host.html",
      bridge: "/bridge",
    },
  });
});

const room = new Room();
const bridge = new VirtualDeviceBridge({
  videoDevice: process.env.SPIDERCAM_VIDEO_DEVICE ?? "/dev/video2",
  audioSink: process.env.SPIDERCAM_AUDIO_SINK ?? "spidercam_sink",
});

const signalingWss = new WebSocketServer({ server: httpServer, path: "/ws" });
attachSignaling(signalingWss, room);

const bridgeWss = new WebSocketServer({ server: httpServer, path: "/bridge" });
attachBridge(bridgeWss, bridge);

httpServer.listen(PORT, HOST, () => {
  const addrs = getLocalAddresses();
  console.log("");
  console.log("  spidercam host running");
  console.log("  ─────────────────────────────────────────");
  for (const addr of addrs) {
    console.log(`  participant → http://${addr}:${PORT}/`);
    console.log(`  host        → http://${addr}:${PORT}/host.html`);
  }
  console.log(`  signaling   → ws://localhost:${PORT}/ws`);
  console.log(`  bridge      → ws://localhost:${PORT}/bridge`);
  console.log("");

  const result = bridge.start();
  for (const w of result.warnings) {
    console.warn(`  ⚠ ${w}`);
  }
});

process.on("SIGINT", () => {
  bridge.stop();
  process.exit(0);
});

function getLocalAddresses(): string[] {
  const nets = networkInterfaces();
  const addrs: string[] = ["localhost", "127.0.0.1"];
  for (const iface of Object.values(nets)) {
    if (!iface) continue;
    for (const net of iface) {
      if (net.family === "IPv4" && !net.internal) {
        addrs.push(net.address);
      }
    }
  }
  return [...new Set(addrs)];
}
