package audio

import (
	"context"
	"math"
	"sync"
	"time"

	audiomath "github.com/markus/spidercam/internal/audio/math"
	"github.com/markus/spidercam/internal/audio/mixer"
	"github.com/markus/spidercam/internal/audio/processor"
	"github.com/markus/spidercam/internal/audio/reference"
	"github.com/markus/spidercam/internal/protocol"
	"github.com/markus/spidercam/internal/selector"
)

type Engine struct {
	mu                   sync.RWMutex
	cfg                  protocol.HostConfig
	streams              map[string]*processor.Pipeline
	reference            *reference.Processor
	selectorState        *selector.State
	mixer                *mixer.Mixer
	tickCount            uint64
	refMetrics           protocol.ReferenceMetrics
	selection            protocol.SelectionState
	metrics              []protocol.StreamMetrics
	enhancementBudgetPct float64
	lastLoopDelayPub     time.Time
	delayEstimates       map[string]protocol.LoopDelayEstimate
}

func NewEngine(cfg protocol.HostConfig) *Engine {
	return &Engine{
		cfg:           cfg,
		streams:       make(map[string]*processor.Pipeline),
		reference:     reference.NewProcessor(cfg),
		selectorState: selector.NewState(cfg),
		mixer:         mixer.New(),
		delayEstimates: make(map[string]protocol.LoopDelayEstimate),
		selection: protocol.SelectionState{
			ActiveAudioID: cfg.DefaultAudioID,
			ActiveVideoID: cfg.DefaultVideoID,
			MainTalkerID:  cfg.DefaultAudioID,
			MixerState:    protocol.MixerSilence,
		},
	}
}

func (e *Engine) AttachStream(p *processor.Pipeline) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.streams[p.ID()] = p
}

func (e *Engine) AttachReference(proc *reference.Processor) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.reference = proc
}

func (e *Engine) ApplyConfig(cfg protocol.HostConfig) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cfg = cfg
	e.reference.ApplyConfig(cfg)
	e.reference.DelayTracker().ApplyConfig(cfg)
}

func (e *Engine) Run(ctx context.Context, feed CaptureFeed, onMix func(mixer.Frame)) {
	ticker := time.NewTicker(audiomath.FrameMs * time.Millisecond)
	defer ticker.Stop()
	refFrame := make([]float32, audiomath.FrameSamples)
	micBuf := make([]float32, audiomath.FrameSamples)
	monBuf := make([]float32, audiomath.FrameSamples)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if feed != nil {
				e.ingestCapture(feed, micBuf, monBuf)
			}
			e.tick(refFrame, onMix)
		}
	}
}

func (e *Engine) ingestCapture(feed CaptureFeed, micBuf, monBuf []float32) {
	micN := feed.ReadMic(micBuf)
	if host := e.Stream(protocol.HostStreamID); host != nil {
		if micN > 0 {
			host.Jitter().Push(micBuf[:micN])
		} else {
			host.Jitter().PushSilence()
		}
	}
	if n := feed.ReadMonitor(monBuf); n > 0 {
		e.reference.PushFrame(monBuf[:n])
	}
}

func (e *Engine) tick(refFrame []float32, onMix func(mixer.Frame)) {
	e.mu.Lock()

	e.tickCount++
	cfg := e.cfg
	now := time.Now()

	if ref, ok := e.reference.Read(); ok {
		copy(refFrame, ref)
		if cfg.ReferenceDelayMs > 0 {
			applyDelayTrim(refFrame, cfg.ReferenceDelayMs)
		}
		e.refMetrics = e.reference.ProcessFrame(refFrame)
	}

	var totalEnhanceUs float64
	metrics := make([]protocol.StreamMetrics, 0, len(e.streams))
	duckDb := 0.0
	if e.refMetrics.Vad && cfg.ReferenceDuckDb < 0 {
		duckDb = cfg.ReferenceDuckDb
	}

	for id, p := range e.streams {
		frame, ok := p.Jitter().Pull()
		if !ok {
			continue
		}
		echo := e.reference.EchoPenalty(frame, refFrame)
		e.reference.DelayTracker().Feed(id, refFrame, frame, e.refMetrics.Active, p.State().Vad)
		p.SetDuckingGainDb(duckDb)
		m := p.Process(frame, refFrame, echo, cfg)
		totalEnhanceUs += m.AecUs + m.DenoiseUs
		metrics = append(metrics, m)
	}

	e.metrics = metrics
	e.enhancementBudgetPct = totalEnhanceUs / 10_000 * 100

	if e.tickCount%2 == 0 {
		e.selection = selector.Select(e.selectorState, cfg, metrics, now)
	}

	if e.lastLoopDelayPub.IsZero() || now.Sub(e.lastLoopDelayPub) >= time.Duration(cfg.LoopDelayPublishMs)*time.Millisecond {
		e.lastLoopDelayPub = now
		e.delayEstimates = make(map[string]protocol.LoopDelayEstimate)
		for id := range e.streams {
			if id == protocol.PlaybackRefStreamID {
				continue
			}
			e.delayEstimates[id] = e.reference.DelayTracker().Estimate(id, now)
		}
		for i := range metrics {
			if est, ok := e.delayEstimates[metrics[i].ParticipantID]; ok {
				metrics[i].LoopDelay = est
			}
		}
		e.metrics = metrics
	}

	mix := e.mixer.Mix(e.streams, e.selection, e.refMetrics, cfg)
	e.mu.Unlock()
	if onMix != nil {
		onMix(mix)
	}
}

func applyDelayTrim(frame []float32, delayMs int) {
	samples := delayMs * audiomath.SampleRate / 1000
	if samples <= 0 || samples >= len(frame) {
		return
	}
	copy(frame, frame[samples:])
	for i := len(frame) - samples; i < len(frame); i++ {
		frame[i] = 0
	}
}

func (e *Engine) Metrics() []protocol.StreamMetrics {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return append([]protocol.StreamMetrics(nil), e.metrics...)
}

func (e *Engine) Selection() protocol.SelectionState {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.selection
}

func (e *Engine) ReferenceMetrics() protocol.ReferenceMetrics {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.refMetrics
}

func (e *Engine) DelayEstimates() map[string]protocol.LoopDelayEstimate {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make(map[string]protocol.LoopDelayEstimate, len(e.delayEstimates))
	for k, v := range e.delayEstimates {
		out[k] = v
	}
	return out
}

func (e *Engine) EnhancementBudgetPct() float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.enhancementBudgetPct
}

func (e *Engine) ResetLoopDelaySamples() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.reference.ResetLoopDelaySamples()
}

func (e *Engine) SetStreamProcessing(participantID string, flags protocol.StreamProcessingFlags) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	p, ok := e.streams[participantID]
	if !ok {
		return protocol.ErrStreamNotFound
	}
	return p.SetProcessing(flags)
}

func (e *Engine) TickCount() uint64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.tickCount
}

func (e *Engine) Stream(id string) *processor.Pipeline {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.streams[id]
}

func SineFrame(amp float64, freq float64, phase float64) []float32 {
	out := make([]float32, audiomath.FrameSamples)
	for i := range out {
		t := float64(i)/audiomath.SampleRate + phase
		out[i] = float32(amp * math.Sin(2*math.Pi*freq*t))
	}
	return out
}
