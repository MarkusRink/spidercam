package preview

import "time"

type mockEncoder struct {
	avcc          []byte
	frameIndex    int
	forceKeyframe bool
	keyint        int
}

func newMockEncoder(annexB []byte, fps int) *mockEncoder {
	keyint := fps
	if keyint <= 0 {
		keyint = 15
	}
	return &mockEncoder{
		avcc:          AnnexBToAVCC(annexB),
		forceKeyframe: true,
		keyint:        keyint,
	}
}

func (e *mockEncoder) Encode(_ []byte, _, _ int, _ time.Time) (EncodeResult, error) {
	keyframe := e.forceKeyframe || e.frameIndex%e.keyint == 0
	e.forceKeyframe = false
	e.frameIndex++
	return EncodeResult{
		AVCC:     e.avcc,
		Keyframe: keyframe,
	}, nil
}

func (e *mockEncoder) ForceKeyframe() {
	e.forceKeyframe = true
}

func (e *mockEncoder) Close() error {
	return nil
}
