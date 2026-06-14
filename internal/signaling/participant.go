package signaling

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/markus/spidercam/internal/protocol"
	"github.com/markus/spidercam/internal/room"
	spwebrtc "github.com/markus/spidercam/internal/webrtc"
)

const participantViewThrottle = 100 * time.Millisecond

type ParticipantHub struct {
	room   *room.Room
	webrtc *spwebrtc.Hub

	mu               sync.Mutex
	clients          map[*wsClient]string
	lastBroadcast    time.Time
	pendingBroadcast *time.Timer
}

func NewParticipantHub(r *room.Room, webrtcHub *spwebrtc.Hub) *ParticipantHub {
	p := &ParticipantHub{
		room:    r,
		webrtc:  webrtcHub,
		clients: make(map[*wsClient]string),
	}
	webrtcHub.SetICECandidateHandler(p.sendICECandidate)
	return p
}

func (p *ParticipantHub) HandleUpgrade(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	clientID := newClientID()
	client := &wsClient{conn: conn}

	p.mu.Lock()
	p.clients[client] = clientID
	p.mu.Unlock()

	_ = client.sendJSON(protocol.WelcomeMsg{
		Type:     "welcome",
		ClientID: clientID,
		View:     p.room.ViewFor(clientID),
	})

	go func() {
		defer func() {
			p.mu.Lock()
			delete(p.clients, client)
			p.mu.Unlock()
			p.webrtc.RemovePeer(clientID)
			_ = conn.Close()
		}()
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			p.handleMessage(client, clientID, data)
		}
	}()
}

func (p *ParticipantHub) ScheduleBroadcast() {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(p.lastBroadcast)
	if elapsed >= participantViewThrottle {
		p.broadcastViewsLocked()
		return
	}
	if p.pendingBroadcast != nil {
		return
	}
	wait := participantViewThrottle - elapsed
	p.pendingBroadcast = time.AfterFunc(wait, func() {
		p.mu.Lock()
		p.pendingBroadcast = nil
		p.broadcastViewsLocked()
		p.mu.Unlock()
	})
}

func (p *ParticipantHub) handleMessage(client *wsClient, clientID string, data []byte) {
	var envelope struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		_ = client.sendJSON(protocol.ErrorMsg{Type: "error", Message: "invalid json"})
		return
	}

	switch envelope.Type {
	case "join":
		var msg protocol.JoinMsg
		if err := json.Unmarshal(data, &msg); err != nil {
			_ = client.sendJSON(protocol.ErrorMsg{Type: "error", Message: "invalid json"})
			return
		}
		if strings.TrimSpace(msg.Name) == "" {
			_ = client.sendJSON(protocol.ErrorMsg{Type: "error", Message: "name required"})
			return
		}
		p.room.Join(clientID, strings.TrimSpace(msg.Name), msg.HasVideo, msg.HasAudio)
		_ = client.sendJSON(protocol.ParticipantViewMsg{
			Type: "participant-view",
			View: p.room.ViewFor(clientID),
		})
		p.ScheduleBroadcast()
	case "leave":
		p.room.Leave(clientID)
		p.webrtc.RemovePeer(clientID)
		p.ScheduleBroadcast()
	case "offer":
		var msg protocol.OfferMsg
		if err := json.Unmarshal(data, &msg); err != nil {
			_ = client.sendJSON(protocol.ErrorMsg{Type: "error", Message: "invalid json"})
			return
		}
		if strings.TrimSpace(msg.SDP) == "" {
			_ = client.sendJSON(protocol.ErrorMsg{Type: "error", Message: "sdp required"})
			return
		}
		answerSDP, err := p.webrtc.HandleOffer(clientID, msg.SDP)
		if err != nil {
			_ = client.sendJSON(protocol.ErrorMsg{Type: "error", Message: err.Error()})
			return
		}
		_ = client.sendJSON(protocol.AnswerMsg{
			Type: "answer",
			SDP:  answerSDP,
		})
	case "ice-candidate":
		var msg protocol.ICECandidateMsg
		if err := json.Unmarshal(data, &msg); err != nil {
			_ = client.sendJSON(protocol.ErrorMsg{Type: "error", Message: "invalid json"})
			return
		}
		if strings.TrimSpace(msg.Candidate) == "" {
			return
		}
		_ = p.webrtc.HandleICE(clientID, msg.Candidate)
	}
}

func (p *ParticipantHub) sendICECandidate(clientID, candidate string) {
	p.mu.Lock()
	var target *wsClient
	for client, id := range p.clients {
		if id == clientID {
			target = client
			break
		}
	}
	p.mu.Unlock()
	if target == nil {
		return
	}
	_ = target.sendJSON(protocol.ICECandidateMsg{
		Type:      "ice-candidate",
		Candidate: candidate,
	})
}

func (p *ParticipantHub) broadcastViewsLocked() {
	p.lastBroadcast = time.Now()
	for client, clientID := range p.clients {
		_ = client.sendJSON(protocol.ParticipantViewMsg{
			Type: "participant-view",
			View: p.room.ViewFor(clientID),
		})
	}
}

func RegisterParticipantREST(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/health", handleHealth)
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
