# Use cases

Spidercam bridges laptops in a physical meeting room into one Teams stream (virtual camera + microphone on Linux). A **Go daemon** on the host machine owns all media; **participants** join from a browser on the LAN.

## Actors

| Actor | Shell | Role |
| ----- | ----- | ---- |
| Host operator | Terminal → `spidercamd`, then **browser** at `http://127.0.0.1:1235/` | Monitor mixer, pick mic/cam/speaker, tune settings |
| Host daemon | `spidercamd` CLI (foreground) | Capture, WebRTC hub, mixer, virtual devices, playback reference |
| Participant | Browser at `http://<host-lan>:1234/` | Sends mic/cam via WebRTC to Go |
| Microsoft Teams | On host machine | Consumes v4l2loopback + PulseAudio virtual mic; plays meeting audio on host speakers |

The host operator is **in the Teams meeting** on the same machine as `spidercamd`. Teams speaker output is captured as the **playback reference** for room loopback correction.

## Host use cases

| ID | Goal | Success criteria |
| -- | ---- | ---------------- |
| UC-H1 | Start from terminal | `spidercamd` prints URLs, opens host browser tab, participant LAN URL ready |
| UC-H2 | See what Teams receives | Hard-cut H.264 preview @ 15 fps; OUT vertical meter + dBFS in preview panel; output health dot |
| UC-H3 | Understand routing | 45 s state timeline, red on-air dot, audio crossfade (`S` segments), score border on cards |
| UC-H4 | Monitor channel health | Per-stream card: vertical meter, transport grid, loop text, score border |
| UC-H5 | Diagnose routing issues | State timeline + per-card transport thresholds + daemon logs |
| UC-H6 | Tune aggressiveness | Settings panel: hold, crossfade, ducking, switch margin, score weights; per-card AEC/NS |
| UC-H7 | Reject TV / Teams bleed | Raw echoPenalty + ref duck; optional per-stream AEC on enhancement branch |
| UC-H8 | See room-loop latency | Per-card loop text; header global max or `—`; passive GCC-PHAT |
| UC-H9 | Select capture devices | Mic, webcam, playback output in settings panel; defaults on load; partial hot reopen |
| UC-H10 | Clean noisy mics | Per-stream RNNoise toggle; `NS · Nms` timing on card |
| UC-H11 | Monitor DSP load | Header `enhancementBudgetPct` when AEC/NS active |

With `--mock`, UC-H2–H11 are fully exercised by the live audio engine. In default production mode, signaling and UIs work; mixer metrics are driven by the scenario engine until native capture and WebRTC RTP are wired into `internal/audio`.

## Participant use cases

| ID | Goal | Success criteria |
| -- | ---- | ---------------- |
| UC-P1 | Join quickly | Single screen; default display name `client-{random}`; mic/camera pickers + toggles; Connect toggle |
| UC-P2 | Confirm mic works | Local preview + RMS always; when connected: SNR, loop delay text from server |
| UC-P3 | Know on-air status | Header red dot when routed (`activeAudioId === clientId`); text row **On air: you** / **On air: {name}** / **On air: host** |
| UC-P4 | Leave cleanly | Disconnect → `leave` + stop WebRTC/WS → **same screen**, local preview continues |
| UC-P5 | Survive host outage | **Lost host connection** banner; auto-reconnect with backoff; auto-`join` when host returns; cancel via Disconnect |

Implemented in `web/participant/` with `LiveParticipantSignaling` + `LiveParticipantPeer`.

## User actions

### Host (browser `:1235`, opened by CLI)

| Action | Effect |
| ------ | ------ |
| Run `spidercamd` | Terminal process starts; browser opens host UI (unless `--no-open-browser`) |
| Open host UI manually | `http://127.0.0.1:1235/` if tab closed |
| Copy participant URL | Clipboard `http://<lan-ip>:1234/` |
| Change capture device | Settings panel dropdown → partial `set-capture-devices` → daemon merges and updates capture state |
| Adjust mixer sliders | Settings panel → partial `config` (session RAM) |
| Toggle AEC / NS on stream | Stream card → `set-stream-processing` (session RAM) |

### Daemon (CLI)

| Action | Effect |
| ------ | ------ |
| `spidercamd` | Foreground: listeners, Pion, output probe; logs to terminal |
| `spidercamd --no-open-browser` | Same without `xdg-open` |
| `spidercamd --mock` | Mock capture/output + live audio engine for dev/CI |
| Ctrl+C / SIGTERM | Stop capture, close peers, release virtual devices, exit |

### Participant (browser `:1234`)

| Action | Effect |
| ------ | ------ |
| Edit display name | Cosmetic label; sent on next `join` |
| Pick mic / camera | `getUserMedia` constraints; works connected or not |
| Toggle mic / camera | Enable/disable tracks; preview updates immediately |
| Connect | WS + `join` + WebRTC offer; room metrics appear |
| Disconnect | `leave`, close peer/WS; stay on same screen; local preview continues |
| Host goes offline while connected | Lost-host banner; auto-reconnect + auto-`join` (UC-P5) |

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

| Situation | Behavior |
| --------- | -------- |
| Silence (all scores low) | Near-silence output; `mixerState: SILENCE` |
| Overlapping speech | Single best talker |
| Host speaking (native mic) | Same score-based routing as participants; `HostPriority` weight only |
| Teams remote speaks (playback ref active) | Lower scores on correlated mic streams; duck participants when `referenceDuckDb < 0` |
| TV bleed into laptop mic | Raw `echoPenalty` + ref duck; optional per-stream AEC on enhancement branch |
