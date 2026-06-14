package processor

import (
	"math"
	"testing"

	audiomath "github.com/markus/spidercam/internal/audio/math"
	"github.com/markus/spidercam/internal/protocol"
)

func TestNoiseFloorOnlyUpdatesWhenNotVAD(t *testing.T) {
	p := NewPipeline("alice", "Alice", protocol.StreamRoleParticipant)
	cfg := protocol.DefaultHostConfig
	cfg.VadSnrOnDb = 15
	cfg.VadSnrOffDb = 10
	cfg.VadHangoverMs = 0

	loud := sine(0.3, 440)
	for i := 0; i < 30; i++ {
		p.Process(loud, nil, 0, cfg)
	}
	floorDuringSpeech := p.state.NoiseFloorDbfs

	silent := make([]float32, audiomath.FrameSamples)
	for i := 0; i < 50; i++ {
		p.Process(silent, nil, 0, cfg)
	}
	if p.state.Vad {
		t.Fatal("expected VAD off during silence")
	}
	if p.state.NoiseFloorDbfs == floorDuringSpeech {
		t.Fatalf("noise floor should update when !vad: during=%v after=%v", floorDuringSpeech, p.state.NoiseFloorDbfs)
	}
}

func TestHighEchoPenaltyLowersScore(t *testing.T) {
	p := NewPipeline("alice", "Alice", protocol.StreamRoleParticipant)
	cfg := protocol.DefaultHostConfig
	frame := sine(0.2, 440)
	mLow := p.Process(frame, nil, 0, cfg)
	p2 := NewPipeline("bob", "Bob", protocol.StreamRoleParticipant)
	mHigh := p2.Process(frame, nil, 0.9, cfg)
	if mHigh.Score >= mLow.Score {
		t.Fatalf("high echo penalty should lower score: %v vs %v", mHigh.Score, mLow.Score)
	}
}

func TestEnhancementToggleDoesNotChangeScore(t *testing.T) {
	p := NewPipeline("alice", "Alice", protocol.StreamRoleParticipant)
	cfg := protocol.DefaultHostConfig
	frame := sine(0.2, 440)
	m1 := p.Process(frame, nil, 0, cfg)
	if err := p.SetProcessing(protocol.StreamProcessingFlags{DenoiseEnabled: true, AecEnabled: true}); err != nil {
		t.Fatal(err)
	}
	m2 := p.Process(frame, frame, 0, cfg)
	if m1.Score != m2.Score {
		t.Fatalf("score should use raw branch: %v vs %v", m1.Score, m2.Score)
	}
}

func TestEnhancementToggleUpdatesTiming(t *testing.T) {
	p := NewPipeline("alice", "Alice", protocol.StreamRoleParticipant)
	cfg := protocol.DefaultHostConfig
	frame := sine(0.2, 440)
	m1 := p.Process(frame, nil, 0, cfg)
	if m1.AecUs != 0 || m1.DenoiseUs != 0 {
		t.Fatalf("expected zero timing when disabled: aec=%v denoise=%v", m1.AecUs, m1.DenoiseUs)
	}
	if err := p.SetProcessing(protocol.StreamProcessingFlags{DenoiseEnabled: true, AecEnabled: true}); err != nil {
		t.Fatal(err)
	}
	m2 := p.Process(frame, frame, 0, cfg)
	if m2.AecUs <= 0 || m2.DenoiseUs <= 0 {
		t.Fatalf("expected positive timing when enabled: aec=%v denoise=%v", m2.AecUs, m2.DenoiseUs)
	}
}

func sine(amp float64, freq float64) []float32 {
	out := make([]float32, audiomath.FrameSamples)
	for i := range out {
		out[i] = float32(amp * math.Sin(2*math.Pi*freq*float64(i)/audiomath.SampleRate))
	}
	return out
}
