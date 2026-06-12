# Audio math

**Target:** `internal/audio/math/math.go`

Port of research formulas. Pure functions, no allocations in hot path where possible (reuse buffers in engine).

```go
package math

const (
	Eps           = 1e-8
	FrameSamples  = 480
	SampleRate    = 48000
	FrameMs       = 10
)

func Clamp(v, min, max float64) float64
func RMS(samples []float32) float64
func Peak(samples []float32) float64
func LinearToDb(linear float64) float64
func DbToLinear(db float64) float64
func RmsDbfs(samples []float32) float64
func PeakDbfs(samples []float32) float64
func SnrDb(signalRms, noiseRms float64) float64
func EMA(prev, next, alpha float64) float64
func NormLevel(dbfs float64) float64
func NormSnr(snr float64) float64

func NormalizedCorrelation(a, b []float32) float64
func GccPhatPeakLag(ref, mic []float32, searchMs int) (lagMs float64, quality float64)

type ScoreInput struct {
	RmsDbfs, SnrDb float64
	Vad            bool
	Role           protocol.StreamRole
	EchoPenalty    float64
	HostPriority   float64
}

func BuildScoreComponents(in ScoreInput) protocol.ScoreComponents
func FrameScore(c protocol.ScoreComponents, w protocol.ScoreWeights) float64
func EqualPowerGains(t float64) (from, to float64)
func AttackRelease(current, target, attack, release float64) float64
```

## Correlation (reference loopback)

```go
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
```

## GCC-PHAT (passive loop delay)

```go
// GccPhatPeakLag returns peak lag in ms and peak quality 0..1.
// searchMs: search ±searchMs around zero lag at SampleRate.
func GccPhatPeakLag(ref, mic []float32, searchMs int) (lagMs float64, quality float64)
```

Used by `reference.DelayTracker` — separate from lag-0 `NormalizedCorrelation` for `echoPenalty`.

## Tests

`math_test.go`: rmsDbfs, normLevel, frameScore, equalPowerGains, correlation, GCC-PHAT peak at known synthetic lag.
