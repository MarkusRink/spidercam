package aec_test

import (
	"math"
	"testing"

	"github.com/markus/spidercam/internal/audio/aec"
	audiomath "github.com/markus/spidercam/internal/audio/math"
)

func TestPassthroughProcessNearIdentity(t *testing.T) {
	p, err := aec.New(aec.Config{Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	in := sineFrame(0.2, 440)
	out := p.ProcessNear(in)
	if rmsDiff(in, out) > 1e-6 {
		t.Fatalf("passthrough should preserve samples, diff=%v", rmsDiff(in, out))
	}
}

func TestPassthroughProcessReverseNoOp(t *testing.T) {
	p, err := aec.New(aec.Config{Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	ref := sineFrame(0.1, 220)
	p.ProcessReverse(ref)
	in := sineFrame(0.2, 440)
	out := p.ProcessNear(in)
	if rmsDiff(in, out) > 1e-6 {
		t.Fatal("ProcessReverse should not alter near path in passthrough")
	}
}

func TestPassthroughStatsTiming(t *testing.T) {
	p, err := aec.New(aec.Config{Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	_ = p.ProcessNear(sineFrame(0.1, 300))
	st := p.Stats()
	if st.ProcessUs <= 0 {
		t.Fatalf("expected positive processUs, got %v", st.ProcessUs)
	}
}

func TestPassthroughReset(t *testing.T) {
	p, err := aec.New(aec.Config{Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	_ = p.ProcessNear(sineFrame(0.1, 300))
	p.Reset()
	st := p.Stats()
	if st.ProcessUs != 0 {
		t.Fatalf("Reset should clear timing state, got processUs=%v", st.ProcessUs)
	}
}

func sineFrame(amp float64, freq float64) []float32 {
	out := make([]float32, audiomath.FrameSamples)
	for i := range out {
		out[i] = float32(amp * math.Sin(2*math.Pi*freq*float64(i)/audiomath.SampleRate))
	}
	return out
}

func rmsDiff(a, b []float32) float64 {
	var sum float64
	for i := range a {
		d := float64(a[i] - b[i])
		sum += d * d
	}
	return math.Sqrt(sum / float64(len(a)))
}
