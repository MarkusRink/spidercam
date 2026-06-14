package signaling

import (
	"context"

	"github.com/markus/spidercam/internal/capture"
	"github.com/markus/spidercam/internal/protocol"
)

type CaptureReopener interface {
	ReopenCapture(ctx context.Context, sel protocol.CaptureSelection) error
}

type CaptureBundleReopener struct {
	Bundle *capture.Bundle
}

func (r *CaptureBundleReopener) ReopenCapture(ctx context.Context, sel protocol.CaptureSelection) error {
	if r == nil || r.Bundle == nil {
		return nil
	}
	return r.Bundle.Reopen(ctx, sel)
}

func reopenCaptureState(ctx context.Context, reopener CaptureReopener, state protocol.CaptureState) error {
	if reopener == nil {
		return nil
	}
	return reopener.ReopenCapture(ctx, protocol.CaptureSelection{
		MicID:    state.MicID,
		CameraID: state.CameraID,
		SinkID:   state.SinkID,
	})
}
