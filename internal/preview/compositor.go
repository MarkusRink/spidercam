package preview

import (
	"context"
	"time"

	"github.com/markus/spidercam/internal/protocol"
	"github.com/markus/spidercam/internal/room"
)

const compositorFPS = 30

func RunMockCompositor(ctx context.Context, stream *Stream, rm *room.Room, onCut func(cut bool, sel *protocol.SelectionState)) {
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
				cut := stream.OnFrame(frame, sel)
				if onCut != nil {
					onCut(cut, sel)
				}
			}
		}
	}()
}
