import { networkInterfaces } from "node:os";
import { createApp } from "./create-app.js";

const PORT = Number(process.env.SPIDERCAM_PORT ?? 9847);
const HOST = process.env.SPIDERCAM_HOST ?? "0.0.0.0";

const { httpServer, bridge, close } = createApp();

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
  void close().then(() => process.exit(0));
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
