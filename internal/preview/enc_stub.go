//go:build !cgo || !linux

package preview

import "errors"

func openX264(cfg Config) (Encoder, error) {
	return nil, errors.New("preview: x264 encoder requires cgo on linux")
}
