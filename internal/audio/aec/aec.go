package aec

type Processor interface {
	ProcessReverse(ref []float32)
	ProcessNear(mic []float32) []float32
	Reset()
	Stats() Stats
}

type Stats struct {
	ErleDb    float64
	Converged bool
	ProcessUs float64
}

type Config struct {
	Enabled bool
}
