# Use cases

Spidercam bridges laptops in a physical meeting room into one Teams stream (virtual camera + microphone on Linux). A **Go daemon** on the host machine owns all media; **participants** join from a browser on the LAN.

## Actors

| Actor           | Shell                                                                 | Role                                                                                 |
| --------------- | --------------------------------------------------------------------- | ------------------------------------------------------------------------------------ |
| Host operator   | Terminal → `spidercamd`, then **browser** at `http://127.0.0.1:1235/` | Monitor mixer, pick mic/cam/speaker, tune settings                                   |
| Host daemon     | `spidercamd` CLI (foreground)                                         | Capture, WebRTC hub, mixer, virtual devices, playback reference                      |
| Participant     | Browser at `http://<host-lan>:1234/`                                  | Sends mic/cam via WebRTC to Go                                                       |
| Microsoft Teams | On host machine                                                       | Consumes v4l2loopback + PulseAudio virtual mic; plays meeting audio on host speakers |

The host operator is **in the Teams meeting** on the same machine as `spidercamd`. Teams speaker output is captured as the **playback reference** for room loopback correction.

## Host use cases

| ID     | Goal                    | Success criteria                                                                               | Detail                                                                                                                   |
| ------ | ----------------------- | ---------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| UC-H1  | Start from terminal     | `spidercamd` prints URLs, opens host browser tab, participant LAN URL ready                    | [architecture/daemon.md](./architecture/daemon.md)                                                                       |
| UC-H2  | See what Teams receives | Hard-cut H.264 preview @ 15 fps; OUT vertical meter + dBFS in preview panel; output health dot | [ui/host-console.md](./ui/host-console.md), [architecture/preview.md](./architecture/preview.md)                         |
| UC-H3  | Understand routing      | 45 s state timeline, red on-air dot, audio crossfade (S segments), score border on cards       | [audio/selector.md](./audio/selector.md)                                                                                 |
| UC-H4  | Monitor channel health  | Per-stream card: vertical meter, transport grid, loop text, score border                       | [ui/host-console.md](./ui/host-console.md)                                                                               |
| UC-H5  | Diagnose routing issues | State timeline + per-card transport thresholds + daemon logs                                   | [ui/host-console.md](./ui/host-console.md)                                                                               |
| UC-H6  | Tune aggressiveness     | Settings panel: hold, crossfade, ducking, switch margin, score weights; per-card AEC/NS        | [domain/host-config.md](./domain/host-config.md)                                                                         |
| UC-H7  | Reject TV / Teams bleed | Raw echoPenalty + ref duck; optional per-stream AEC on enhancement branch                      | [audio/reference-loopback.md](./audio/reference-loopback.md), [audio/echo-cancellation.md](./audio/echo-cancellation.md) |
| UC-H10 | Clean noisy mics        | Per-stream RNNoise toggle; `NS · Nms` timing on card                                           | [audio/enhancement.md](./audio/enhancement.md)                                                                           |
| UC-H11 | Monitor DSP load        | Header `enhancementBudgetPct` when AEC/NS active                                               | [audio/enhancement.md](./audio/enhancement.md)                                                                           |
| UC-H8  | See room-loop latency   | Per-card loop text; header global max or `—`; passive GCC-PHAT                                 | [audio/reference-loopback.md](./audio/reference-loopback.md)                                                             |
| UC-H9  | Select capture devices  | Mic, webcam, playback output in settings panel; hot reopen                                     | [architecture/capture.md](./architecture/capture.md)                                                                     |

## Participant use cases

