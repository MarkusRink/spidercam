# Frame engine

**Target:** `internal/audio/engine/`

Pull-based clock @ 10 ms. Fan-in: host mic, playback reference, Pion participant streams.

## Engine

```go
package engine

type Engine struct {
	cfg       protocol.HostConfig
	streams   map[string]*processor.Pipeline
	reference *reference.Processor
	selector  *selector.State
	mixer     *mixer.Mixer
	tickCount uint64
}

func NewEngine(cfg protocol.HostConfig) *Engine

func (e *Engine) AttachHostMic(id string, src capture.Mic)
func (e *Engine) AttachReference(proc *reference.Processor)
func (e *Engine) AttachParticipant(id, name string, inbound *jitter.Buffer)

func (e *Engine) Run(ctx context.Context, onMix func(mixer.Frame))

func (e *Engine) Metrics() []protocol.StreamMetrics
func (e *Engine) Selection() *protocol.SelectionState
func (e *Engine) DelayEstimates() map[string]protocol.LoopDelayEstimate
func (e *Engine) EnhancementBudgetPct() float64
func (e *Engine) ApplyConfig(partial protocol.HostConfig)
func (e *Engine) SetStreamProcessing(participantID string, flags protocol.StreamProcessingFlags) error
```

## Main loop

```go
func (e *Engine) Run(ctx context.Context, onMix func(mixer.Frame)) {
	ticker := time.NewTicker(FrameMs * time.Millisecond)
	refFrame := make([]float32, FrameSamples)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.tickCount++

			if ref, ok := e.reference.Read(); ok {
				copy(refFrame, ref)
				if e.cfg.ReferenceDelayMs > 0 {
					applyDelayTrim(refFrame, e.cfg.ReferenceDelayMs)
				}
				e.refMetrics = e.reference.ProcessFrame(refFrame)
			}

			var totalEnhanceUs float64
			for id, p := range e.streams {
				frame, ok := p.Jitter().Pull()
				if !ok {
					continue
				}
				echo := e.reference.EchoPenalty(frame, refFrame)
				e.reference.DelayTracker().Feed(id, refFrame, frame, e.refMetrics.Active, p.State().Vad)
				m := p.Process(frame, refFrame, echo, e.cfg)
				totalEnhanceUs += m.AecUs + m.DenoiseUs
			}

			e.enhancementBudgetPct = totalEnhanceUs / 10_000 * 100

			if e.tickCount%2 == 0 {
				e.selection = selector.Select(e.selectorState, e.cfg, e.allMetrics(), time.Now())
			}

			mix := e.mixer.Mix(e.streams, e.selection, refFrame, e.refMetrics)
			onMix(mix)
		}
	}
}
```

## Jitter buffer

```go
// internal/audio/jitter/buffer.go
type Buffer struct {
	targetFrames int // ~5 frames = 50ms
}

func (b *Buffer) Push(pcm []float32)
func (b *Buffer) Pull() ([]float32, bool)
```

Push from Pion decode goroutine; pull from engine ticker.

## Playback-ref exclusion

`playback-ref` metrics appear in `RoomState.reference` but **never** enter selector candidate list.

## Capture device change

`capture.Reopen` → `engine.ResetLoopDelaySamples()` and reset all AEC states (new acoustic/ref path).
