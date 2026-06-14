package signaling

import (
	"github.com/markus/spidercam/internal/protocol"
	"github.com/markus/spidercam/internal/room"
)

type StreamProcessor interface {
	SetStreamProcessing(participantID string, flags protocol.StreamProcessingFlags) error
}

func applyStreamProcessing(r *room.Room, proc StreamProcessor, participantID string, flags protocol.StreamProcessingFlags) bool {
	if !r.SetStreamProcessing(participantID, flags) {
		return false
	}
	if proc != nil {
		_ = proc.SetStreamProcessing(participantID, flags)
	}
	return true
}
