package enhance

type Denoiser interface {
	Process(frame []float32) (out []float32, processUs float64)
	Reset()
}
