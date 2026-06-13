# Stream processor

**Target:** `internal/audio/processor/`

Per-stream pipeline after jitter pull. Implements the **dual branch**: analysis on raw calibrated audio; enhancement for mixer output.

## State

```go
type State struct {
	NoiseFloorDbfs   float64
	CalibrationGain  float64
	CalibrationPhase string
	Vad              bool
	VadHangoverUntil time.Time
	ScoreSmooth      float64
	GateOpen         bool
	DuckingGainDb    float64

	AecEnabled     bool
	DenoiseEnabled bool
	AecUs          float64 // EMA
	DenoiseUs      float64 // EMA
}

type Pipeline struct {
	state    State
	aec      aec.Processor
	denoiser enhance.Denoiser
	lastFrame []float32
	role     protocol.StreamRole
}
```

## Process (per 10 ms frame)

```go
func (p *Pipeline) Process(
	raw []float32,
	ref []float32,
	echoPenalty float64,
	cfg protocol.HostConfig,
) protocol.StreamMetrics {
	raw = hpf(raw)
	raw = applyGain(raw, p.state.CalibrationGain)

	// --- analysis tap (raw calibrated) ---
	rmsRaw := math.RmsDbfs(raw)
	p.updateNoiseFloor(rmsRaw, p.state.Vad)
	snr := math.SnrDb(rmsLinear, noiseLinear)
	p.state.Vad = updateVAD(snr, p.state, cfg)

	components := math.BuildScoreComponents(math.ScoreInput{
		RmsDbfs: rmsRaw, SnrDb: snr, Vad: p.state.Vad,
		Role: p.role, EchoPenalty: echoPenalty,
		HostPriority: cfg.HostPriority,
	})
	score := math.FrameScore(components, cfg.ScoreWeights)
	p.state.ScoreSmooth = math.EMA(p.state.ScoreSmooth, score, cfg.ScoreSmoothingAlpha)

	// --- enhancement branch ---
	enhanced := raw
	if p.state.AecEnabled && p.aec != nil {
		p.aec.ProcessReverse(ref)
		enhanced = p.aec.ProcessNear(enhanced)
		st := p.aec.Stats()
		p.state.AecUs = math.EMA(p.state.AecUs, st.ProcessUs, 0.05)
	}
	if p.state.DenoiseEnabled && p.denoiser != nil {
		var us float64
		enhanced, us = p.denoiser.Process(enhanced)
		p.state.DenoiseUs = math.EMA(p.state.DenoiseUs, us, 0.05)
	}

	outRms := math.RmsDbfs(enhanced)
	outPeak := math.PeakDbfs(enhanced)

	gateDb := gate(p.state, rmsRaw, cfg) // gate from raw energy
	p.lastFrame = applyGainDb(enhanced, gateDb+p.state.DuckingGainDb)

	return p.buildMetrics(components, score, outRms, outPeak, gateDb)
}
```

## Stream processing flags

```go
func (p *Pipeline) SetProcessing(flags protocol.StreamProcessingFlags) error {
	if flags.AecEnabled != p.state.AecEnabled {
		p.state.AecEnabled = flags.AecEnabled
		// create or destroy p.aec
	}
	if flags.DenoiseEnabled != p.state.DenoiseEnabled {
		p.state.DenoiseEnabled = flags.DenoiseEnabled
		// create or destroy p.denoiser
	}
	return nil
}
```

## Metric sources

| Field                                                        | Branch                                                |
| ------------------------------------------------------------ | ----------------------------------------------------- |
| `score`, `vad`, `snrDb`, `noiseFloorDbfs`, `scoreComponents` | Raw                                                   |
| `rmsDbfs`, `peakDbfs` (card meter)                           | Post-enhancement                                      |
| `echoPenalty`                                                | Computed on raw (in engine, from reference processor) |
| `aecEnabled`, `denoiseEnabled`, `aecUs`, `denoiseUs`         | Enhancement state                                     |

## Echo penalty

`echoPenalty` from [reference-loopback.md](./reference-loopback.md) — correlation between **raw** mic and `playback-ref` in the same 10 ms frame.

High penalty suppresses TV bleed in routing without muting true local speech (uncorrelated).

## Tests

`processor_test.go`: noise floor only updates when !vad; VAD hangover; calibration clamp; high echoPenalty lowers rank; scores unchanged when only enhancement toggled; meters reflect post-NS frame when denoise on.
