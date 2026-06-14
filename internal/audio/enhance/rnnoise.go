//go:build cgo && linux

package enhance

func New() (Denoiser, error) {
	return newPassthrough(false), nil
}

func NewPassthrough() Denoiser {
	return newPassthrough(false)
}
