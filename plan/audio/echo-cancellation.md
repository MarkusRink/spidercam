# Acoustic echo cancellation (problem class 1)

**Target:** `internal/audio/aec/`

Far-end Teams speech plays on host speakers/TV → room acoustics → in-room laptop mics. Spidercam must not treat that bleed as a local talker or send it back to Teams.

Teams AEC on the virtual mic **cannot** cancel bleed that enters **before** the mix on separate devices.

---

## Layered mitigation

| Layer | Mechanism | Removes echo? | Routing | Mix output |
|-------|-----------|---------------|---------|------------|
| Analysis | `echoPenalty` (corr @ lag 0 on **raw** mic) | No | Lowers score | No |
| Mix gain | Reference ducking (`referenceDuckDb`) | Attenuates | No | Yes |
| Enhancement | **WebRTC APM AEC3** per stream | Yes (adaptive) | No | Yes |
| Diagnostic | Passive loop delay (GCC-PHAT on **raw**) | No | UI only | No |

Correlation + ducking prevent **false talkers**. AEC cleans the **active** mic path when enabled.

---

## WebRTC APM AEC3 (chosen)

| Property | Value |
|----------|-------|
| Type | Frequency-domain adaptive filter + delay estimator + NLP / residual suppression |
| Reference | **Required** — `playback-ref` each 10 ms |
| Block size | 10 ms @ 48 kHz (matches engine) |
| CPU / stream | Typically &lt;1–2% of one core @ 48 kHz mono |
| License | BSD-like (WebRTC `audio_processing` subset) |
| Integration | C++ `AudioProcessing` shim → cgo |

**One AEC instance per mic stream** (host + each participant). Same `playback-ref` frame fed to each `ProcessReverseStream`; each `ProcessStream` returns echo-cancelled mic for the enhancement branch.

### Alternatives considered

| Option | Why not primary |
|--------|-----------------|
| SpeexDSP MDF | Aging; weaker in reverberant TV rooms |
| Custom NLMS/FDAF | High R&D for DTD + NLP stack |
| Neural AEC (research) | Immature OSS; domain risk |
| Correlation only | Does not cancel on active mic |

---

## Interface

```go
package aec

type Processor interface {
	ProcessReverse(ref []float32)        // playback-ref, 480 samples — call before ProcessNear each tick
	ProcessNear(mic []float32) []float32 // echo-cancelled output
	Reset()
	Stats() Stats
}

type Stats struct {
	ErleDb    float64 `json:"erleDb"`    // EMA, optional UI
	Converged bool    `json:"converged"`
	ProcessUs float64 `json:"processUs"` // wall time EMA
}

type Config struct {
	Enabled bool // per-stream; default false
}

func New(cfg Config) (Processor, error) // build tag: aec_webrtc | aec_passthrough
```

### Build layout

```text
native/aec/
  apm_shim.cpp          // webrtc::AudioProcessing wrapper
internal/audio/aec/
  apm.go                // //go:build linux && cgo
  passthrough.go        // //go:build mock || !cgo
```

Extract `modules/audio_processing` from WebRTC tree (community forks exist). Static link subset; no full browser stack.

### Pipeline placement

```text
raw tap ─────────────────────────────────────────► analysis (unchanged)
raw ──► [AEC enabled?] ──► [RNNoise enabled?] ──► gate/duck ──► mixer
         ▲
    playback-ref (ProcessReverse each tick)
```

`echoPenalty` and loop delay always use **pre-AEC** raw mic.

---

## Per-stream control

Session RAM — `StreamProcessingFlags` in [domain/types.md](../domain/types.md); not in `HostConfig`.

Default both **false** on join. Host toggles on stream card → WS `set-stream-processing` ([domain/messages.md](../domain/messages.md)).

Toggle on → create AEC state; off → passthrough + destroy (or keep warm — implementation choice).

---

## Integration constraints

### Delay alignment

- AEC needs `playback-ref` and mic aligned within ~1–2 ms (acoustic tail handled by filter).
- `ReferenceDelayMs` — fixed trim on ref for scoring; share with AEC delay estimator when tuned.
- **Risk:** clock drift between PipeWire monitor and WebRTC-decoded frames → per-stream resample offset tracking if ERLE poor.

### Double-talk

When local and remote speak together, adaptation pauses; residual echo may remain. Selector still picks single best talker.

### Teams stacking

Spidercam AEC reduces echo before virtual mic; Teams AEC handles residual. Use conservative NLP settings in APM.

### Browser uplink preprocessing

Participant browsers may already apply AEC/NS. Further Spidercam AEC may help less — operator disables per stream when uplink sounds over-processed.

---

## Failure modes (TV-in-room)

| Failure | Symptom | Mitigation |
|---------|---------|------------|
| Delay mismatch | Poor convergence | Ref trim; APM delay estimator; GCC-PHAT diagnostic |
| Long reverb | High residual echo | Ducking + penalty; enable AEC; accept Teams tail |
| Nonlinear TV speaker | Filter mis-models | Raw echoPenalty still applies |
| Over-suppression | Thin local speech | Disable AEC on that stream |

---

## Tests

- Synthetic ref + mic with known FIR echo path → ERLE &gt; 15 dB after convergence (manual golden WAV or tolerance band)
- Double-talk segment → near-end energy preserved
- `--mock` passthrough + synthetic `processUs`
- No bitwise sample equality across compilers — statistical thresholds only
