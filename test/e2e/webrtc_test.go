//go:build e2e

package e2e_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	pion "github.com/pion/webrtc/v4"
	"github.com/markus/spidercam/internal/protocol"
)

func TestWebRTCOfferAnswer(t *testing.T) {
	d := StartMockDaemon(t)

	conn, _, err := websocket.DefaultDialer.Dial(d.ParticipantWS(), nil)
	if err != nil {
		t.Fatalf("dial participant ws: %v", err)
	}
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	var welcome protocol.WelcomeMsg
	if err := conn.ReadJSON(&welcome); err != nil {
		t.Fatalf("read welcome: %v", err)
	}
	if welcome.Type != "welcome" || welcome.ClientID == "" {
		t.Fatalf("unexpected welcome: %+v", welcome)
	}

	if err := conn.WriteJSON(protocol.JoinMsg{
		Type:     "join",
		Name:     "e2e-webrtc",
		HasVideo: true,
		HasAudio: true,
	}); err != nil {
		t.Fatalf("send join: %v", err)
	}

	var view protocol.ParticipantViewMsg
	if err := conn.ReadJSON(&view); err != nil {
		t.Fatalf("read participant-view: %v", err)
	}
	if view.Type != "participant-view" {
		t.Fatalf("type = %q, want participant-view", view.Type)
	}

	offerSDP, cleanup := minimalOfferSDP(t)
	defer cleanup()

	if err := conn.WriteJSON(protocol.OfferMsg{
		Type: "offer",
		SDP:  offerSDP,
	}); err != nil {
		t.Fatalf("send offer: %v", err)
	}

	answer, err := readAnswer(t, conn)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(answer.SDP) == "" {
		t.Fatal("expected non-empty answer sdp")
	}
	if !strings.Contains(answer.SDP, "v=0") {
		t.Fatalf("answer sdp does not look like SDP: %q", answer.SDP)
	}
}

func readAnswer(t *testing.T, conn *websocket.Conn) (protocol.AnswerMsg, error) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, data, err := conn.ReadMessage()
		if err != nil {
			return protocol.AnswerMsg{}, fmt.Errorf("read message: %w", err)
		}
		var envelope struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(data, &envelope); err != nil {
			return protocol.AnswerMsg{}, fmt.Errorf("decode envelope: %w", err)
		}
		switch envelope.Type {
		case "answer":
			var answer protocol.AnswerMsg
			if err := json.Unmarshal(data, &answer); err != nil {
				return protocol.AnswerMsg{}, fmt.Errorf("decode answer: %w", err)
			}
			return answer, nil
		case "participant-view", "ice-candidate":
			continue
		case "error":
			var errMsg protocol.ErrorMsg
			_ = json.Unmarshal(data, &errMsg)
			return protocol.AnswerMsg{}, fmt.Errorf("server error: %s", errMsg.Message)
		default:
			return protocol.AnswerMsg{}, fmt.Errorf("unexpected message type %q", envelope.Type)
		}
	}
	return protocol.AnswerMsg{}, fmt.Errorf("timed out waiting for answer")
}

func minimalOfferSDP(t *testing.T) (string, func()) {
	t.Helper()

	pc, err := pion.NewPeerConnection(pion.Configuration{
		ICEServers: []pion.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}},
	})
	if err != nil {
		t.Fatalf("client peer connection: %v", err)
	}

	track, err := pion.NewTrackLocalStaticSample(
		pion.RTPCodecCapability{MimeType: pion.MimeTypeOpus},
		"audio",
		"pion",
	)
	if err != nil {
		_ = pc.Close()
		t.Fatalf("audio track: %v", err)
	}
	if _, err := pc.AddTrack(track); err != nil {
		_ = pc.Close()
		t.Fatalf("add track: %v", err)
	}

	offer, err := pc.CreateOffer(nil)
	if err != nil {
		_ = pc.Close()
		t.Fatalf("create offer: %v", err)
	}
	if err := pc.SetLocalDescription(offer); err != nil {
		_ = pc.Close()
		t.Fatalf("set local description: %v", err)
	}

	return offer.SDP, func() { _ = pc.Close() }
}
