import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";
import solid from "vite-plugin-solid";

const apiProxy = {
  target: "http://127.0.0.1:1235",
  ws: true,
};

export default defineConfig({
  plugins: [solid(), tailwindcss()],
  server: {
    port: 5175,
    strictPort: true,
    proxy: {
      "/api": apiProxy,
      "/dev": apiProxy,
    },
  },
  preview: {
    port: 4175,
    strictPort: true,
    host: "127.0.0.1",
    proxy: {
      "/api": apiProxy,
      "/dev": apiProxy,
    },
  },
});
