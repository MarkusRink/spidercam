package preview

import "time"

const DefaultCodec = "avc1.42E01E"

type EncodeResult struct {
	AVCC     []byte
	Keyframe bool
}

type Encoder interface {
	Encode(rgba []byte, w, h int, ts time.Time) (EncodeResult, error)
	ForceKeyframe()
	Close() error
}
