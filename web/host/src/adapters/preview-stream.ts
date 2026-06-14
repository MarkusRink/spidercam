import type { PreviewCutMsg, PreviewStreamInitMsg } from "@spidercam/protocol";

function wsUrl(path: string): string {
  const proto = location.protocol === "https:" ? "wss:" : "ws:";
  return `${proto}//${location.host}${path}`;
}

function parseAvccNals(avcc: Uint8Array): Uint8Array[] {
  const nals: Uint8Array[] = [];
  let i = 0;
  while (i + 4 <= avcc.length) {
    const len =
      (avcc[i]! << 24) |
      (avcc[i + 1]! << 16) |
      (avcc[i + 2]! << 8) |
      avcc[i + 3]!;
    i += 4;
    if (len <= 0 || i + len > avcc.length) {
      break;
    }
    nals.push(avcc.subarray(i, i + len));
    i += len;
  }
  return nals;
}

function buildAvcDecoderDescription(avcc: Uint8Array): Uint8Array | undefined {
  let sps: Uint8Array | undefined;
  let pps: Uint8Array | undefined;
  for (const nal of parseAvccNals(avcc)) {
    const type = nal[0]! & 0x1f;
    if (type === 7) {
      sps = nal;
    }
    if (type === 8) {
      pps = nal;
    }
  }
  if (!sps || !pps) {
    return undefined;
  }
  const desc = new Uint8Array(11 + sps.length + pps.length);
  let o = 0;
  desc[o++] = 1;
  desc[o++] = sps[1]!;
  desc[o++] = sps[2]!;
  desc[o++] = sps[3]!;
  desc[o++] = 0xff;
  desc[o++] = 0xe1;
  desc[o++] = (sps.length >> 8) & 0xff;
  desc[o++] = sps.length & 0xff;
  desc.set(sps, o);
  o += sps.length;
  desc[o++] = 1;
  desc[o++] = (pps.length >> 8) & 0xff;
  desc[o++] = pps.length & 0xff;
  desc.set(pps, o);
  return desc;
}

export class PreviewStream {
  private ws: WebSocket | null = null;
  private decoder: VideoDecoder | null = null;
  private canvas: HTMLCanvasElement;
  private awaitingKeyframe = true;
  private codec = "";
  private decoderConfigured = false;

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
      error: (e) => {
        console.error("preview decode", e);
      },
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
    this.decoderConfigured = false;
    this.codec = "";
  }

  private onMessage(ev: MessageEvent): void {
    if (typeof ev.data === "string") {
      const msg = JSON.parse(ev.data) as PreviewStreamInitMsg | PreviewCutMsg;
      if (msg.type === "preview-stream-init") {
        this.canvas.width = msg.width;
        this.canvas.height = msg.height;
        this.codec = msg.codec;
        this.decoderConfigured = false;
        this.awaitingKeyframe = true;
      }
      if (msg.type === "preview-cut") {
        this.clearCanvas();
        this.awaitingKeyframe = true;
        this.decoderConfigured = false;
        if (this.decoder?.state === "configured") {
          this.decoder.reset();
        }
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
    const len = new DataView(buf.buffer, buf.byteOffset + 9, 4).getUint32(0);
    const nal = buf.subarray(13, 13 + len);

    if (!this.decoderConfigured) {
      const description = buildAvcDecoderDescription(nal);
      if (!key && !description) {
        return;
      }
      if (this.decoder.state === "configured") {
        this.decoder.reset();
      }
      this.decoder.configure({
        codec: this.codec,
        ...(description ? { description } : {}),
        optimizeForLatency: true,
      });
      this.decoderConfigured = true;
      this.awaitingKeyframe = false;
    } else if (this.awaitingKeyframe && !key) {
      return;
    } else if (key) {
      this.awaitingKeyframe = false;
    }

    const pts = Number(
      new DataView(buf.buffer, buf.byteOffset + 1, 8).getBigUint64(0),
    );

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
