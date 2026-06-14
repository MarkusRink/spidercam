package preview

import (
	"testing"

	"github.com/markus/spidercam/internal/protocol"
)

func TestSubsampleTick(t *testing.T) {
	var sub subsample
	var ticks int
	for i := 0; i < 6; i++ {
		if sub.Tick() {
			ticks++
		}
	}
	if ticks != 3 {
		t.Fatalf("subsample ticks = %d, want 3 from 6 compositor frames", ticks)
	}
}

func TestStreamSubsampleEmitsHalfRate(t *testing.T) {
	annexB := loadKeyframeFixture(t)
	stream, err := New(Config{
		Width:        DefaultWidth,
		Height:       DefaultHeight,
		FPS:          DefaultFPS,
		Mock:         true,
		MockKeyframe: annexB,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer stream.Close()

	sel := &protocol.SelectionState{ActiveVideoID: protocol.HostStreamID}
	for i := 0; i < 4; i++ {
		stream.OnFrame(VideoFrame{}, sel)
	}

	count := 0
	for {
		select {
		case <-stream.Chunks():
			count++
		default:
			goto done
		}
	}
done:
	if count != 2 {
		t.Fatalf("encoded chunks = %d, want 2 from 4 compositor frames", count)
	}
}

func TestStreamForceKeyframeOnSelectionChange(t *testing.T) {
	annexB := loadKeyframeFixture(t)
	stream, err := New(Config{
		Width:        DefaultWidth,
		Height:       DefaultHeight,
		FPS:          DefaultFPS,
		Mock:         true,
		MockKeyframe: annexB,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer stream.Close()

	selA := &protocol.SelectionState{ActiveVideoID: "stream-a"}
	selB := &protocol.SelectionState{ActiveVideoID: "stream-b"}

	stream.OnFrame(VideoFrame{}, selA)
	stream.OnFrame(VideoFrame{}, selA)
	<-stream.Chunks()

	stream.OnFrame(VideoFrame{}, selA)
	cut := stream.OnFrame(VideoFrame{}, selB)
	if !cut {
		t.Fatal("expected preview cut when active video changes")
	}

	chunk := <-stream.Chunks()
	if chunk[0]&flagKeyframe == 0 {
		t.Fatal("expected keyframe flag after active video change")
	}
}

func TestStreamPackChunkOutput(t *testing.T) {
	annexB := loadKeyframeFixture(t)
	stream, err := New(Config{
		Mock:         true,
		MockKeyframe: annexB,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer stream.Close()

	sel := &protocol.SelectionState{ActiveVideoID: "host"}
	stream.OnFrame(VideoFrame{}, sel)
	stream.OnFrame(VideoFrame{}, sel)

	chunk := <-stream.Chunks()
	if len(chunk) <= chunkHeaderSize {
		t.Fatalf("chunk too short: %d bytes", len(chunk))
	}
	if chunk[0]&flagKeyframe == 0 {
		t.Fatal("first emitted chunk should be a keyframe")
	}
}
