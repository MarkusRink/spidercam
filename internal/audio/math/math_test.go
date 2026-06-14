package audiomath

import (
	"math"
	"testing"

	"github.com/markus/spidercam/internal/protocol"
)

func TestRmsDbfs(t *testing.T) {
	samples := make([]float32, FrameSamples)
	for i := range samples {
		samples[i] = 0.1
	}
	got := RmsDbfs(samples)
	want := LinearToDb(0.1)
	if math.Abs(got-want) > 0.01 {
		t.Fatalf("RmsDbfs = %v, want %v", got, want)
	}
}

func TestNormLevel(t *testing.T) {
	if NormLevel(-60) != 0 {
		t.Fatalf("NormLevel(-60) should be 0")
	}
	if NormLevel(-20) != 1 {
		t.Fatalf("NormLevel(-20) should be 1")
	}
	if NormLevel(-40) != 0.5 {
		t.Fatalf("NormLevel(-40) should be 0.5, got %v", NormLevel(-40))
	}
}

func TestFrameScore(t *testing.T) {
	comp := protocol.ScoreComponents{Level: 1, Snr: 1, Vad: 1, Priority: 1, EchoPenalty: 0}
	w := protocol.DefaultScoreWeights
	got := FrameScore(comp, w)
	if got != 1 {
		t.Fatalf("FrameScore should clamp to 1, got %v", got)
	}
	comp.EchoPenalty = 1
	got = FrameScore(comp, w)
	want := w.Level + w.Snr + w.Vad + w.Priority - w.EchoPenalty
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("FrameScore with echo = %v, want %v", got, want)
	}
}

func TestEqualPowerGains(t *testing.T) {
	a, b := EqualPowerGains(0)
	if math.Abs(a-1) > 1e-9 || math.Abs(b) > 1e-9 {
		t.Fatalf("t=0: got %v, %v", a, b)
	}
	a, b = EqualPowerGains(1)
	if math.Abs(a) > 1e-9 || math.Abs(b-1) > 1e-9 {
		t.Fatalf("t=1: got %v, %v", a, b)
	}
	a, b = EqualPowerGains(0.5)
	sum := a*a + b*b
	if math.Abs(sum-1) > 0.01 {
		t.Fatalf("equal power sum = %v, want ~1", sum)
	}
}

func TestNormalizedCorrelation(t *testing.T) {
	a := []float32{1, 0, -1, 0}
	b := []float32{1, 0, -1, 0}
	if NormalizedCorrelation(a, b) != 1 {
		t.Fatalf("identical signals should correlate to 1")
	}
	c := []float32{0, 1, 0, -1}
	if NormalizedCorrelation(a, c) > 0.5 {
		t.Fatalf("orthogonal-ish signals should have low correlation")
	}
}

func TestGccPhatPeakLag(t *testing.T) {
	const freq = 440.0
	const lagSamples = 48
	n := FrameSamples * 10
	ref := make([]float32, n)
	mic := make([]float32, n)
	for i := 0; i < n; i++ {
		ref[i] = float32(math.Sin(2 * math.Pi * freq * float64(i) / SampleRate))
	}
	for i := 0; i < n; i++ {
		src := i - lagSamples
		if src >= 0 && src < n {
			mic[i] = ref[src]
		}
	}
	lagMs, quality := GccPhatPeakLag(ref, mic, 50)
	wantMs := float64(lagSamples) * 1000 / SampleRate
	if math.Abs(lagMs-wantMs) > 2 {
		t.Fatalf("lagMs = %v, want ~%v (quality %v)", lagMs, wantMs, quality)
	}
	if quality < 0.5 {
		t.Fatalf("quality = %v, want high correlation", quality)
	}
}

func TestBuildScoreComponentsHostPriority(t *testing.T) {
	comp := BuildScoreComponents(ScoreInput{
		RmsDbfs: -30, SnrDb: 10, Vad: true,
		Role: protocol.StreamRoleHost, HostPriority: 1.0,
	})
	if comp.Priority != 1 {
		t.Fatalf("host priority = %v", comp.Priority)
	}
	comp = BuildScoreComponents(ScoreInput{
		RmsDbfs: -30, SnrDb: 10, Vad: true,
		Role: protocol.StreamRoleParticipant, HostPriority: 1.0,
	})
	if comp.Priority != 0 {
		t.Fatalf("participant priority = %v", comp.Priority)
	}
}
