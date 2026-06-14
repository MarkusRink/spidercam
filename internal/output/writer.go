package output

import audiomath "github.com/markus/spidercam/internal/audio/math"

type Writer interface {
	WritePCM(samples []float32) error
	WriteVideo(rgba []byte, width, height int) error
	Healthy() bool
	Close() error
}

const FrameSamples = audiomath.FrameSamples
