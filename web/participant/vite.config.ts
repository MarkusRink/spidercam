import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";
import solid from "vite-plugin-solid";

const apiProxy = {
  target: "http://127.0.0.1:1234",
  changeOrigin: true,
  ws: true,
};

export default defineConfig({
  plugins: [solid(), tailwindcss()],
  server: {
    port: 5174,
    strictPort: true,
    proxy: {
      "/api": apiProxy,
    },
  },
  preview: {
    port: 4174,
    strictPort: true,
    host: "127.0.0.1",
    proxy: {
      "/api": apiProxy,
    },
  },
});
