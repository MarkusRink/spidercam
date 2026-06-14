package webrtc

import (
	"encoding/json"
	"fmt"
	"sync"

	pion "github.com/pion/webrtc/v4"
	"github.com/markus/spidercam/internal/room"
)

type Hub struct {
	room *room.Room

	mu               sync.Mutex
	peers            map[string]*peer
	onICECandidate   func(clientID, candidate string)
}

func NewHub(r *room.Room) *Hub {
	return &Hub{
		room:  r,
		peers: make(map[string]*peer),
	}
}

func (h *Hub) SetICECandidateHandler(fn func(clientID, candidate string)) {
	h.mu.Lock()
	h.onICECandidate = fn
	h.mu.Unlock()
}

func (h *Hub) HandleOffer(clientID, sdp string) (answer string, err error) {
	peer, err := h.getOrCreate(clientID)
	if err != nil {
		return "", err
	}
	return peer.setRemoteOfferAndCreateAnswer(sdp)
}

func (h *Hub) HandleICE(clientID, candidate string) error {
	h.mu.Lock()
	peer, ok := h.peers[clientID]
	h.mu.Unlock()
	if !ok {
		return fmt.Errorf("webrtc: unknown peer %q", clientID)
	}
	return peer.addICE(candidate)
}

func (h *Hub) RemovePeer(clientID string) {
	h.mu.Lock()
	peer, ok := h.peers[clientID]
	if ok {
		delete(h.peers, clientID)
	}
	h.mu.Unlock()
	if ok {
		_ = peer.close()
	}
}

func (h *Hub) Stats(clientID string) TransportStats {
	h.mu.Lock()
	_, ok := h.peers[clientID]
	h.mu.Unlock()
	if !ok {
		return TransportStats{}
	}
	return TransportStats{}
}

func (h *Hub) getOrCreate(clientID string) (*peer, error) {
	h.mu.Lock()
	if peer, ok := h.peers[clientID]; ok {
		h.mu.Unlock()
		return peer, nil
	}
	onICE := h.onICECandidate
	h.mu.Unlock()

	peer, err := newPeer(clientID, func(candidate string) {
		if onICE != nil {
			onICE(clientID, candidate)
		}
	})
	if err != nil {
		return nil, err
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if existing, ok := h.peers[clientID]; ok {
		_ = peer.close()
		return existing, nil
	}
	h.peers[clientID] = peer
	return peer, nil
}

func mustMarshalICE(init pion.ICECandidateInit) string {
	b, err := json.Marshal(init)
	if err != nil {
		return ""
	}
	return string(b)
}

func parseICE(candidateJSON string) (pion.ICECandidateInit, error) {
	var init pion.ICECandidateInit
	if err := json.Unmarshal([]byte(candidateJSON), &init); err != nil {
		return init, fmt.Errorf("parse ice candidate: %w", err)
	}
	return init, nil
}
