# Virtual device output

**Target:** `internal/output/`

Replaces the old browser → WS → ffmpeg bridge. Go writes mixed media directly.

## Flow

```text
Mixer PCM float32 mono 48kHz ──► PulseAudio spidercam_sink (virtual mic)
Compositor RGBA 1280×720 @ 30fps ──► v4l2loopback /dev/video2 (virtual cam)
```

## Interface

```go
package output

type Writer interface {
	WriteAudio(pcm []float32) error
	WriteVideo(rgba []byte, width, height int) error
	Healthy() bool
}

func Open(ctx context.Context, cfg Config) (Writer, error)
```

## Video

- Internal compositor produces RGBA (active talker video — **hard cut**, no blend)
- Write raw frames to v4l2loopback (existing kernel module setup from README)
- Optional ffmpeg helper subprocess only if pixel format conversion needed — prefer direct write

## Audio

- Float32 mono → PulseAudio sink `spidercam_sink` (Teams selects as microphone input)
- Chunk size matches engine frame (480 samples) or aggregate to PA buffer size

## Environment

| Variable | Default |
|----------|---------|
| `SPIDERCAM_VIDEO_DEVICE` | `/dev/video2` |
| `SPIDERCAM_AUDIO_SINK` | `spidercam_sink` |

## Health

`host-state` includes `outputHealthy: bool` — false if v4l2 open fails or sink missing. Host UI shows red dot in header.
