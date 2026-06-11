export interface MixerOutput {
  videoStream: MediaStream;
  audioStream: MediaStream;
  canvas: HTMLCanvasElement;
}

export class StreamMixer {
  private canvas: HTMLCanvasElement;
  private ctx: CanvasRenderingContext2D;
  private audioCtx: AudioContext;
  private destination: MediaStreamAudioDestinationNode;
  private gainNodes = new Map<string, GainNode>();
  private sources = new Map<string, MediaStreamAudioSourceNode>();
  private animFrame = 0;
  private videoId = "host";
  private audioId = "host";
  private hostVideo: HTMLVideoElement | null = null;
  private peerVideos = new Map<string, HTMLVideoElement>();

  constructor(width = 1280, height = 720) {
    this.canvas = document.createElement("canvas");
    this.canvas.width = width;
    this.canvas.height = height;
    this.ctx = this.canvas.getContext("2d")!;
    this.audioCtx = new AudioContext();
    this.destination = this.audioCtx.createMediaStreamDestination();
  }

  setHostVideo(video: HTMLVideoElement): void {
    this.hostVideo = video;
  }

  setPeerVideo(id: string, video: HTMLVideoElement): void {
    this.peerVideos.set(id, video);
  }

  removePeer(id: string): void {
    this.peerVideos.delete(id);
    const gain = this.gainNodes.get(id);
    if (gain) {
      gain.disconnect();
      this.gainNodes.delete(id);
    }
    const src = this.sources.get(id);
    if (src) {
      src.disconnect();
      this.sources.delete(id);
    }
  }

  setSelection(videoId: string, audioId: string): void {
    this.videoId = videoId;
    this.audioId = audioId;
    this.updateAudioGains();
  }

  connectAudio(id: string, stream: MediaStream): void {
    if (this.sources.has(id)) return;
    const source = this.audioCtx.createMediaStreamSource(stream);
    const gain = this.audioCtx.createGain();
    source.connect(gain);
    gain.connect(this.destination);
    this.sources.set(id, source);
    this.gainNodes.set(id, gain);
    this.updateAudioGains();
  }

  connectHostAudio(stream: MediaStream): void {
    this.connectAudio("host", stream);
  }

  private updateAudioGains(): void {
    for (const [id, gain] of this.gainNodes) {
      gain.gain.value = id === this.audioId ? 1.0 : 0.0;
    }
  }

  start(): MixerOutput {
    const draw = () => {
      this.ctx.fillStyle = "#0a0a0b";
      this.ctx.fillRect(0, 0, this.canvas.width, this.canvas.height);

      let video: HTMLVideoElement | null = null;
      if (this.videoId === "host") {
        video = this.hostVideo;
      } else {
        video = this.peerVideos.get(this.videoId) ?? null;
      }

      if (video && video.readyState >= 2) {
        const vw = video.videoWidth;
        const vh = video.videoHeight;
        if (vw && vh) {
          const scale = Math.min(this.canvas.width / vw, this.canvas.height / vh);
          const dw = vw * scale;
          const dh = vh * scale;
          const dx = (this.canvas.width - dw) / 2;
          const dy = (this.canvas.height - dh) / 2;
          this.ctx.drawImage(video, dx, dy, dw, dh);
        }
      } else {
        this.ctx.fillStyle = "#1a1a1e";
        this.ctx.font = "24px monospace";
        this.ctx.fillStyle = "#6b6b70";
        this.ctx.textAlign = "center";
        this.ctx.fillText(`no video: ${this.videoId}`, this.canvas.width / 2, this.canvas.height / 2);
      }

      this.ctx.fillStyle = "rgba(0,0,0,0.5)";
      this.ctx.fillRect(8, this.canvas.height - 28, 300, 20);
      this.ctx.fillStyle = "#3dd68c";
      this.ctx.font = "12px monospace";
      this.ctx.textAlign = "left";
      this.ctx.fillText(`V:${this.videoId} A:${this.audioId}`, 12, this.canvas.height - 14);

      this.animFrame = requestAnimationFrame(draw);
    };
    draw();

    const videoStream = this.canvas.captureStream(30);
    return {
      videoStream,
      audioStream: this.destination.stream,
      canvas: this.canvas,
    };
  }

  stop(): void {
    cancelAnimationFrame(this.animFrame);
    void this.audioCtx.close();
  }

  getAudioContext(): AudioContext {
    return this.audioCtx;
  }
}
