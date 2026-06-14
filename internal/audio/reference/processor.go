package reference

import (
	"time"

	audiomath "github.com/markus/spidercam/internal/audio/math"
	"github.com/markus/spidercam/internal/protocol"
)

type Processor struct {
	cfg       protocol.HostConfig
	vad       bool
	rmsDbfs   float64
	peakDbfs  float64
	holdUntil time.Time
	delay     *DelayTracker
	frameBuf  []float32
	hasFrame  bool
}

func NewProcessor(cfg protocol.HostConfig) *Processor {
	return &Processor{
		cfg:      cfg,
		delay:    NewDelayTracker(cfg),
		frameBuf: make([]float32, audiomath.FrameSamples),
	}
}

func (p *Processor) ApplyConfig(cfg protocol.HostConfig) {
	p.cfg = cfg
	p.delay.ApplyConfig(cfg)
}

func (p *Processor) Read() ([]float32, bool) {
	if !p.hasFrame {
		return nil, false
	}
	out := make([]float32, len(p.frameBuf))
	copy(out, p.frameBuf)
	return out, true
}

func (p *Processor) PushFrame(samples []float32) {
	if len(samples) < len(p.frameBuf) {
		return
	}
	copy(p.frameBuf, samples[:len(p.frameBuf)])
	p.hasFrame = true
}

func (p *Processor) ProcessFrame(samples []float32) protocol.ReferenceMetrics {
	p.rmsDbfs = audiomath.RmsDbfs(samples)
	p.peakDbfs = audiomath.PeakDbfs(samples)
	p.vad = p.updateVAD(p.rmsDbfs)
	return protocol.ReferenceMetrics{
		StreamID: protocol.PlaybackRefStreamID,
		RmsDbfs:  p.rmsDbfs,
		PeakDbfs: p.peakDbfs,
		Vad:      p.vad,
		Active:   p.vad,
	}
}

func (p *Processor) updateVAD(rmsDbfs float64) bool {
	now := time.Now()
	if p.vad {
		if rmsDbfs < p.cfg.ReferenceVadOffDbfs {
			if now.After(p.holdUntil) {
				p.vad = false
			}
		} else {
			p.holdUntil = now.Add(time.Duration(p.cfg.VadHangoverMs) * time.Millisecond)
		}
	} else if rmsDbfs > p.cfg.ReferenceVadOnDbfs {
		p.vad = true
		p.holdUntil = now.Add(time.Duration(p.cfg.VadHangoverMs) * time.Millisecond)
	}
	return p.vad
}

func (p *Processor) EchoPenalty(micSamples, refSamples []float32) float64 {
	return audiomath.NormalizedCorrelation(micSamples, refSamples)
}

func (p *Processor) DelayTracker() *DelayTracker {
	return p.delay
}

func (p *Processor) ResetLoopDelaySamples() {
	p.delay.Reset()
}
