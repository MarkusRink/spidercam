# Host preview stream (H.264)

**Target:** `internal/preview/`, `internal/signaling/preview_hub.go`, `web/host/src/preview-stream.ts`

Local monitoring video for the host console. **Not** the Teams output path — preview is a second sink off the same compositor RGBA that feeds v4l2loopback.

See [API.md](../API.md) for `/api/v1/ws/preview` contract.

## Role in the pipeline

```text
Compositor RGBA @ ~30 fps (mixer.Mix)
├── output.Writer.WriteVideo  → v4l2loopback (raw, Teams)
└── preview.Stream.OnFrame    → H.264 @ 15 fps → PreviewHub → host UI WebCodecs
```

**Invariant:** `activeVideoId` in `host-state`, the v4l2 picture, and the preview canvas all derive from the same `SelectionState` / compositor output. OUT/REF meters come from the same `mixer.Frame` via `host-state` JSON — preview binary carries **pixels only**.

## Integration — daemon `onMix` fan-out

```go
// internal/daemon/run.go
func Run(ctx context.Context, cfg Config) error {
	engine := engine.NewEngine(protocol.DefaultHostConfig())
	out, _ := output.Open(ctx, cfg.Output)
	prev := preview.New(preview.Config{
		Width:  cfg.VideoWidth,  // 1280
		Height: cfg.VideoHeight, // 720
		FPS:    15,
		BitrateKbps: 1200,
		Mock:   cfg.Mock,
	})

	controlHub := signaling.NewControlHub()
	previewHub := signaling.NewPreviewHub()

	engine.Run(ctx, func(mix mixer.Frame) {
		_ = out.WriteAudio(mix.PCM)
		_ = out.WriteVideo(mix.Video.RGBA, mix.Video.Width, mix.Video.Height)

		sel := engine.Selection()
		if cut := prev.OnFrame(mix.Video, sel); cut {
			previewHub.BroadcastJSON(protocol.PreviewCutMsg{
				Type:          "preview-cut",
				ActiveVideoID: sel.ActiveVideoID,
				Seq:           prev.Seq(),
			})
		}
	})

	go controlHub.BroadcastLoop(ctx, engine, room, cap) // 50 Hz host-state
	go previewHub.PublishLoop(ctx, prev)                // ~15 Hz binary

	hostSrv := newHostServer(cfg, controlHub, previewHub, embedHostUI)
	// ...
}
```

## Compositor (unchanged contract)

```go
// internal/audio/mixer/compositor.go
func (c *VideoCompositor) Composite(
	streams map[string]*StreamPipeline,
	sel *protocol.SelectionState,
) VideoFrame {
	id := sel.ActiveVideoID
	if id == "" {
		return c.blackFrame
	}
	return streams[id].LatestVideo() // hard cut — no blend
}
```

## Preview stream

```go
// internal/preview/stream.go
package preview

type Config struct {
	Width       int
	Height      int
	FPS         int   // 15
	BitrateKbps int   // 1200
	Mock        bool
}

type Stream struct {
	enc       Encoder
	lastVideo string
	frameN    uint64
	sub       subsample // 30 fps compositor → 15 fps encode (every 2nd)
	out       chan []byte
}

func New(cfg Config) *Stream

func (s *Stream) OnFrame(v VideoFrame, sel *protocol.SelectionState) (cut bool) {
	cut = sel.ActiveVideoID != s.lastVideo
	if cut {
		s.lastVideo = sel.ActiveVideoID
		s.enc.ForceKeyframe()
	}
	if !s.sub.Tick() {
		return cut
	}
	chunk, err := s.enc.Encode(v.RGBA, v.Width, v.Height, time.Now())
	if err == nil && chunk != nil {
		select {
		case s.out <- chunk:
		default: // drop if UI slow — preview is best-effort
		}
	}
	return cut
}

func (s *Stream) Chunks() <-chan []byte { return s.out }
func (s *Stream) Seq() uint64          { return s.frameN }
```

## H.264 encoder

```go
// internal/preview/encoder.go
package preview

type Encoder interface {
	Encode(rgba []byte, w, h int, ts time.Time) (chunk []byte, err error)
	ForceKeyframe()
	Close() error
}

// Production: enc_x264.go + enc_x264.c (libx264 cgo)
// CI/mock:    enc_mock.go — replays testdata/preview/keyframe.h264
```

Encoder settings (libx264):

| Parameter | Value |
|-----------|-------|
| Profile | baseline (`avc1.42E01E`) |
| Preset | `ultrafast` |
| Tune | `zerolatency` |
| B-frames | 0 |
| Keyint | 15 (1 s @ 15 fps) |
| Forced IDR | on `activeVideoId` change |

## Binary chunk format (WebSocket)

Each **binary** WS frame on `/api/v1/ws/preview`:

```text
offset  size  field
0       1     flags — bit0 = keyframe
1       8     PTS microseconds (big-endian uint64)
9       4     NAL length (big-endian uint32)
13      N     H.264 access unit (AVCC-style length-prefixed NALs)
```

JSON control messages on the same preview socket: `preview-stream-init`, `preview-cut`.

## Signaling hub

