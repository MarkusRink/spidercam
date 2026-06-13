# Audio pipeline overview

**Target:** `internal/audio/*`, `internal/selector/*`

Spidercam’s audio problem splits into two classes:

| Class                        | Symptom                                                                                         | Mitigation                                                                                                                                             |
| ---------------------------- | ----------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **(1) Echo / room loopback** | Far-end Teams speech on TV/speakers re-enters in-room laptop mics; selector may route echo back | `playback-ref` correlation (`echoPenalty`), reference ducking, passive loop delay, **per-stream AEC** — [echo-cancellation.md](./echo-cancellation.md) |
| **(2) Voice isolation**      | Pick intended in-room talker over HVAC, keyboard, reverb, overlapping local speech              | Optional per-stream **RNNoise** — [enhancement.md](./enhancement.md)                                                                                   |

**NS alone does not fix (1).** Neural enhancers treat speech-like TV bleed as speech. Echo-aware methods (correlation, ducking, AEC) are required for false routing.

Teams runs its own AEC/NS on the **mixed** virtual mic. Spidercam must avoid over-suppressing; stacked AEC is acceptable in practice but not guaranteed across Teams updates.

---

## Dual-branch graph

Analysis runs on **calibrated raw** mic (post-HPF, pre-AEC/NS). Enhancement feeds the mixer only.

```text
[Host mic — PipeWire] ─────────────────────────────┐
[Pion participants] → jitter → decode ───────────┤
[playback-ref — sink monitor] ───────────────────┤
                                                 ▼
                              JitterBuffer (per mic stream)
                              HPF 100 Hz
                              Calibration gain (slow EMA)
                                                 │
                    ┌────────────────────────────┴────────────────────────────┐
                    │ ANALYSIS TAP (raw calibrated)                           │
                    │  echoPenalty(ref↔mic @ lag 0)                           │
                    │  GCC-PHAT loop delay (ref↔mic, gated phases)            │
                    │  noise floor, VAD, SNR, score → selector                │
                    └────────────────────────────┬────────────────────────────┘
                                                 │
                    ┌────────────────────────────▼────────────────────────────┐
                    │ ENHANCEMENT BRANCH (output path)                        │
                    │  AEC(ref, mic) — per stream, host toggle, default off   │
                    │  RNNoise — per stream, host toggle, default off       │
                    └────────────────────────────┬────────────────────────────┘
                                                 │
                              Gate + reference ducking gains
                                                 │
[Global mixer @ 10 ms pull] ◄────────────────────┘
    → Selector @ 20 ms (hysteresis, audio crossfade)
    → Equal-power crossfade (audio only)
    → Master soft limiter
    → output.Writer (virtual mic + compositor)
```

**Why split:** AEC/NS before analysis can hide echo energy, break `echoPenalty`, and mislead loop-delay estimation. Card **meters** show **post-enhancement** RMS; **scores** use **raw** acoustics.

---

## Playback reference

`playback-ref` feeds analysis and each stream’s AEC far-end input — never routed to Teams.

1. VAD on reference → `reference.active`
2. Per-stream **echoPenalty** on **raw** mic (lag-0 correlation)
3. **Reference ducking** on enhanced frames when ref active (`referenceDuckDb` slider)
4. **Passive loop delay** — GCC-PHAT on **raw** mic vs ref buffers
5. Same ref frame → each per-stream AEC `ProcessReverseStream`

See [reference-loopback.md](./reference-loopback.md), [echo-cancellation.md](./echo-cancellation.md).

---

## Multi-mic realities

- Each laptop has a **distinct acoustic path** (TV → room → mic): different delay, impulse response, reverb.
- Digital `playback-ref` delay is **common**; acoustic loop differs per stream → **per-stream AEC state**.
- Correlation + ducking does **not remove** echo on the active mic — it avoids **false talkers** and attenuates bleed; AEC reduces leakage on the routed stream.
- Independent laptops are **not** a synchronous mic array — beamforming / blind separation out of scope.

---

## Timing

| Constant                  | Value                   |
| ------------------------- | ----------------------- |
| Sample rate               | 48 kHz                  |
| Frame size                | 480 samples (10 ms)     |
| Selector tick             | 20 ms (50 Hz)           |
| Score EMA α               | 0.1 (~200 ms)           |
| Loop delay publish        | 3 s (default)           |
| RNNoise effective latency | ~10–20 ms (overlap-add) |
| AEC algorithmic delay     | ~0–20 ms (APM AEC3)     |

**CPU budget:** entire engine tick must finish inside 10 ms with margin. Profile `aecUs` + `denoiseUs` per stream and `enhancementBudgetPct` globally — [enhancement.md](./enhancement.md), [echo-cancellation.md](./echo-cancellation.md).

---

## Stage ownership

| Stage                    | Package           | Spec                                             |
| ------------------------ | ----------------- | ------------------------------------------------ |
| Pull clock + fan-in      | `audio/engine`    | [frame-engine.md](./frame-engine.md)             |
| Echo / loopback analysis | `audio/reference` | [reference-loopback.md](./reference-loopback.md) |
| AEC                      | `audio/aec`       | [echo-cancellation.md](./echo-cancellation.md)   |
| Per-stream DSP           | `audio/processor` | [stream-processor.md](./stream-processor.md)     |
| Voice isolation          | `audio/enhance`   | [enhancement.md](./enhancement.md)               |
| Math                     | `audio/math`      | [math.md](./math.md)                             |
| Routing                  | `selector`        | [selector.md](./selector.md)                     |
| Mix + video              | `audio/mixer`     | [mixer.md](./mixer.md)                           |

---

## Policies

| Policy                     | Choice                                                          |
| -------------------------- | --------------------------------------------------------------- |
| Overlapping speech         | Single best talker                                              |
| Silence output             | Near-silence (gated)                                            |
| Teams / TV bleed (routing) | Raw `echoPenalty` + ref duck                                    |
| Teams / TV bleed (output)  | Per-stream AEC when enabled                                     |
| Host mic                   | Same pipeline as participants                                   |
| Voice isolation            | Optional RNNoise per stream; default **off**                    |
| AEC                        | Optional per stream; default **off**                            |
| Analysis vs output         | Raw tap for scores/echo/delay; enhanced for mixer + card meters |

---

## Out of scope

- DeepFilterNet / WPE dereverb
- Multi-mic beamforming / blind separation
- Custom NLMS AEC (use WebRTC APM AEC3)
- Cloud enhancement APIs
