//go:build !cgo || !linux

package aec

func New(cfg Config) (Processor, error) {
	return newPassthrough(cfg, true), nil
}