| ID    | Goal                | Success criteria                                                                                                                                 | Detail                                                   |
| ----- | ------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------- |
| UC-P1 | Join quickly        | Single screen; default display name `client-{random}`; mic/camera pickers + toggles; Connect toggle; **no seat**                                 | [ui/participant-monitor.md](./ui/participant-monitor.md) |
| UC-P2 | Confirm mic works   | Local preview + RMS always; when connected: SNR, **loop delay text** from server                                                                 | [ui/participant-monitor.md](./ui/participant-monitor.md) |
| UC-P3 | Know on-air status  | Header red dot when routed (`activeAudioId === clientId`); text row **On air: you** / **On air: {name}** / **On air: host** — no dot on that row | [domain/types.md](./domain/types.md)                     |
| UC-P4 | Leave cleanly       | Disconnect → `leave` + stop WebRTC/WS → **same screen**, local preview continues                                                                 | [ui/participant-monitor.md](./ui/participant-monitor.md) |
| UC-P5 | Survive host outage | **Lost host connection** banner; auto-reconnect with backoff; auto-`join` when host returns; cancel via Disconnect                               | [ui/participant-monitor.md](./ui/participant-monitor.md) |

## User actions

### Host (browser `:1235`, opened by CLI)

| Action                    | Effect                                                                      |
| ------------------------- | --------------------------------------------------------------------------- |
| Run `spidercamd`          | Terminal process starts; browser opens host UI (unless `--no-open-browser`) |
| Open host UI manually     | `http://127.0.0.1:1235/` if tab closed                                      |
| Copy participant URL      | Clipboard `http://<lan-ip>:1234/`                                           |
| Change capture device     | Settings panel dropdown → `set-capture-devices` → daemon reopens PW/v4l2    |
| Adjust mixer sliders      | Settings panel → partial `config` (session RAM)                             |
| Toggle AEC / NS on stream | Stream card → `set-stream-processing` (session RAM)                         |

### Daemon (CLI)

| Action                         | Effect                                                         |
| ------------------------------ | -------------------------------------------------------------- |
| `spidercamd`                   | Foreground: capture, listeners, Pion, output; logs to terminal |
| `spidercamd --no-open-browser` | Same without `xdg-open`                                        |
| `spidercamd --mock`            | Mock capture/output for dev/CI                                 |
| Ctrl+C / SIGTERM               | Stop capture, close peers, release virtual devices, exit       |

### Participant (browser `:1234`)

| Action                             | Effect                                                               |
| ---------------------------------- | -------------------------------------------------------------------- |
| Edit display name                  | Cosmetic label; sent on next `join`                                  |
| Pick mic / camera                  | `getUserMedia` constraints; works connected or not                   |
| Toggle mic / camera                | Enable/disable tracks; preview updates immediately                   |
| Connect                            | WS + `join` + WebRTC; room metrics appear                            |
| Disconnect                         | `leave`, close peer/WS; stay on same screen; local preview continues |
| Host goes offline while connected  | Lost-host banner; auto-reconnect + auto-`join` (UC-P5)               |
| Retry now / Disconnect (lost-host) | Force immediate reconnect attempt or abort retry loop                |

## Non-goals

- Seat map / geometry-based routing
- Beamforming / blind source separation
- DeepFilterNet / WPE dereverb / cloud enhancement APIs
- Audible latency calibration tones (chirp/MLS)
- Manual talker override
- Multi-host / federated rooms
- Screen-share as first-class video source
- Persisting host config or device selection to disk
- Host UI debug drawer / raw JSON panel / transport footer table

## Policies

| Situation                                 | Behavior                                                                             |
| ----------------------------------------- | ------------------------------------------------------------------------------------ |
| Silence (all scores low)                  | Near-silence output; `mixerState: SILENCE`                                           |
| Overlapping speech                        | Single best talker                                                                   |
| Host speaking (native mic)                | Same score-based routing as participants; `HostPriority` weight only                 |
| Teams remote speaks (playback ref active) | Lower scores on correlated mic streams; duck participants when `referenceDuckDb < 0` |
| TV bleed into laptop mic                  | Raw `echoPenalty` + ref duck; optional per-stream AEC on enhancement branch          |

See [audio/overview.md](./audio/overview.md).
