# Glossar

Kurze Erklärungen zu Begriffen in den Dokumenten.

## Audio & Signalverarbeitung

| Begriff | Bedeutung |
| ------- | --------- |
| **dBFS** | Decibels relative to full scale — digitaler Pegel. |
| **RMS / Peak** | Mittlere bzw. Spitzenamplitude pro 10-ms-Frame. |
| **SNR** | Signal-to-noise ratio — Sprache vs. geschätzter Rauschboden. |
| **VAD** | Voice activity detection — Sprache ja/nein. |
| **Playback reference** | Loopback vom **Host-Lautsprecher** (Teams-Audio) — Stream `playback-ref`. |
| **Echo penalty** | Korrelation Mic↔Reference bei Lag 0; bestraft TV/Raum-Bleed in der Score. |
| **Loop delay / passive loop** | GCC-PHAT ref↔mic — geschätzte Raum-Schleifenlatenz; viele Phasen während Remote spricht. |
| **LoopDelayText** | UI: `~118 ms` oder `—`; ungefähre passive Schätzung. |
| **globalLatencyMs** | Max bekannte Teilnehmer-Loop-Latenz; `—` wenn unbekannt. |
| **Reference ducking** | Dämpft Teilnehmer-Mics bei aktivem Reference-VAD; Slider 0…−12 dB in Settings-Panel. |
| **Frame** | 480 Samples @ 48 kHz = 10 ms. |
| **Gate / Ducking / Limiter** | Pro Stream bzw. Master — implementiert in `internal/audio/mixer/`. |
| **Dual-branch pipeline** | Raw-Analyse (Scores, Echo, Delay) vs. Enhancement-Pfad (AEC, RNNoise → Mixer). |
| **AEC** | WebRTC APM AEC3 pro Stream; `playback-ref` als Far-End; Toggle auf Card. |
| **RNNoise / NS** | Optionale Rauschunterdrückung pro Stream; unabhängig von AEC. |
| **enhancementBudgetPct** | (Σ aecUs + Σ denoiseUs) / 10 ms — DSP-Last im Header. |

## Mixer & Auswahl

| Begriff | Bedeutung |
| ------- | --------- |
| **Selector** | `internal/selector` — Hysterese, Audio-Crossfade. |
| **State timeline** | 45 s Strip: `_` Silence, `L` Locked, `H` Hold (teal), `S` Switch (yellow). |
| **Stream grid** | Feste 168×240 Cards, `grid-cols-5`; host + Teilnehmer. |
| **Score border** | Card-Rahmen-Opacity ∝ `scoreSmooth` — Aktivität/Energie. |
| **TransportBlock** | 2×3 Mono-Grid auf Card: rtt, loss, jitter, buf, fps, A/V. |
| **Main talker / On air** | Gewählter Sprecher (`mainTalkerId`); Host: roter Punkt auf Stream-Card; Participant: Textzeile „On air: …“, roter Punkt im Header wenn geroutet. |
| **REF meter** | Vertikaler Pegel in Preview-Panel (Playback-Reference, nicht „on air“). |

## Netzwerk & Architektur

| Begriff | Bedeutung |
| ------- | --------- |
| **spidercamd** | Go **CLI** — Terminal start, Browser für Host-UI, Media + HTTP embedded. |
| **Pion** | Go-WebRTC-Stack — Hub für Browser-Teilnehmer. |
| **:1234** | Participant HTTP + WS (LAN). |
| **:1235** | Host HTTP + WS (nur Loopback). |
| **host-state** | Volles `RoomState` @ ~50 Hz → Host-UI (`/api/v1/ws`). |
| **participant-view** | Schlankes Update → Participant-UI. |
| **Star topology** | Jeder Browser nur Verbindung zu Pion, nicht untereinander. |
| **--mock** | Entwicklungs-/CI-Modus: Mock-Capture, live Audio-Engine, Mock-Output. |

## Plattform

| Begriff | Bedeutung |
| ------- | --------- |
| **PipeWire + C shim** | Audio-Capture in C (`internal/capture/native/`); Mic + Sink-Monitor via cgo. |
| **Playback output** | Host-UI: PW-Sink-Auswahl — Teams-Lautsprecher; Monitor = Reference. |
| **v4l2loopback** | Virtuelle Webcam für Teams. |
| **spidercam_sink** | Virtuelles PulseAudio-Mic für Teams. |
| **SolidJS** | UI-Framework — dünne SPAs, kein Media. |

## Update-Loops (Host-UI)

| Intervall | Quelle |
| --------- | ------ |
| **~20 ms** | WS `host-state` |
| **rAF** | Meter-Interpolation |
| **15 fps** | H.264 preview on `/api/v1/ws/preview` (hard-cut compositor output) |
| **Preview stream** | `internal/preview` — encodes compositor RGBA for host UI WebCodecs |

## Code-Referenzen

| Konzept | Paket / Datei |
| ------- | ------------- |
| Protokoll-Typen | `internal/protocol/`, `web/protocol/` |
| Audio-Engine | `internal/audio/engine.go` |
| Raumzustand | `internal/room/room.go` |
| Signaling | `internal/signaling/` |
| Host-UI Store | `web/host/src/stores/session-store.tsx` |
| Participant-UI Store | `web/participant/src/stores/participant-store.ts` |
