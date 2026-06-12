# Voice isolation (problem class 2)

**Target:** `internal/audio/enhance/`

Noise suppression and speech cleanup for **in-room talkers** — HVAC, keyboard, room noise, modest reverb. Does **not** distinguish TV speech from local speech; see [echo-cancellation.md](./echo-cancellation.md) for echo.

---

## RNNoise

| Property | Value |
|----------|-------|
| Library | [RNNoise](https://github.com/xiph/rnnoise) (BSD-3-Clause) |
| Rate / frame | 48 kHz, 480 samples (10 ms) |
| Reference input | **None** |
| State | One `DenoiseState` per enabled stream |
| Default | **Off** |
| Control | Host toggle per stream on stream card |

### Why RNNoise

| vs | RNNoise | WebRTC NS (in APM) | DeepFilterNet |
|----|---------|-------------------|---------------|
| CPU | Moderate, desktop-friendly | Very low (bundled with AEC) | Higher |
| Quality | Good general enhancement | Stationary noise mainly | Stronger; LGPL |
| Coupling | Independent of AEC toggle | Tied to APM build | Separate stack |

DeepFilterNet / WPE dereverb out of scope — higher latency and build complexity.

---

## Pipeline placement

On the **enhancement branch** after optional AEC ([overview.md](./overview.md)):

```text
raw ──► analysis tap
raw ──► AEC? ──► RNNoise? ──► gate/duck ──► lastFrame ──► mixer
```

| Metric | Source |
|--------|--------|
| VAD, SNR, score, echoPenalty, loop delay | **Raw** tap |
| Card RMS / peak meters | **Post-enhancement** |
| Mixer `lastFrame` | **Post-enhancement** |

---

## Go integration

```go
package enhance

type Denoiser interface {
	Process(frame []float32) (out []float32, processUs float64)
	Reset()
}

func NewRNNoise() (Denoiser, error)   // //go:build linux && cgo
func NewPassthrough() Denoiser        // //go:build mock || !cgo
```

```text
native/rnnoise/
  rnnoise_shim.c
internal/audio/enhance/
  rnnoise.go
  passthrough.go
```

C API: `rnnoise_create`, `rnnoise_process_frame`, `rnnoise_destroy`. Convert float32 ↔ int16 with clip.

Build with AVX2 RTCD when available (`--enable-x86-rtcd`).

### Lifecycle

| Event | Action |
|-------|--------|
| Stream joins | `denoiseEnabled = false` |
| Toggle on | `rnnoise_create()` |
| Toggle off | passthrough; `rnnoise_destroy()` |
| Stream leaves | destroy state |

Toggling mid-utterance causes timbre step — no NS crossfade.

---

## Performance telemetry

Wall time around cgo each frame; EMA (α ≈ 0.05) for UI.

```go
// On StreamMetrics
DenoiseEnabled bool    `json:"denoiseEnabled"`
DenoiseUs      float64 `json:"denoiseUs"` // 0 when disabled

AecEnabled     bool    `json:"aecEnabled"`
AecUs          float64 `json:"aecUs"`     // 0 when disabled
```

```go
// On RoomState
EnhancementBudgetPct float64 `json:"enhancementBudgetPct"` // (Σ aecUs + Σ denoiseUs) / 10_000 × 100
```

**Card UI** when enabled:

```text
AEC · 0.42ms
NS  · 0.18ms
```

**Header** (when any processing on): `enhancementBudgetPct` with tiers — green &lt;5%, amber 5–15%, red &gt;15% of 10 ms tick.

---

## CPU envelope (indicative, desktop x86)

| Component | Per stream / 10 ms |
|-----------|---------------------|
| RNNoise (AVX2) | ~0.05–0.3 ms |
| RNNoise (scalar) | ~0.3–1.0 ms |
| APM AEC3 | ~0.1–1.0 ms |

Four streams with both enabled ≈ 1–8 ms of 10 ms budget — profile on target host.

---

## Limitations

- No echo cancellation — use AEC for TV bleed
- May thin speech at very low SNR
- Stacks with browser NS on participant uplink
- ~10–20 ms RNNoise effective latency (overlap-add)

---

## Tests

| Layer | Cases |
|-------|-------|
| Unit | float↔int16 clip; passthrough when disabled |
| Unit | EMA `denoiseUs`, `aecUs`, `enhancementBudgetPct` |
| E2E | `set-stream-processing` → `host-state` flags |
| Mock | `--mock` synthetic timing without native libs |
