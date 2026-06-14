package aec

import (
	"time"

	audiomath "github.com/markus/spidercam/internal/audio/math"
)

type passthrough struct {
	out       []float32
	processUs float64
	synthetic bool
}

func newPassthrough(_ Config, synthetic bool) *passthrough {
	return &passthrough{
		out:       make([]float32, audiomath.FrameSamples),
		synthetic: synthetic,
	}
}

func (p *passthrough) ProcessReverse(_ []float32) {}

func (p *passthrough) ProcessNear(mic []float32) []float32 {
	start := time.Now()
	n := len(mic)
	if n > len(p.out) {
		p.out = make([]float32, n)
	}
	copy(p.out[:n], mic)
	if p.synthetic {
		p.processUs = 42
	} else {
		p.processUs = float64(time.Since(start).Microseconds())
		if p.processUs < 1 {
			p.processUs = 1
		}
	}
	return p.out[:n]
}

func (p *passthrough) Reset() {
	p.processUs = 0
}

func (p *passthrough) Stats() Stats {
	return Stats{
		ErleDb:    0,
		Converged: false,
		ProcessUs: p.processUs,
	}
}
