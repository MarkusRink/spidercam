# spidercam

Bridge local meeting-room laptops into a single Teams stream via virtual webcam and microphone on Linux.

## Problem

In a hybrid meeting room, one laptop runs Teams and shares a screen to the TV. People on the far side of the table are hard to hear and only the host is visible on camera.

## Solution

```
┌─────────────┐  WebRTC   ┌──────────────┐  v4l2loopback  ┌───────┐
│ participant │ ────────► │  host mixer  │ ─────────────► │ Teams │
│  (browser)  │           │  + selector  │  PulseAudio    │       │
└─────────────┘           └──────────────┘                └───────┘
```

1. **Host** runs `spidercam` on your Linux laptop and opens the host dashboard.
2. **Participants** open the connect page on their laptops (`http://<host-ip>:9847/`).
3. Each person allows webcam/mic and picks their **seat** around the table.
4. The host mixer selects the best video/audio sources and outputs a single stream to virtual devices.
5. Point Teams at the virtual camera (`/dev/video2`) and microphone (`spidercam_mic`).

## Source selection

| Situation | Video | Audio |
|-----------|-------|-------|
| Connected person speaks | That person | That person |
| Silence | Host (default) | Host (default) |
| Unconnected person speaks (picked up by host mic) | Host | Nearest **connected** seat |

Seat numbers are arranged around the table (0 … N−1). The host configures how many seats exist and which seat they occupy.

## Requirements

- Linux (Ubuntu 22.04+ recommended)
- Node.js 20+
- `ffmpeg`
- [v4l2loopback](https://github.com/umlaeute/v4l2loopback) kernel module
- PulseAudio or PipeWire with PulseAudio compatibility

## Setup

```bash
# virtual devices (once per boot)
sudo bash scripts/setup-virtual-devices.sh

# install & build
npm install
npm run build

# start host
npm start
```

Open:

- **Host dashboard**: http://localhost:9847/host.html
- **Participant connect**: http://\<your-lan-ip\>:9847/

### Teams

| Device | Value |
|--------|-------|
| Camera | `spidercam` / `/dev/video2` |
| Microphone | `spidercam_mic` |

### Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SPIDERCAM_PORT` | `9847` | HTTP/WebSocket port |
| `SPIDERCAM_HOST` | `0.0.0.0` | Bind address |
| `SPIDERCAM_VIDEO_DEVICE` | `/dev/video2` | v4l2loopback device |
| `SPIDERCAM_AUDIO_SINK` | `spidercam_sink` | PulseAudio sink name |

## Development

```bash
npm install
npm run dev        # server with hot reload (port 9847)
```

## Testing

```bash
npm run build
npm run test           # unit + integration
npm run test:unit      # selector, messages, room
npm run test:integration  # signaling, bridge, HTTP (in-process server)
npm run test:e2e       # Playwright browser tests (fake cam/mic)
npm run test:all       # everything
```

CI runs all three layers on every push via GitHub Actions (`.github/workflows/ci.yml`):

| Job | What it covers |
|-----|----------------|
| **unit** | Source selector, message validation, room state |
| **integration** | WebSocket signaling relay, bridge pipe, static HTTP |
| **e2e** | Participant connect, host mixer start, multi-client room flow |

In another terminal, for frontend hot reload during development:

```bash
npm run dev -w @spidercam/web
```

## Host UI

The host dashboard is intentionally minimal and technical:

- **Output preview** with active video/audio source labels
- **Per-participant stream cards** with level meters and V/A badges
- **Metrics table**: RTT, packet loss, jitter, FPS, bitrate
- **Bridge status** for the virtual device pipe

## Calibration (planned)

A future release will add speaker test-tone calibration and ambient noise profiling to improve seat-distance audio routing. The seat model and selector already support this direction.

## Architecture

| Package | Role |
|---------|------|
| `apps/server` | HTTP server, WebSocket signaling, virtual device bridge |
| `apps/web` | Participant connect page + host dashboard |
| `packages/shared` | Types, signaling protocol, source selector |

## License

MIT
