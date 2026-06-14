package protocol

import "errors"

var (
	errCrossfadeMsOutOfRange = errors.New("crossfadeMs out of range")
	errAudioHoldMsOutOfRange = errors.New("audioHoldMs out of range")
	ErrStreamNotFound        = errors.New("stream not found")
)
