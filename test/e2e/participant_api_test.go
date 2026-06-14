//go:build e2e

package e2e_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/markus/spidercam/internal/protocol"
)

func TestParticipantWelcomeJoinView(t *testing.T) {
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
		Name:     "e2e-participant",
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
	if view.View.SelfMetric.CalibrationPhase == "" {
		t.Fatal("expected self metric in participant view")
	}
}

func TestParticipantNeverReceivesFullRoomState(t *testing.T) {
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

	if err := conn.WriteJSON(protocol.JoinMsg{
		Type:     "join",
		Name:     "e2e-scope",
		HasVideo: true,
		HasAudio: true,
	}); err != nil {
		t.Fatalf("send join: %v", err)
	}

	type result struct {
		data []byte
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				ch <- result{err: err}
				return
			}
			ch <- result{data: data}
		}
	}()

	deadline := time.After(2 * time.Second)
	for {
		select {
		case res := <-ch:
			if res.err != nil {
				return
			}
			if isFullRoomStatePayload(res.data) {
				t.Fatalf("participant received full room state: %s", res.data)
			}
		case <-deadline:
			return
		}
	}
}

func isFullRoomStatePayload(data []byte) bool {
	var envelope struct {
		Type  string          `json:"type"`
		State json.RawMessage `json:"state"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return false
	}
	if envelope.Type == "host-state" {
		return true
	}
	if len(envelope.State) == 0 {
		return false
	}
	var room struct {
		Metrics   json.RawMessage `json:"metrics"`
		Reference json.RawMessage `json:"reference"`
		Capture   json.RawMessage `json:"capture"`
	}
	if err := json.Unmarshal(envelope.State, &room); err != nil {
		return false
	}
	return len(room.Metrics) > 0 || len(room.Reference) > 0 || len(room.Capture) > 0
}
