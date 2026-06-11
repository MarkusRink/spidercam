import type { VirtualDeviceBridgeLike } from "../src/virtual-devices.js";

export class MockVirtualDeviceBridge implements VirtualDeviceBridgeLike {
  videoFrames: Buffer[] = [];
  audioChunks: Buffer[] = [];
  started = false;

  start(): { video: boolean; audio: boolean; warnings: string[] } {
    this.started = true;
    return { video: true, audio: true, warnings: [] };
  }

  writeVideoFrame(rgba: Buffer): void {
    this.videoFrames.push(Buffer.from(rgba));
  }

  writeAudioChunk(pcm: Buffer): void {
    this.audioChunks.push(Buffer.from(pcm));
  }

  stop(): void {
    this.started = false;
  }
}
