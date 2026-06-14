package jitter

import (
	audiomath "github.com/markus/spidercam/internal/audio/math"
)

type Buffer struct {
	targetFrames int
	frames       [][]float32
}

func NewBuffer(targetFrames int) *Buffer {
	if targetFrames < 1 {
		targetFrames = 5
	}
	return &Buffer{targetFrames: targetFrames}
}

func (b *Buffer) Push(pcm []float32) {
	if len(pcm) == 0 {
		return
	}
	frame := make([]float32, len(pcm))
	copy(frame, pcm)
	b.frames = append(b.frames, frame)
	max := b.targetFrames * 2
	if len(b.frames) > max {
		b.frames = b.frames[len(b.frames)-max:]
	}
}

func (b *Buffer) Pull() ([]float32, bool) {
	if len(b.frames) == 0 {
		return nil, false
	}
	frame := b.frames[0]
	b.frames = b.frames[1:]
	return frame, true
}

func (b *Buffer) Depth() int {
	return len(b.frames)
}

func (b *Buffer) PushSilence() {
	b.Push(make([]float32, audiomath.FrameSamples))
}