```go
// internal/signaling/preview_hub.go
func (h *PreviewHub) HandleUpgrade(w http.ResponseWriter, r *http.Request) {
	c := h.upgrade(w, r)
	c.SendJSON(protocol.PreviewStreamInitMsg{
		Type:   "preview-stream-init",
		Codec:  "avc1.42E01E",
		Width:  h.cfg.Width,
		Height: h.cfg.Height,
		FPS:    h.cfg.FPS,
	})
	for chunk := range c.Subscribe(h.chunks) {
		c.SendBinary(chunk)
	}
}
```

```go
// internal/signaling/host_server.go
func (s *HostServer) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/ws", s.controlHub.HandleUpgrade)
	mux.HandleFunc("GET /api/v1/ws/preview", s.previewHub.HandleUpgrade)
	mux.Handle("/api/", s.restAPI)
	mux.Handle("/", s.spa)
	return mux
}
```

Both paths on **`:1235` only** (loopback). No preview traffic on `:1234`.

## Host UI — WebCodecs

```ts
// web/host/src/preview-stream.ts
export class PreviewStream {
  private ws!: WebSocket;
  private decoder!: VideoDecoder;
  private canvas: HTMLCanvasElement;

  constructor(canvas: HTMLCanvasElement) {
    this.canvas = canvas;
  }

  connect(): void {
    this.decoder = new VideoDecoder({
      output: (frame) => {
        const ctx = this.canvas.getContext("2d")!;
        ctx.drawImage(frame, 0, 0, this.canvas.width, this.canvas.height);
        frame.close();
      },
      error: (e) => console.error("preview decode", e),
    });

    this.ws = new WebSocket(`ws://${location.host}/api/v1/ws/preview`);
    this.ws.binaryType = "arraybuffer";
    this.ws.onmessage = (ev) => this.onMessage(ev);
  }

  private onMessage(ev: MessageEvent): void {
    if (typeof ev.data === "string") {
      const msg = JSON.parse(ev.data);
      if (msg.type === "preview-stream-init") {
        this.decoder.configure({ codec: msg.codec, optimizeForLatency: true });
      }
      if (msg.type === "preview-cut") {
        const ctx = this.canvas.getContext("2d")!;
        ctx.fillStyle = "#000";
        ctx.fillRect(0, 0, this.canvas.width, this.canvas.height);
      }
      return;
    }

    const buf = new Uint8Array(ev.data as ArrayBuffer);
    const key = (buf[0] & 0x01) !== 0;
    const pts = Number(new DataView(buf.buffer, buf.byteOffset + 1, 8).getBigUint64(0));
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
}
```

```tsx
// web/host/src/components/OutputPreview.tsx
export function OutputPreview() {
  let canvasRef!: HTMLCanvasElement;
  const store = useSessionStore();

  onMount(() => {
    new PreviewStream(canvasRef).connect();
  });

  return (
    <div class="flex gap-3">
      <canvas ref={canvasRef} width={640} height={360} class="bg-black" />
      <VerticalVuMeter level={store.state?.outLevelDbfs} peak={store.state?.outPeakDbfs} label="OUT" />
      <VerticalVuMeter level={store.state?.reference.rmsDbfs} peak={store.state?.reference.peakDbfs} label="REF" />
    </div>
  );
}
```

Host page opens **two** WebSockets on mount: `/api/v1/ws` (control + `host-state`) and `/api/v1/ws/preview` (video). See [ui/host-console.md](../ui/host-console.md).

## Capture device changes

`set-capture-devices` → `capture.Reopen` changes host camera input. Compositor picks up new host RGBA automatically; preview encoder needs no separate config message.

## Dependencies

### System (build host)

| Package | Purpose |
|---------|---------|
| `libx264-dev` | H.264 encode for preview |
| `pkg-config` | cgo libx264 discovery |

PipeWire / v4l2 deps unchanged — see [capture.md](./capture.md).

### Go modules

| Module | Purpose |
|--------|---------|
| `github.com/pion/webrtc/v4` | Participant ingress (unchanged) |
| `github.com/pion/opus` | Opus decode (unchanged) |
| cgo `libx264` | Preview encoder (`internal/preview/enc_x264.c`) — no pure-Go H.264 encoder in v1 |

No new Go dependency for preview beyond cgo + libx264. Mock encoder avoids libx264 in `--mock` CI when `SPIDERCAM_MOCK_PREVIEW=1`.

### Host UI (npm)

| API | Purpose |
|-----|---------|
| **WebCodecs** (`VideoDecoder`, `EncodedVideoChunk`) | Browser built-in — no npm package |
| SolidJS / Vite / Tailwind | unchanged |

Target browser: Chromium (Chrome/Edge). Host UI opened by `xdg-open` on Linux.

## Testing

| Layer | What |
|-------|------|
| Go unit | `preview` chunk framing, subsample tick, `ForceKeyframe` on id change |
| Go E2E | `GET /api/v1/ws/preview` → `preview-stream-init` → binary key chunk; `testdata/preview/keyframe.h264` |
| Playwright+MSW | **Skip** preview decode — assert OUT/REF meters and timeline from mocked `host-state` only |

See [testing.md](../testing.md).
