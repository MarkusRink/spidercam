import type { WebSocketServer } from "ws";
import type { VirtualDeviceBridgeLike } from "./virtual-devices.js";

interface BridgeConfig {
  width: number;
  height: number;
  sampleRate: number;
  channels: number;
}

export function attachBridge(wss: WebSocketServer, bridge: VirtualDeviceBridgeLike): void {
  wss.on("connection", (ws) => {
    let config: BridgeConfig | null = null;

    ws.send(JSON.stringify({ type: "bridge-ready" }));

    const handleControl = (message: {
      type: string;
      width?: number;
      height?: number;
      sampleRate?: number;
      channels?: number;
      data?: string;
    }): void => {
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
    };

    ws.on("message", (raw) => {
      const buffer = Buffer.isBuffer(raw) ? raw : Buffer.from(raw as ArrayBuffer);

      if (config && buffer.length > 0 && buffer[0] !== 0x7b) {
        bridge.writeVideoFrame(buffer);
        return;
      }

      try {
        handleControl(JSON.parse(buffer.toString()));
      } catch {
        if (config) {
          bridge.writeVideoFrame(buffer);
        }
      }
    });
  });
}
