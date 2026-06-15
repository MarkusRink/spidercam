package preview

import (
	"context"
	"time"

	"github.com/markus/spidercam/internal/output"
	"github.com/markus/spidercam/internal/protocol"
	"github.com/markus/spidercam/internal/room"
)

const compositorFPS = 30

type CameraReader interface {
	ReadCamera() (rgba []byte, width, height int, ok bool)
}

func RunMockCompositor(ctx context.Context, stream *Stream, rm *room.Room, onCut func(cut bool, sel *protocol.SelectionState)) {
	RunCompositor(ctx, stream, rm, nil, nil, onCut)
}

func RunCompositor(ctx context.Context, stream *Stream, rm *room.Room, camera CameraReader, videoOut output.Writer, onCut func(cut bool, sel *protocol.SelectionState)) {
	width := stream.cfg.Width
	height := stream.cfg.Height
	rgba := make([]byte, width*height*4)
	for i := 0; i < len(rgba)/4; i++ {
		rgba[i*4+0] = 255
		rgba[i*4+3] = 255
	}
	frame := VideoFrame{RGBA: rgba, Width: width, Height: height}

	ticker := time.NewTicker(time.Second / compositorFPS)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				state := rm.State()
				var sel *protocol.SelectionState
				if state.Selection != nil {
					copy := *state.Selection
					sel = &copy
				}
				if camera != nil {
					if camRGBA, w, h, ok := camera.ReadCamera(); ok && len(camRGBA) > 0 {
						if w == width && h == height {
							copy(frame.RGBA, camRGBA)
						} else {
							scaleRGBA(frame.RGBA, width, height, camRGBA, w, h)
						}
					}
				}
				cut := stream.OnFrame(frame, sel)
				if videoOut != nil && videoOut.Healthy() {
					_ = videoOut.WriteVideo(frame.RGBA, width, height)
				}
				if onCut != nil {
					onCut(cut, sel)
				}
			}
		}
	}()
}
