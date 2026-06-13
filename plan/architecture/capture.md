# Native capture

**Target:** `internal/capture/`, `internal/capture/native/`

Linux v1: **PipeWire via thin C shim** (D11). Go calls a stable C API; libpipewire graph linking stays in C.

## Streams

| Stream                 | ID             | Source                                   | Purpose                  |
| ---------------------- | -------------- | ---------------------------------------- | ------------------------ |
| Host mic               | `host`         | PW capture source (user-selected)        | Host talker in mix       |
| Host camera            | `host` (video) | v4l2 device (user-selected)              | Host video in composite  |
| **Playback reference** | `playback-ref` | **Monitor of user-selected output sink** | Room loopback correction |
| Participant A/V        | participant id | Pion RTP (not capture package)           | Remote laptops           |

## Playback reference rationale

The host machine runs Teams. Remote meeting audio plays on **host speakers** (often routed to room TV). That signal is exactly what bleeds into participant laptop mics.

Capture the **speaker monitor** (loopback), not the pre-mix virtual mic feed:

```text
Teams → selected audio sink → physical speakers/TV
              ↓ monitor tap (PW link to pw_stream)
        playback-ref PCM → reference processor
```

**Teams must play to the same sink** selected in the host UI. Hint shown in settings: “Playback output (Teams speaker).”

## Go interface

```go
package capture

type Devices struct {
	Mics     []DeviceInfo
	Cameras  []DeviceInfo
	Sinks    []DeviceInfo // output sinks; monitor port used for playback-ref
}

type DeviceInfo struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type Selection struct {
	MicID    string `json:"micId"`
	CameraID string `json:"cameraId"` // e.g. /dev/video0
	SinkID   string `json:"sinkId"`   // PW sink → monitor for playback-ref
}

type Bundle struct {
	Mic         Mic
	Camera      Camera
	PlaybackRef PlaybackRef
	Selection   Selection
}

func ListDevices() (Devices, error)
func Open(ctx context.Context, sel Selection, sampleRate int) (*Bundle, error)
func (b *Bundle) Reopen(ctx context.Context, sel Selection) error
func (b *Bundle) Close() error
```

```go
type PlaybackRef interface {
	ReadFrame() ([]float32, bool) // 480 samples @ 48kHz, non-blocking
	LevelDbfs() float64
}

type Mic interface {
	ReadFrame() ([]float32, bool)
}

type Camera interface {
	ReadFrame() (rgba []byte, width, height int, ok bool)
}
```

## C shim (`internal/capture/native/`)

```text
pw_capture.h   — stable C API for cgo
pw_capture.c   — pw_main_loop, pw_stream, ring buffers (mic + sink monitor)
pw_devices.c   — enumerate PW sources (mic) and sinks; v4l2 list in Go or list_cameras.c
```

```c
// pw_capture.h (sketch)
typedef struct sp_capture sp_capture;

int sp_list_sinks(sp_device *out, int max, int *count);
int sp_list_sources(sp_device *out, int max, int *count);

sp_capture *sp_capture_open(const char *mic_id, const char *sink_id, int sample_rate);
void sp_capture_close(sp_capture *c);

// non-blocking; returns samples written (0 if underrun)
int sp_capture_read_mic(sp_capture *c, float *buf, int frames);
int sp_capture_read_monitor(sp_capture *c, float *buf, int frames);
```

C responsibilities:

1. `pw_main_loop` + `pw_stream` for mic (capture source port).
2. Second `pw_stream` linked to **monitor port** of selected sink (`PW_DIRECTION_INPUT`).
3. Negotiate F32 @ 48 kHz mono (resample in C if needed).
4. Push PCM into lock-free ring buffers; Go pull reads on engine tick.
5. On `sink_id` / `mic_id` change: tear down and relink streams (`Reopen`).

Go responsibilities:

- cgo wrapper, `ListDevices`, v4l2 camera enumeration (`/dev/video*`).
- In-memory `Selection` for the session (D15); env vars for bootstrap defaults only.
- No direct libpipewire calls from Go.

## Camera

v1: **v4l2** in Go (`github.com/blackjack/webcam` or raw ioctl) — separate from PW audio shim. Listed in `Devices.Cameras` with path + card name.

## Defaults

On first run (no saved selection):

- PW default source (mic)
- PW default sink (playback-ref monitor)
- First available v4l2 node

Env overrides for headless/bootstrap only (seed first open; not persisted):

| Variable                  | Default                       |
| ------------------------- | ----------------------------- |
| `SPIDERCAM_MIC`           | PW default source if UI unset |
| `SPIDERCAM_CAMERA`        | first v4l2                    |
| `SPIDERCAM_PLAYBACK_SINK` | PW default sink               |
| `SPIDERCAM_SAMPLE_RATE`   | `48000`                       |

Device selection from the host UI applies for the session only; restart reverts to env/bootstrap defaults until UI sends `set-capture-devices` again.

## Reopen on UI change

Host settings → `set-capture-devices` → `capture.Reopen`:

1. Stop PW streams in C.
2. Open new mic + sink monitor (+ swap v4l2 fd).
3. Reset loop-delay sample buffers for affected paths.
4. Push updated `capture` labels in next `host-state`.

Brief glitch acceptable; no full process restart.

## Alignment

Playback reference and mic streams share the engine clock. Log initial device latency offset; optional `referenceDelayMs` in [host-config](../domain/host-config.md).

## Build

```makefile
# CGO_ENABLED=1, link -lpipewire-0.3
go build -tags cgo ./cmd/spidercamd
```

CI / mock: `SPIDERCAM_MOCK_CAPTURE=1` skips C layer, feeds silence.
