import type { WebSocket, WebSocketServer } from "ws";
import { VirtualDeviceBridge } from "./virtual-devices.js";

interface BridgeConfig {
  width: number;
  height: number;
  sampleRate: number;
  channels: number;
}

export function attachBridge(wss: WebSocketServer, bridge: VirtualDeviceBridge): void {
  wss.on("connection", (ws) => {
    let config: BridgeConfig | null = null;

    ws.send(JSON.stringify({ type: "bridge-ready" }));

    ws.on("message", (raw) => {
      if (raw instanceof Buffer) {
        if (config) {
          bridge.writeVideoFrame(raw);
        }
        return;
      }

      try {
        const message = JSON.parse(raw.toString()) as {
          type: string;
          width?: number;
          height?: number;
          sampleRate?: number;
          channels?: number;
          data?: string;
        };

        if (message.type === "bridge-config") {
          config = {
            width: message.width ?? 1280,
            height: message.height ?? 720,
            sampleRate: message.sampleRate ?? 48000,
            channels: message.channels ?? 1,
          };
          bridge.start();
          return;
        }

        if (message.type === "audio-chunk" && message.data) {
          const pcm = Buffer.from(message.data, "base64");
          bridge.writeAudioChunk(pcm);
        }
      } catch {
        return;
      }
    });
  });
}
