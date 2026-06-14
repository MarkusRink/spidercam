//go:build !cgo || !linux

package enhance

func New() (Denoiser, error) {
	return newPassthrough(true), nil
}

func NewPassthrough() Denoiser {
	return newPassthrough(true)
}
