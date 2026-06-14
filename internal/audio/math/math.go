package audiomath

import (
	"math"

	"github.com/markus/spidercam/internal/protocol"
)

const (
	Eps          = 1e-8
	FrameSamples = 480
	SampleRate   = 48000
	FrameMs      = 10
)

func Clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func RMS(samples []float32) float64 {
	if len(samples) == 0 {
		return 0
	}
	var sum float64
	for _, s := range samples {
		f := float64(s)
		sum += f * f
	}
	return math.Sqrt(sum / float64(len(samples)))
}

func Peak(samples []float32) float64 {
	var peak float64
	for _, s := range samples {
		a := math.Abs(float64(s))
		if a > peak {
			peak = a
		}
	}
	return peak
}

func LinearToDb(linear float64) float64 {
	return 20 * math.Log10(linear+Eps)
}

func DbToLinear(db float64) float64 {
	return math.Pow(10, db/20)
}

func RmsDbfs(samples []float32) float64 {
	return LinearToDb(RMS(samples))
}

func PeakDbfs(samples []float32) float64 {
	return LinearToDb(Peak(samples))
}

func SnrDb(signalRms, noiseRms float64) float64 {
	return 20 * math.Log10((signalRms+Eps)/(noiseRms+Eps))
}

func EMA(prev, next, alpha float64) float64 {
	return (1-alpha)*prev + alpha*next
}

func NormLevel(dbfs float64) float64 {
	return Clamp((dbfs+60)/40, 0, 1)
}

func NormSnr(snr float64) float64 {
	return Clamp(snr/20, 0, 1)
}

func NormalizedCorrelation(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		af, bf := float64(a[i]), float64(b[i])
		dot += af * bf
		na += af * af
		nb += bf * bf
	}
	den := math.Sqrt(na * nb)
	if den < Eps {
		return 0
	}
	return Clamp(dot/den, 0, 1)
}

func GccPhatPeakLag(ref, mic []float32, searchMs int) (lagMs float64, quality float64) {
	if len(ref) == 0 || len(mic) == 0 || searchMs <= 0 {
		return 0, 0
	}
	searchSamples := searchMs * SampleRate / 1000
	bestLag := 0
	best := 0.0
	for lag := -searchSamples; lag <= searchSamples; lag++ {
		c := laggedCorrelation(ref, mic, lag)
		if c > best {
			best = c
			bestLag = lag
		}
	}
	return float64(bestLag) * 1000 / float64(SampleRate), best
}

func laggedCorrelation(ref, mic []float32, lag int) float64 {
	var dot, na, nb float64
	for i := range ref {
		j := i + lag
		if j < 0 || j >= len(mic) {
			continue
		}
		af, bf := float64(ref[i]), float64(mic[j])
		dot += af * bf
		na += af * af
		nb += bf * bf
	}
	den := math.Sqrt(na * nb)
	if den < Eps {
		return 0
	}
	return Clamp(dot/den, 0, 1)
}

type ScoreInput struct {
	RmsDbfs, SnrDb float64
	Vad            bool
	Role           protocol.StreamRole
	EchoPenalty    float64
	HostPriority   float64
}

func BuildScoreComponents(in ScoreInput) protocol.ScoreComponents {
	vad := 0.0
	if in.Vad {
		vad = 1
	}
	priority := 0.0
	if in.Role == protocol.StreamRoleHost {
		priority = in.HostPriority
	}
	return protocol.ScoreComponents{
		Level:       NormLevel(in.RmsDbfs),
		Snr:         NormSnr(in.SnrDb),
		Vad:         vad,
		Priority:    priority,
		EchoPenalty: Clamp(in.EchoPenalty, 0, 1),
	}
}

func FrameScore(c protocol.ScoreComponents, w protocol.ScoreWeights) float64 {
	raw := w.Level*c.Level +
		w.Snr*c.Snr +
		w.Vad*c.Vad +
		w.Priority*c.Priority -
		w.EchoPenalty*c.EchoPenalty
	return Clamp(raw, 0, 1)
}

func EqualPowerGains(t float64) (from, to float64) {
	t = Clamp(t, 0, 1)
	theta := t * math.Pi / 2
	return math.Cos(theta), math.Sin(theta)
}

func AttackRelease(current, target, attack, release float64) float64 {
	if target > current {
		return current + attack*(target-current)
	}
	return current + release*(target-current)
}
