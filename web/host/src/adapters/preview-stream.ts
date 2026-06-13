import type { PreviewCutMsg, PreviewStreamInitMsg } from "@spidercam/protocol";

function wsUrl(path: string): string {
  const proto = location.protocol === "https:" ? "wss:" : "ws:";
  return `${proto}//${location.host}${path}`;
}

export class PreviewStream {
  private ws: WebSocket | null = null;
  private decoder: VideoDecoder | null = null;
  private canvas: HTMLCanvasElement;
  private awaitingKeyframe = false;

  constructor(canvas: HTMLCanvasElement) {
    this.canvas = canvas;
  }

  connect(): void {
    this.decoder = new VideoDecoder({
      output: (frame) => {
        const ctx = this.canvas.getContext("2d");
        if (ctx) {
          ctx.drawImage(frame, 0, 0, this.canvas.width, this.canvas.height);
        }
        frame.close();
      },
      error: (e) => console.error("preview decode", e),
    });

    const ws = new WebSocket(wsUrl("/api/v1/ws/preview"));
    ws.binaryType = "arraybuffer";
    ws.addEventListener("message", (ev) => this.onMessage(ev));
    ws.addEventListener("close", () => {
      if (this.ws === ws) {
        this.ws = null;
      }
    });
    this.ws = ws;
  }

  disconnect(): void {
    this.ws?.close();
    this.ws = null;
    this.decoder?.close();
    this.decoder = null;
  }

  private onMessage(ev: MessageEvent): void {
    if (typeof ev.data === "string") {
      const msg = JSON.parse(ev.data) as PreviewStreamInitMsg | PreviewCutMsg;
      if (msg.type === "preview-stream-init") {
        this.canvas.width = msg.width;
        this.canvas.height = msg.height;
        this.decoder?.configure({
          codec: msg.codec,
          optimizeForLatency: true,
        });
      }
      if (msg.type === "preview-cut") {
        this.clearCanvas();
        this.awaitingKeyframe = true;
      }
      return;
    }

    if (!this.decoder) {
      return;
    }

    const buf = new Uint8Array(ev.data as ArrayBuffer);
    if (buf.length < 13) {
      return;
    }

    const key = (buf[0]! & 0x01) !== 0;
    if (this.awaitingKeyframe && !key) {
      return;
    }
    if (key) {
      this.awaitingKeyframe = false;
    }

    const pts = Number(
      new DataView(buf.buffer, buf.byteOffset + 1, 8).getBigUint64(0),
    );
    const len = new DataView(buf.buffer, buf.byteOffset + 9, 4).getUint32(0);
    const nal = buf.subarray(13, 13 + len);

    this.decoder.decode(
      new EncodedVideoChunk({
        type: key ? "key" : "delta",
        timestamp: pts,
        data: nal,
      }),
    );
  }

  private clearCanvas(): void {
    const ctx = this.canvas.getContext("2d");
    if (!ctx) {
      return;
    }
    ctx.fillStyle = "#000";
    ctx.fillRect(0, 0, this.canvas.width, this.canvas.height);
  }
}
