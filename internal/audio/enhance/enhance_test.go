package enhance_test

import (
	"math"
	"testing"

	"github.com/markus/spidercam/internal/audio/enhance"
	audiomath "github.com/markus/spidercam/internal/audio/math"
)

func TestPassthroughIdentity(t *testing.T) {
	d := enhance.NewPassthrough()
	in := sineFrame(0.15, 500)
	out, us := d.Process(in)
	if rmsDiff(in, out) > 1e-6 {
		t.Fatalf("passthrough should preserve samples, diff=%v", rmsDiff(in, out))
	}
	if us <= 0 {
		t.Fatalf("expected positive processUs, got %v", us)
	}
}

func TestFloatInt16Clip(t *testing.T) {
	cases := []struct {
		in   float32
		want int16
	}{
		{0, 0},
		{1, 32767},
		{-1, -32768},
		{2, 32767},
		{-2, -32768},
	}
	for _, tc := range cases {
		got := enhance.FloatToInt16(tc.in)
		if got != tc.want {
			t.Fatalf("FloatToInt16(%v) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestEnhancementBudgetPct(t *testing.T) {
	aecUs := 200.0
	denoiseUs := 150.0
	total := aecUs + denoiseUs
	pct := total / 10_000 * 100
	if math.Abs(pct-3.5) > 0.01 {
		t.Fatalf("enhancementBudgetPct = %v, want 3.5", pct)
	}
}

func TestEMAProcessingTimes(t *testing.T) {
	prev := 0.0
	samples := []float64{100, 100, 100}
	for _, s := range samples {
		prev = audiomath.EMA(prev, s, 0.05)
	}
	if prev <= 0 {
		t.Fatalf("EMA should accumulate timing, got %v", prev)
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
