export class BridgeClient {
  private ws: WebSocket | null = null;
  private audioNode: ScriptProcessorNode | null = null;
  private configured = false;

  async connect(url?: string): Promise<boolean> {
    const wsUrl = url ?? `${location.protocol === "https:" ? "wss" : "ws"}://${location.host}/bridge`;

    return new Promise((resolve) => {
      this.ws = new WebSocket(wsUrl);

      this.ws.onopen = () => resolve(true);
      this.ws.onerror = () => resolve(false);

      this.ws.onmessage = () => {};
    });
  }

  configure(width: number, height: number, sampleRate: number, channels: number): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;
    this.ws.send(
      JSON.stringify({
        type: "bridge-config",
        width,
        height,
        sampleRate,
        channels,
      }),
    );
    this.configured = true;
  }

  attachVideo(canvas: HTMLCanvasElement, fps = 30): () => void {
    const stream = canvas.captureStream(fps);
    const video = document.createElement("video");
    video.srcObject = stream;
    video.muted = true;
    void video.play();

    const captureCanvas = document.createElement("canvas");
    captureCanvas.width = canvas.width;
    captureCanvas.height = canvas.height;
    const ctx = captureCanvas.getContext("2d")!;

    let running = true;
    const sendFrame = () => {
      if (!running || !this.ws || this.ws.readyState !== WebSocket.OPEN) return;
      ctx.drawImage(canvas, 0, 0);
      const imageData = ctx.getImageData(0, 0, captureCanvas.width, captureCanvas.height);
      this.ws.send(imageData.data.buffer);
      requestAnimationFrame(sendFrame);
    };
    sendFrame();

    return () => {
      running = false;
    };
  }

  attachAudio(audioContext: AudioContext, stream: MediaStream): () => void {
    const source = audioContext.createMediaStreamSource(stream);
    const processor = audioContext.createScriptProcessor(4096, 1, 1);
    this.audioNode = processor;

    processor.onaudioprocess = (ev) => {
      if (!this.ws || this.ws.readyState !== WebSocket.OPEN || !this.configured) return;
      const input = ev.inputBuffer.getChannelData(0);
      const pcm = new Int16Array(input.length);
      for (let i = 0; i < input.length; i++) {
        const s = Math.max(-1, Math.min(1, input[i]));
        pcm[i] = s < 0 ? s * 0x8000 : s * 0x7fff;
      }
      const bytes = new Uint8Array(pcm.buffer);
      let binary = "";
      for (let i = 0; i < bytes.length; i++) binary += String.fromCharCode(bytes[i]);
      this.ws.send(
        JSON.stringify({
          type: "audio-chunk",
          data: btoa(binary),
        }),
      );
    };

    source.connect(processor);
    processor.connect(audioContext.destination);

    return () => {
      processor.disconnect();
      source.disconnect();
    };
  }

  close(): void {
    this.ws?.close();
    this.ws = null;
  }
}
