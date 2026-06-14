# Virtual device output

**Target:** `internal/output/`

Replaces the old browser → WS → ffmpeg bridge. Go writes mixed media directly.

**Validated:** [experiments/wave5/p5.5-loopback](../../experiments/wave5/p5.5-loopback/) (v4l2 write), [p5.6-pulse](../../experiments/wave5/p5.6-pulse/) (Pulse sink). Decisions D22, D24, D28, D30.

## Flow

```text
Mixer PCM float32 mono 48kHz ──► PulseAudio spidercam_sink (virtual mic)
Compositor RGBA 1280×720 @ 30fps ──► v4l2loopback (virtual cam — sysfs discovery)
```

## Interface

```go
package output

type Config struct {
    Mock        bool
    VideoDevice string // SPIDERCAM_VIDEO_DEVICE; empty → FindLoopbackDevice()
    AudioSink   string // SPIDERCAM_AUDIO_SINK default spidercam_sink
    Width       int    // 1280
    Height      int    // 720
}

type Writer interface {
    WriteAudio(pcm []float32) error
    WriteVideo(rgba []byte, width, height int) error
    Healthy() bool
    Close() error
}

func Open(ctx context.Context, cfg Config) (Writer, error)
```

## Video (v4l2loopback)

- Compositor produces RGBA (active talker — **hard cut**)
- **Discover** loopback node: sysfs `/sys/class/video4linux/video*/name` contains `loopback` (D30)
- `video_nr=2` modprobe hint does **not** guarantee `/dev/video2` if that node is already in use (P5.5: loopback at `/dev/video4`)
- `VIDIOC_QUERYCAP` with **`V4L2_CAP_VIDEO_OUTPUT` = `0x00000002`** (not `0x20000`); use `_IOR` ioctl direction
- `VIDIOC_ENUM_FMT` → prefer YUYV, then RGB24 (D24)
- `VIDIOC_G_FMT` then `VIDIOC_S_FMT`; on failure use **G_FMT defaults** and write at negotiated size (P5.5)
- Convert RGBA → device pixel format before `write()`

```go
// internal/output/v4l2.go — promote from experiments/wave5/p5.5-loopback/write_bars.go
func FindLoopbackDevice() (string, error) {
    entries, _ := os.ReadDir("/sys/class/video4linux")
    for _, e := range entries {
        name, _ := os.ReadFile("/sys/class/video4linux/" + e.Name() + "/name")
        if strings.Contains(strings.ToLower(string(name)), "loopback") {
            return "/dev/" + e.Name(), nil
        }
    }
    return "", errors.New("no v4l2loopback device found")
}
```

## Audio (PulseAudio)

- **`github.com/jfreymuth/pulse`** (D22) — not `jfreymuth/pulseaudio` (module does not exist)
- Float32 mono @ 48 kHz → sink `spidercam_sink`
- Chunk size: 480 samples per engine frame

```go
// internal/output/pulse.go — promote from experiments/wave5/p5.6-pulse/main.go
import "github.com/jfreymuth/pulse"

func openPulseSink(name string) (*pulseSink, error) {
    client, err := pulse.NewClient()
    sink, err := client.SinkByID(name)
    stream, err := client.NewPlayback(
        pulse.Float32Reader(feedFromRing),
        pulse.PlaybackSampleRate(48000),
        pulse.PlaybackChannels(1),
    )
}
```

## Operator setup (D28 — check-only, no auto-create)

```bash
sudo modprobe v4l2loopback video_nr=2 card_label="spidercam-loopback" exclusive_caps=1
pactl load-module module-null-sink sink_name=spidercam_sink \
  sink_properties=device.description=Spidercam_Virtual_Mic
```

Daemon prints setup commands when devices missing; sets `outputHealthy: false`.

## Environment

| Variable                 | Default                                |
| ------------------------ | -------------------------------------- |
| `SPIDERCAM_VIDEO_DEVICE` | auto-discover loopback via sysfs (D30) |
| `SPIDERCAM_AUDIO_SINK`   | `spidercam_sink`                       |

## Health

`host-state.outputHealthy` — `false` if loopback or Pulse sink open fails. Host header red dot.

## Build

```makefile
# Linux production
CGO_ENABLED=1 go build ./cmd/spidercamd

# CI / dev without hardware
spidercamd --mock
```

`//go:build !cgo || !linux` stub in `output_stub.go` for `go test` without devices (D27).
