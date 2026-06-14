package reference

import (
	"sort"
	"time"

	audiomath "github.com/markus/spidercam/internal/audio/math"
	"github.com/markus/spidercam/internal/protocol"
)

type delaySample struct {
	TauMs   float64
	Quality float64
	At      time.Time
}

type streamDelayState struct {
	samples    []delaySample
	refRing    []float32
	micRing    []float32
	lastAnalysis time.Time
}

type DelayTracker struct {
	cfg     protocol.HostConfig
	streams map[string]*streamDelayState
}

func NewDelayTracker(cfg protocol.HostConfig) *DelayTracker {
	return &DelayTracker{
		cfg:     cfg,
		streams: make(map[string]*streamDelayState),
	}
}

func (t *DelayTracker) ApplyConfig(cfg protocol.HostConfig) {
	t.cfg = cfg
}

func (t *DelayTracker) Reset() {
	t.streams = make(map[string]*streamDelayState)
}

func (t *DelayTracker) windowSamples() int {
	return t.cfg.LoopDelayWindowMs * audiomath.SampleRate / 1000
}

func (t *DelayTracker) stateFor(id string) *streamDelayState {
	st, ok := t.streams[id]
	if !ok {
		capacity := t.windowSamples()
		if capacity < audiomath.FrameSamples {
			capacity = audiomath.FrameSamples
		}
		st = &streamDelayState{
			refRing: make([]float32, 0, capacity),
			micRing: make([]float32, 0, capacity),
		}
		t.streams[id] = st
	}
	return st
}

func (t *DelayTracker) Feed(participantID string, refBuf, micBuf []float32, refActive bool, localVad bool) {
	if !refActive || localVad {
		return
	}
	st := t.stateFor(participantID)
	st.refRing = appendRing(st.refRing, refBuf, t.windowSamples())
	st.micRing = appendRing(st.micRing, micBuf, t.windowSamples())

	now := time.Now()
	interval := time.Duration(t.cfg.LoopDelayAnalysisMs) * time.Millisecond
	if interval > 0 && !st.lastAnalysis.IsZero() && now.Sub(st.lastAnalysis) < interval {
		return
	}
	st.lastAnalysis = now

	if len(st.refRing) < audiomath.FrameSamples*2 || len(st.micRing) < audiomath.FrameSamples*2 {
		return
	}
	lagMs, quality := audiomath.GccPhatPeakLag(st.refRing, st.micRing, t.cfg.LoopDelayLagSearchMs)
	if quality < t.cfg.LoopDelayMinPeak {
		return
	}
	st.samples = append(st.samples, delaySample{TauMs: lagMs, Quality: quality, At: now})
	st.samples = pruneStale(st.samples, now, time.Duration(t.cfg.LoopDelayStaleMs)*time.Millisecond)
}

func appendRing(buf, frame []float32, maxLen int) []float32 {
	buf = append(buf, frame...)
	if len(buf) > maxLen {
		buf = buf[len(buf)-maxLen:]
	}
	return buf
}

func pruneStale(samples []delaySample, now time.Time, stale time.Duration) []delaySample {
	cutoff := now.Add(-stale)
	out := samples[:0]
	for _, s := range samples {
		if s.At.After(cutoff) {
			out = append(out, s)
		}
	}
	return out
}

func (t *DelayTracker) Estimate(participantID string, now time.Time) protocol.LoopDelayEstimate {
	st, ok := t.streams[participantID]
	if !ok {
		return protocol.LoopDelayEstimate{Known: false}
	}
	window := time.Duration(t.cfg.LoopDelayWindowMs) * time.Millisecond
	recent := filterRecent(st.samples, now, window)
	if len(recent) < t.cfg.LoopDelayMinSamples {
		return protocol.LoopDelayEstimate{Known: false}
	}
	med := medianTau(recent)
	spread := madTau(recent, med)
	ms := int(med + 0.5)
	return protocol.LoopDelayEstimate{
		Ms:            &ms,
		UncertaintyMs: spread,
		Known:         true,
	}
}

func filterRecent(samples []delaySample, now time.Time, window time.Duration) []delaySample {
	cutoff := now.Add(-window)
	var out []delaySample
	for _, s := range samples {
		if s.At.After(cutoff) {
			out = append(out, s)
		}
	}
	return out
}

func medianTau(samples []delaySample) float64 {
	vals := make([]float64, len(samples))
	for i, s := range samples {
		vals[i] = s.TauMs
	}
	sort.Float64s(vals)
	n := len(vals)
	if n%2 == 1 {
		return vals[n/2]
	}
	return (vals[n/2-1] + vals[n/2]) / 2
}

func madTau(samples []delaySample, med float64) float64 {
	devs := make([]float64, len(samples))
	for i, s := range samples {
		devs[i] = abs(s.TauMs - med)
	}
	sort.Float64s(devs)
	n := len(devs)
	if n%2 == 1 {
		return devs[n/2]
	}
	return (devs[n/2-1] + devs[n/2]) / 2
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func GlobalLatency(estimates map[string]protocol.LoopDelayEstimate) *int {
	var max *int
	for id, e := range estimates {
		if id == protocol.PlaybackRefStreamID || id == protocol.HostStreamID {
			continue
		}
		if !e.Known || e.Ms == nil {
			continue
		}
		if max == nil || *e.Ms > *max {
			v := *e.Ms
			max = &v
		}
	}
	return max
}
