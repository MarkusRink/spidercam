# Mixer

**Target:** `internal/audio/mixer/`

```go
package mixer

type Frame struct {
	PCM      []float32
	Video    VideoFrame // RGBA 1280x720
	OutDbfs  float64
	OutPeakDbfs float64
}

type Mixer struct {
	limiter *SoftLimiter
	compositor *VideoCompositor
}

func (m *Mixer) Mix(
	streams map[string]*processor.Pipeline,
	sel *protocol.SelectionState,
	refMetrics protocol.ReferenceMetrics,
) Frame
```

## Audio

- Equal-power crossfade between `crossfade.fromId` and `crossfade.toId` using `math.EqualPowerGains(t)` — **audio only**
- Apply per-stream `lastFrame` with gate/duck gains
- Reference ducking when `refMetrics.Active && cfg.ReferenceDuckDb < 0`
- Master soft limiter → `OutDbfs` / `OutPeakDbfs` for preview OUT meter

## Video

- Composite **active video track only** — hard cut when `activeVideoId` changes
- No RGBA blend during audio crossfade; video switches immediately (respect `videoHoldMs` in selector before `activeVideoId` changes)
- Output resolution from config (1280×720 default)

## No bridge

Mixed output goes to `internal/output` directly — no WebSocket PCM path.
