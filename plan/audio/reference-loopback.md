# Playback reference & room loopback correction

**Target:** `internal/audio/reference/`

## Problem

```text
Remote Teams participant speaks
    → Teams on host laptop plays on speakers / HDMI to TV
    → Sound in room
    → Participant laptop mics pick it up
    → Selector sees VAD + SNR on Alice/Bob
    → Wrong routing or echo back to Teams
```

Teams AEC on the virtual mic **cannot** cancel bleed that enters **before** the mix on separate devices.

## Solution

Capture **what the host machine is playing** — the default speaker output (Teams meeting audio). The host is in the meeting, so this equals “what the room hears from the conference.”

```text
Teams → audio sink → speakers/TV
           ↓ monitor tap (playback-ref)
     reference processor
           ↓
     echoPenalty + ref duck on mic streams
           ↓
     passive loop delay (GCC-PHAT) → latency UI
```

## Stream identity

```go
const PlaybackRefStreamID = "playback-ref"
const HostStreamID = "host"
```

`playback-ref` is analysis-only — never routed to Teams. Host UI shows REF level in the **preview panel** vertical meter (not on the stream grid).

## Analysis tap only

`echoPenalty` and GCC-PHAT loop delay run on the **raw calibrated** mic branch — never on AEC- or RNNoise-processed audio. Enhancement must not hide echo energy from the selector.

The same `playback-ref` frame also feeds each stream’s AEC `ProcessReverse` on the enhancement branch ([echo-cancellation.md](./echo-cancellation.md)).

## Per-frame processing (10 ms)

```go
package reference

type Processor struct {
	cfg HostConfig
	vad bool
	rmsDbfs float64
	holdUntil time.Time
	delay *DelayTracker
}

func (p *Processor) ProcessFrame(samples []float32) ReferenceMetrics {
	p.rmsDbfs = math.RmsDbfs(samples)
	p.vad = p.updateVAD(p.rmsDbfs) // separate thresholds: RefVadOnDbfs, RefVadOffDbfs
	return ReferenceMetrics{
		StreamID: PlaybackRefStreamID,
		RmsDbfs:  p.rmsDbfs,
		PeakDbfs: p.peakDbfs,
		Vad:      p.vad,
		Active:   p.vad,
	}
}

func (p *Processor) EchoPenalty(micSamples, refSamples []float32) float64 {
	// Normalized cross-correlation at lag 0 (same 10ms frame)
	// Returns 0..1 — 1 = mic frame is strongly correlated with reference
	corr := normalizedCorrelation(micSamples, refSamples)
	return clamp(corr, 0, 1)
}
```

## Score integration

In `stream-processor` — `micFrame` is **raw** (pre-AEC/NS):

```go
echoPenalty := refProcessor.EchoPenalty(micFrame, refFrame)
components := math.BuildScoreComponents(ScoreInput{
	RmsDbfs: rmsDbfs,
	SnrDb:   snrDb,
	Vad:     vad,
	Role:    role,
	EchoPenalty: echoPenalty,
	HostPriority: cfg.HostPriority,
})
// frameScore subtracts weights.echoPenalty * echoPenalty
```

When `echoPenalty` is high, laptop mic bleed from TV scores **below** true local speech.

## Reference ducking (host-configurable)

When reference VAD active and `cfg.ReferenceDuckDb < 0`:

```go
if ref.Vad && cfg.ReferenceDuckDb < 0 {
	for _, stream := range participantStreams {
		stream.DuckingGainDb = cfg.ReferenceDuckDb // 0 = off; default -12 dB
	}
}
```

Host sets **Ducking** slider (0 … −12 dB) in the settings panel ([domain/host-config.md](../domain/host-config.md)). `0` disables ducking. Works together with `echoPenalty` — penalty affects routing scores, ducking attenuates mix level.

## Passive loop delay estimation

No test tones. Remote Teams speech on `playback-ref` is the probe; participant mics that hear the room are the return path.

### Measured quantity

Cross-correlate ref vs mic over a lag window → peak at **τ ms**:

```text
τ ≈ Teams→ref + speaker→air→mic + mic→daemon (uplink + jitter)
```

Many **phases** occur naturally whenever someone remote speaks and room mics pick up speakers. Track drift over the meeting; do not inject chirp/MLS.

### Phase gates

Run analysis only when all pass:

