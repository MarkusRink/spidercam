import {
  PREVIEW_CODEC,
  PREVIEW_FPS,
  PREVIEW_HEIGHT,
  PREVIEW_WIDTH,
} from "./config.js";
import { loadPreviewKeyframe } from "./fixtures.js";

export type PreviewChunkListener = (chunk: Buffer) => void;
export type PreviewCutListener = (activeVideoId: string, seq: number) => void;

function annexBToAvcc(data: Buffer): Buffer {
  const nals: Buffer[] = [];
  let i = 0;
  while (i < data.length) {
    if (
      i + 3 < data.length &&
      data[i] === 0 &&
      data[i + 1] === 0 &&
      data[i + 2] === 1
    ) {
      i += 3;
    } else if (
      i + 4 < data.length &&
      data[i] === 0 &&
      data[i + 1] === 0 &&
      data[i + 2] === 0 &&
      data[i + 3] === 1
    ) {
      i += 4;
    } else {
      i += 1;
      continue;
    }
    const start = i;
    while (i < data.length) {
      if (
        (i + 3 < data.length &&
          data[i] === 0 &&
          data[i + 1] === 0 &&
          data[i + 2] === 1) ||
        (i + 4 < data.length &&
          data[i] === 0 &&
          data[i + 1] === 0 &&
          data[i + 2] === 0 &&
          data[i + 3] === 1)
      ) {
        break;
      }
      i += 1;
    }
    if (i > start) {
      nals.push(data.subarray(start, i));
    }
  }

  const parts: Buffer[] = [];
  for (const nal of nals) {
    const len = Buffer.alloc(4);
    len.writeUInt32BE(nal.length);
    parts.push(len, nal);
  }
  return Buffer.concat(parts);
}

function packPreviewFrame(
  avcc: Buffer,
  ptsUs: number,
  keyframe: boolean,
): Buffer {
  const header = Buffer.alloc(13);
  header[0] = keyframe ? 0x01 : 0x00;
  header.writeBigUInt64BE(BigInt(ptsUs), 1);
  header.writeUInt32BE(avcc.length, 9);
  return Buffer.concat([header, avcc]);
}

export class PreviewStream {
  private readonly avcc: Buffer;
  private timer: ReturnType<typeof setInterval> | null = null;
  private frameIndex = 0;
  private startTime = Date.now();
  private chunkListeners = new Set<PreviewChunkListener>();
  private cutListeners = new Set<PreviewCutListener>();
  private forceKeyframe = true;

  constructor() {
    this.avcc = annexBToAvcc(loadPreviewKeyframe());
  }

  start(): void {
    if (this.timer) {
      return;
    }
    this.startTime = Date.now();
    const intervalMs = 1000 / PREVIEW_FPS;
    this.timer = setInterval(() => this.emitFrame(), intervalMs);
  }

  stop(): void {
    if (this.timer) {
      clearInterval(this.timer);
      this.timer = null;
    }
  }

  onChunk(listener: PreviewChunkListener): () => void {
    this.chunkListeners.add(listener);
    return () => this.chunkListeners.delete(listener);
  }

  onCut(listener: PreviewCutListener): () => void {
    this.cutListeners.add(listener);
    return () => this.cutListeners.delete(listener);
  }

  notifyCut(activeVideoId: string, seq: number): void {
    this.forceKeyframe = true;
    for (const listener of this.cutListeners) {
      listener(activeVideoId, seq);
    }
  }

  initMessage() {
    return {
      type: "preview-stream-init" as const,
      codec: PREVIEW_CODEC,
      width: PREVIEW_WIDTH,
      height: PREVIEW_HEIGHT,
      fps: PREVIEW_FPS,
    };
  }

  private emitFrame(): void {
    const ptsUs = Math.round(
      (Date.now() - this.startTime) * 1000 +
        this.frameIndex * (1_000_000 / PREVIEW_FPS),
    );
    const keyframe = this.forceKeyframe || this.frameIndex % PREVIEW_FPS === 0;
    this.forceKeyframe = false;
    const chunk = packPreviewFrame(this.avcc, ptsUs, keyframe);
    this.frameIndex += 1;
    for (const listener of this.chunkListeners) {
      listener(chunk);
    }
  }
}
