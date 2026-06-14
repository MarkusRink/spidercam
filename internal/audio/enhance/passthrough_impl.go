package enhance

import (
	"time"

	audiomath "github.com/markus/spidercam/internal/audio/math"
)

type passthrough struct {
	out       []float32
	synthetic bool
}

func newPassthrough(synthetic bool) *passthrough {
	return &passthrough{
		out:       make([]float32, audiomath.FrameSamples),
		synthetic: synthetic,
	}
}

func (p *passthrough) Process(frame []float32) ([]float32, float64) {
	start := time.Now()
	n := len(frame)
	if n > len(p.out) {
		p.out = make([]float32, n)
	}
	copy(p.out[:n], frame)
	var us float64
	if p.synthetic {
		us = 28
	} else {
		us = float64(time.Since(start).Microseconds())
		if us < 1 {
			us = 1
		}
	}
	return p.out[:n], us
}

func (p *passthrough) Reset() {}