| Gate | Reason |
|------|--------|
| `reference.active` | Probe signal present |
| Peak correlation ≥ `LoopDelayMinPeak` | Mic hears room, not noise |
| Local mic not dominant (double-talk guard) | Correlation meaningful |
| Optional: `echoPenalty > 0.3` | Cheap pre-filter |

### Estimator

```go
package reference

type DelayTracker struct {
	cfg HostConfig
	samples map[string][]delaySample // per participant + host stream id
}

type delaySample struct {
	TauMs   float64
	Quality float64
	At      time.Time
}

type LoopDelayEstimate struct {
	Ms            *int
	UncertaintyMs float64
	Known         bool
}

func (t *DelayTracker) Feed(participantID string, refBuf, micBuf []float32, refActive bool, localVad bool) {
	if !refActive || localVad {
		return
	}
	peak, quality := math.GccPhatPeakLag(refBuf, micBuf, t.cfg.LoopDelayLagSearchMs)
	if quality < t.cfg.LoopDelayMinPeak {
		return
	}
	t.appendSample(participantID, delaySample{TauMs: peak, Quality: quality, At: time.Now()})
}

func (t *DelayTracker) Estimate(participantID string, now time.Time) LoopDelayEstimate {
	s := t.recentSamples(participantID, now, t.cfg.LoopDelayStaleMs)
	if len(s) < t.cfg.LoopDelayMinSamples {
		return LoopDelayEstimate{Known: false}
	}
	med := medianTau(s)
	spread := madTau(s, med) // or IQR/2
	ms := int(math.Round(med))
	return LoopDelayEstimate{Ms: &ms, UncertaintyMs: spread, Known: true}
}
```

- **Analysis window:** `LoopDelayWindowMs` (default 500 ms) of ref+mic PCM, refreshed every `LoopDelayAnalysisMs` (default 250 ms) per eligible stream — not on every 10 ms tick.
- **Publish:** recompute `Estimate()` every `LoopDelayPublishMs` (default 3 s); UI treats latency as stable (“outdated but accurate”).
- **Stale TTL:** if no good sample in `LoopDelayStaleMs` (default 5 min), set `Known: false`.

### Global latency

```go
func GlobalLatency(estimates map[string]LoopDelayEstimate) *int {
	var max *int
	for id, e := range estimates {
		if id == PlaybackRefStreamID {
			continue
		}
		if !e.Known || e.Ms == nil {
			continue
		}
		if max == nil || *e.Ms > *max {
			max = e.Ms
		}
	}
	return max // nil → host header "—"
}
```

Participant streams only — host mic loop is diagnostic on host strip, not used for global max.

### Separation from echoPenalty

| Output | Lag | Rate | Use |
|--------|-----|------|-----|
| `echoPenalty` | 0 (same frame) | 10 ms | Scoring, ducking |
| `loopDelay` | GCC-PHAT peak τ | ~3 s publish | Latency bar, global max |

## HostConfig additions

```go
ReferenceVadOnDbfs  float64 // default -35
ReferenceVadOffDbfs float64 // default -45
ReferenceDuckDb     float64 // default -12; 0 = off
ReferenceDelayMs     int     // default 0 — fixed trim on ref channel for scoring

LoopDelayScaleMaxMs   int     // default 100 — UI bar full scale
LoopDelayWindowMs     int     // default 500
LoopDelayLagSearchMs  int     // default 300
LoopDelayAnalysisMs   int     // default 250
LoopDelayPublishMs    int     // default 3000
LoopDelayMinSamples   int     // default 3
LoopDelayMinPeak      float64 // default 0.25
LoopDelayStaleMs      int     // default 300000 (5 min)
```

## UI

Preview panel: **REF** vertical meter beside OUT ([ui/design-system.md](../ui/design-system.md)). No REF card on stream grid.

Stream cards: **LoopDelayText** (`~118 ms` or `—`). Header: `globalLatencyMs` only.

Participant session card: **LoopDelayText** from `selfMetric.loopDelay`.

Bleed mitigation visible via **score border** brightness and routing behavior — no duck/echo pills on stream cards.

## Tests

- Synthetic correlated sine @ lag τ → GCC-PHAT peak near τ; uncertainty low
- Uncorrelated noise → no sample appended
- Double-talk (local VAD) → phase skipped
- Global max ignores host/ref; returns nil when no participant known
- Stale TTL clears `Known`
- echoPenalty @ lag 0 unchanged from prior cases
