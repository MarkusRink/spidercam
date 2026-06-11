import { defineConfig } from "vite";
import { resolve } from "node:path";

export default defineConfig({
  root: ".",
  build: {
    outDir: "dist",
    rollupOptions: {
      input: {
        main: resolve(__dirname, "index.html"),
        host: resolve(__dirname, "host.html"),
      },
    },
  },
  server: {
    proxy: {
      "/ws": { target: "ws://localhost:9847", ws: true },
      "/bridge": { target: "ws://localhost:9847", ws: true },
      "/api": "http://localhost:9847",
    },
  },
});
