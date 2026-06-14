package mixer

import (
	audiomath "github.com/markus/spidercam/internal/audio/math"
	"github.com/markus/spidercam/internal/audio/processor"
	"github.com/markus/spidercam/internal/protocol"
)

type Frame struct {
	PCM         []float32
	OutDbfs     float64
	OutPeakDbfs float64
}

type Mixer struct {
	outBuf []float32
}

func New() *Mixer {
	return &Mixer{outBuf: make([]float32, audiomath.FrameSamples)}
}

func (m *Mixer) Mix(
	streams map[string]*processor.Pipeline,
	sel protocol.SelectionState,
	refMetrics protocol.ReferenceMetrics,
	cfg protocol.HostConfig,
) Frame {
	for i := range m.outBuf {
		m.outBuf[i] = 0
	}

	if sel.MixerState == protocol.MixerSilence {
		return Frame{
			PCM:         append([]float32(nil), m.outBuf...),
			OutDbfs:     audiomath.RmsDbfs(m.outBuf),
			OutPeakDbfs: audiomath.PeakDbfs(m.outBuf),
		}
	}

	var fromGain, toGain float64 = 1, 0
	activeID := sel.ActiveAudioID
	if sel.Crossfade != nil && sel.Crossfade.T < 1 {
		fromGain, toGain = audiomath.EqualPowerGains(sel.Crossfade.T)
		m.addStream(m.outBuf, streams, sel.Crossfade.FromID, fromGain)
		m.addStream(m.outBuf, streams, sel.Crossfade.ToID, toGain)
	} else if activeID != "" {
		m.addStream(m.outBuf, streams, activeID, 1)
	}

	if refMetrics.Active && cfg.ReferenceDuckDb < 0 {
		duck := audiomath.DbToLinear(cfg.ReferenceDuckDb)
		for i := range m.outBuf {
			m.outBuf[i] *= float32(duck)
		}
	}

	m.softLimit(m.outBuf)

	return Frame{
		PCM:         append([]float32(nil), m.outBuf...),
		OutDbfs:     audiomath.RmsDbfs(m.outBuf),
		OutPeakDbfs: audiomath.PeakDbfs(m.outBuf),
	}
}

func (m *Mixer) addStream(out []float32, streams map[string]*processor.Pipeline, id string, gain float64) {
	p, ok := streams[id]
	if !ok || gain == 0 {
		return
	}
	frame := p.LastFrame()
	for i := range out {
		if i < len(frame) {
			out[i] += float32(float64(frame[i]) * gain)
		}
	}
}

func (m *Mixer) softLimit(samples []float32) {
	const ceiling = 0.95
	for i, s := range samples {
		v := float64(s)
		abs := v
		if abs < 0 {
			abs = -abs
		}
		if abs > ceiling {
			sign := 1.0
			if v < 0 {
				sign = -1
			}
			samples[i] = float32(sign * ceiling)
		}
	}
}
