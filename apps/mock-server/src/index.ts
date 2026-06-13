import { createMockServer } from "./server.js";

const server = createMockServer();

function shutdown(): void {
  server.stop();
  process.exit(0);
}

process.on("SIGINT", shutdown);
process.on("SIGTERM", shutdown);
