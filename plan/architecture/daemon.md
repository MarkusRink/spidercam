# Daemon (CLI)

**Target:** `cmd/spidercamd/`, `internal/daemon/`

`spidercamd` is a **foreground CLI program**: start from a terminal, logs to stdout/stderr, Ctrl+C for clean shutdown. The host opens a **normal browser tab** to the embedded UI.

## Operator workflow

```text
$ spidercamd
spidercamd 0.1.0
  host UI:        http://127.0.0.1:1235/
  participant UI: http://192.168.1.42:1234/
  virtual mic:    spidercam_sink
  virtual cam:    /dev/video4 (spidercam-loopback; auto-detected — D30)
  capture:        mic=Built-in · speaker=HDMI-A · cam=/dev/video0
  → opened host UI in browser (disable with --no-open-browser)
```

1. Run `spidercamd` in a terminal (or tmux).
2. Daemon prints URLs and capture summary.
3. Default: **`xdg-open`** (Linux) / `open` (macOS) host UI at `http://127.0.0.1:1235/`.
4. Operator copies participant URL for the room; configures mic/cam/speaker in host settings if needed.
5. SIGINT/SIGTERM → stop capture, close peers, release virtual devices, exit 0.

No background det fork in v1. Optional **systemd user unit** documented later for always-on installs.

## Responsibilities

- Parse **flags** + env + saved device config
- Start PipeWire capture (C shim) + v4l2 camera
- Start Pion WebRTC hub
- Start audio engine + virtual device output
- Run **two HTTP servers** — each serves embedded Solid UI at `/` and API at `/api/v1/`
- Host: WS `host-state` + `/ws/preview` H.264, REST snapshots, capture device routes — [API.md](../API.md)
- Participant: WS signaling + `participant-view`
- Optional: open system browser once listeners are ready

## Entry

```go
// cmd/spidercamd/main.go
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/markus/spidercam/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
```

```go
// internal/cli/run.go
package cli

func Run(args []string) int {
	fs := flag.NewFlagSet("spidercamd", flag.ExitOnError)
	noOpen := fs.Bool("no-open-browser", false, "do not open host UI in browser")
	hostAddr := fs.String("host-addr", "127.0.0.1:1235", "host UI bind address")
	participantAddr := fs.String("participant-addr", "0.0.0.0:1234", "participant UI bind address")
	mock := fs.Bool("mock", false, "mock capture and output (dev/CI)")
	_ = fs.Parse(args)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg := daemon.LoadConfig(*hostAddr, *participantAddr, *mock)
	if err := daemon.Run(ctx, cfg); err != nil {
		log.Error(err)
		return 1
	}
	return 0
}
```

## Startup sequence

```go
func Run(ctx context.Context, cfg Config) error {
	printBanner(cfg)

	out, err := output.Open(ctx, cfg.Output)
	cap, err := capture.Open(ctx, cfg.Capture.Selection, cfg.SampleRate)
	// ... engine, rtc, servers ...

	if cfg.OpenBrowser {
		go openBrowser("http://" + cfg.HostHTTPAddr + "/")
	}

	printReady(cfg) // URLs, participant LAN IP, device labels
	return g.Wait()
}
```

## Config

```go
package daemon

type Config struct {
	HostHTTPAddr        string // 127.0.0.1:1235
	ParticipantHTTPAddr string // 0.0.0.0:1234
	ParticipantURL      string // computed LAN URL for clipboard
	OpenBrowser         bool
	Mock                bool
	SampleRate          int // 48000
	FrameSamples        int // 480
	VideoWidth          int // 1280
	VideoHeight         int // 720
	Capture             capture.Config // Selection + paths
	Output              output.Config
}

func LoadConfig(hostAddr, participantAddr string, mock bool) Config {
	// flags, env bootstrap for capture; DefaultHostConfig in RAM
}
```

## Run loop

```go
func Run(ctx context.Context, cfg Config) error {
	room := room.New()
	out, err := output.Open(ctx, cfg.Output)
	cap, err := capture.Open(ctx, cfg.Capture.Selection, cfg.SampleRate)
	engine := engine.NewEngine(protocol.DefaultHostConfig())
	engine.AttachReference(cap.PlaybackRef())
	engine.AttachHostMic(cap.Mic())
	engine.AttachHostVideo(cap.Camera())

	prev := preview.New(preview.Config{
		Width: cfg.VideoWidth, Height: cfg.VideoHeight,
		FPS: 15, Mock: cfg.Mock,
	})
	controlHub := signaling.NewControlHub()
	previewHub := signaling.NewPreviewHub(prev)

	engine.Run(ctx, func(mix mixer.Frame) {
		_ = out.WriteAudio(mix.PCM)
		_ = out.WriteVideo(mix.Video.RGBA, mix.Video.Width, mix.Video.Height)
		if cut := prev.OnFrame(mix.Video, engine.Selection()); cut {
			previewHub.BroadcastJSON(protocol.PreviewCutMsg{
				Type: "preview-cut", ActiveVideoID: engine.Selection().ActiveVideoID,
				Seq: prev.Seq(),
			})
		}
	})

	participantSrv := newParticipantServer(cfg, room, rtc, embedParticipantUI)
	hostSrv := newHostServer(cfg, room, engine, cap, controlHub, previewHub, embedHostUI)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error { return controlHub.BroadcastLoop(ctx, engine, room, cap) })
	g.Go(func() error { return previewHub.PublishLoop(ctx) })
	g.Go(func() error { return participantSrv.ListenAndServe(ctx) })
	g.Go(func() error { return hostSrv.ListenAndServe(ctx) })
	return g.Wait()
}
```

## Embedded UI

```go
//go:embed all:../../web/participant/dist
var participantFS embed.FS

//go:embed all:../../web/host/dist
var hostFS embed.FS
```

Build UIs before `go build` (Makefile: `npm run build -w web/participant && npm run build -w web/host`).

## HTTP routing (per listener)

```go
mux.Handle("/api/", apiRouter)       // REST + WS at /api/v1/ws and /api/v1/ws/preview (host only)
mux.Handle("/", spaHandler(distFS))  // Solid SPA fallback
```

## Host state broadcaster

```go
func (h *HostWS) broadcastLoop(ctx context.Context) {
	t := time.NewTicker(20 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			state := h.room.FullState(
				h.engine.Metrics(),
				h.engine.Selection(),
				h.engine.DelayEstimates(),
				h.capture.ActiveSelection(),
			)
			h.hub.BroadcastJSON(protocol.Msg{Type: "host-state", State: state})
		}
	}
}
```

## Logging

Structured text to stderr by default (`log/slog` or std `log`):

- capture open/reopen, sink/mic ids
- participant join/leave
- output health changes
- errors from PW C layer

No log file in v1; operator watches the terminal.

Participants never connect to `:1235`. Host UI never uses `:1234` for state.
