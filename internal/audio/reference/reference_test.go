package reference

import (
	"math"
	"testing"
	"time"

	audiomath "github.com/markus/spidercam/internal/audio/math"
	"github.com/markus/spidercam/internal/protocol"
)

func TestReferenceVAD(t *testing.T) {
	p := NewProcessor(protocol.DefaultHostConfig)
	m := p.ProcessFrame(silentFrame(-50))
	if m.Vad {
		t.Fatal("silent ref should not be VAD")
	}
	loud := sineFrame(0.3, 440)
	m = p.ProcessFrame(loud)
	if !m.Vad {
		t.Fatal("loud ref should be VAD")
	}
}

func TestEchoPenaltyCorrelated(t *testing.T) {
	p := NewProcessor(protocol.DefaultHostConfig)
	ref := sineFrame(0.2, 440)
	mic := make([]float32, len(ref))
	copy(mic, ref)
	penalty := p.EchoPenalty(mic, ref)
	if penalty < 0.99 {
		t.Fatalf("echoPenalty = %v, want ~1", penalty)
	}
	noise := sineFrame(0.2, 880)
	penalty = p.EchoPenalty(noise, ref)
	if penalty > 0.3 {
		t.Fatalf("uncorrelated echoPenalty = %v, want low", penalty)
	}
}

func TestDelayTrackerSkipsDoubleTalk(t *testing.T) {
	cfg := protocol.DefaultHostConfig
	cfg.LoopDelayMinPeak = 0.1
	cfg.LoopDelayAnalysisMs = 0
	dt := NewDelayTracker(cfg)
	ref := sineFrame(0.2, 440)
	mic := delayedCopy(ref, 48)
	dt.Feed("alice", ref, mic, true, true)
	if est := dt.Estimate("alice", time.Now()); est.Known {
		t.Fatal("double-talk should skip sample")
	}
}

func TestDelayTrackerSyntheticLag(t *testing.T) {
	cfg := protocol.DefaultHostConfig
	cfg.LoopDelayMinPeak = 0.1
	cfg.LoopDelayMinSamples = 1
	cfg.LoopDelayAnalysisMs = 0
	cfg.LoopDelayWindowMs = 500
	cfg.LoopDelayLagSearchMs = 50
	dt := NewDelayTracker(cfg)
	n := audiomath.FrameSamples * 20
	ref := make([]float32, n)
	mic := make([]float32, n)
	const lagSamples = 48
	const freq = 440.0
	for i := 0; i < n; i++ {
		ref[i] = float32(math.Sin(2 * math.Pi * freq * float64(i) / audiomath.SampleRate))
	}
	for i := 0; i < n; i++ {
		j := i - lagSamples
		if j >= 0 {
			mic[i] = ref[j]
		}
	}
	for off := 0; off+audiomath.FrameSamples <= n; off += audiomath.FrameSamples {
		dt.Feed("alice", ref[off:off+audiomath.FrameSamples], mic[off:off+audiomath.FrameSamples], true, false)
	}
	est := dt.Estimate("alice", time.Now())
	if !est.Known {
		t.Fatal("expected known delay estimate")
	}
	want := int(48 * 1000 / audiomath.SampleRate)
	if est.Ms == nil || abs(float64(*est.Ms-want)) > 3 {
		t.Fatalf("delay ms = %v, want ~%d", est.Ms, want)
	}
}

func TestGlobalLatencyIgnoresHostAndRef(t *testing.T) {
	ms := 100
	estimates := map[string]protocol.LoopDelayEstimate{
		protocol.HostStreamID:        {Ms: &ms, Known: true},
		protocol.PlaybackRefStreamID:   {Ms: &ms, Known: true},
		"alice":                      {Ms: &ms, Known: true},
	}
	got := GlobalLatency(estimates)
	if got == nil || *got != 100 {
		t.Fatalf("global latency = %v", got)
	}
}

func sineFrame(amp float64, freq float64) []float32 {
	out := make([]float32, audiomath.FrameSamples)
	for i := range out {
		out[i] = float32(amp * math.Sin(2*math.Pi*freq*float64(i)/audiomath.SampleRate))
	}
	return out
}

func silentFrame(dbfs float64) []float32 {
	amp := audiomath.DbToLinear(dbfs)
	out := make([]float32, audiomath.FrameSamples)
	for i := range out {
		out[i] = float32(amp)
	}
	return out
}

func delayedCopy(src []float32, lag int) []float32 {
	out := make([]float32, len(src))
	for i := range out {
		j := i - lag
		if j >= 0 && j < len(src) {
			out[i] = src[j]
		}
	}
	return out
}
