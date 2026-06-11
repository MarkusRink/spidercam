import { spawn, type ChildProcess } from "node:child_process";
import { existsSync } from "node:fs";

export interface VirtualDeviceConfig {
  videoDevice: string;
  width: number;
  height: number;
  fps: number;
  audioSink: string;
  sampleRate: number;
  channels: number;
}

export interface VirtualDeviceBridgeLike {
  start(): { video: boolean; audio: boolean; warnings: string[] };
  writeVideoFrame(rgba: Buffer): void;
  writeAudioChunk(pcm: Buffer): void;
  stop(): void;
}

const DEFAULT_CONFIG: VirtualDeviceConfig = {
  videoDevice: "/dev/video2",
  width: 1280,
  height: 720,
  fps: 30,
  audioSink: "spidercam_sink",
  sampleRate: 48000,
  channels: 1,
};

export class VirtualDeviceBridge implements VirtualDeviceBridgeLike {
  private videoProc: ChildProcess | null = null;
  private audioProc: ChildProcess | null = null;
  private config: VirtualDeviceConfig;
  private videoReady = false;
  private audioReady = false;

  constructor(config: Partial<VirtualDeviceConfig> = {}) {
    this.config = { ...DEFAULT_CONFIG, ...config };
  }

  get status(): { video: boolean; audio: boolean; config: VirtualDeviceConfig } {
    return {
      video: this.videoReady,
      audio: this.audioReady,
      config: this.config,
    };
  }

  start(): { video: boolean; audio: boolean; warnings: string[] } {
    const warnings: string[] = [];

    if (existsSync(this.config.videoDevice)) {
      this.startVideo();
      this.videoReady = true;
    } else {
      warnings.push(`Video device ${this.config.videoDevice} not found. Load v4l2loopback or set SPIDERCAM_VIDEO_DEVICE.`);
    }

    this.startAudio();
    this.audioReady = true;

    return { video: this.videoReady, audio: this.audioReady, warnings };
  }

  private startVideo(): void {
    this.videoProc = spawn(
      "ffmpeg",
      [
        "-hide_banner",
        "-loglevel",
        "error",
        "-f",
        "rawvideo",
        "-pix_fmt",
        "rgba",
        "-s",
        `${this.config.width}x${this.config.height}`,
        "-r",
        String(this.config.fps),
        "-i",
        "pipe:0",
        "-f",
        "v4l2",
        this.config.videoDevice,
      ],
      { stdio: ["pipe", "ignore", "pipe"] },
    );

    this.videoProc.stderr?.on("data", (d: Buffer) => {
      console.error("[virtual-video]", d.toString());
    });
  }

  private startAudio(): void {
    this.audioProc = spawn(
      "ffmpeg",
      [
        "-hide_banner",
        "-loglevel",
        "error",
        "-f",
        "s16le",
        "-ar",
        String(this.config.sampleRate),
        "-ac",
        String(this.config.channels),
        "-i",
        "pipe:0",
        "-f",
        "pulse",
        this.config.audioSink,
      ],
      { stdio: ["pipe", "ignore", "pipe"] },
    );

    this.audioProc.stderr?.on("data", (d: Buffer) => {
      console.error("[virtual-audio]", d.toString());
    });
  }

  writeVideoFrame(rgba: Buffer): void {
    if (!this.videoProc?.stdin?.writable) return;
    this.videoProc.stdin.write(rgba);
  }

  writeAudioChunk(pcm: Buffer): void {
    if (!this.audioProc?.stdin?.writable) return;
    this.audioProc.stdin.write(pcm);
  }

  stop(): void {
    this.videoProc?.stdin?.end();
    this.videoProc?.kill();
    this.videoProc = null;
    this.audioProc?.stdin?.end();
    this.audioProc?.kill();
    this.audioProc = null;
    this.videoReady = false;
    this.audioReady = false;
  }
}
