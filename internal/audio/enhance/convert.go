package enhance

import (
	"math"

	audiomath "github.com/markus/spidercam/internal/audio/math"
)

func float32ToInt16(samples []float32, dst []int16) {
	for i, s := range samples {
		v := float64(s) * 32767
		if v > 32767 {
			v = 32767
		} else if v < -32768 {
			v = -32768
		}
		dst[i] = int16(v)
	}
}

func int16ToFloat32(samples []int16, dst []float32) {
	for i, s := range samples {
		dst[i] = float32(float64(s) / 32768)
	}
}

func clipFloat(v float64) float64 {
	return audiomath.Clamp(v, -1, 1)
}

func clipFloat32(v float32) float32 {
	return float32(clipFloat(float64(v)))
}

func FloatToInt16(s float32) int16 {
	if s >= 1 {
		return 32767
	}
	if s <= -1 {
		return -32768
	}
	return int16(float64(s) * 32767)
}

func rmsDiff(a, b []float32) float64 {
	if len(a) == 0 || len(b) == 0 {
		return math.MaxFloat64
	}
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	var sum float64
	for i := 0; i < n; i++ {
		d := float64(a[i] - b[i])
		sum += d * d
	}
	return math.Sqrt(sum / float64(n))
}
