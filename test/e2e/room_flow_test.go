//go:build e2e

package e2e_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/markus/spidercam/internal/protocol"
)

func TestParticipantJoinUpdatesHostStateAndLeaveCleansUp(t *testing.T) {
	d := StartMockDaemon(t)

	before := fetchHostState(t, d)
	beforeCount := len(before.Participants)

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

	name := "room-flow-e2e"
	if err := conn.WriteJSON(protocol.JoinMsg{
		Type:     "join",
		Name:     name,
		HasVideo: true,
		HasAudio: true,
	}); err != nil {
		t.Fatalf("send join: %v", err)
	}
	var view protocol.ParticipantViewMsg
	if err := conn.ReadJSON(&view); err != nil {
		t.Fatalf("read participant-view: %v", err)
	}

	waitFor(t, 5*time.Second, func() bool {
		state := fetchHostState(t, d)
		return len(state.Participants) == beforeCount+1 && metricNamed(state.Metrics, name)
	})

	if err := conn.WriteJSON(protocol.LeaveMsg{Type: "leave"}); err != nil {
		t.Fatalf("send leave: %v", err)
	}

	waitFor(t, 5*time.Second, func() bool {
		state := fetchHostState(t, d)
		return len(state.Participants) == beforeCount && !participantPresent(state.Participants, welcome.ClientID)
	})
}

func fetchHostState(t *testing.T, d *Daemon) protocol.RoomState {
	t.Helper()
	resp, err := http.Get(d.HostBase() + "/api/v1/host/state")
	if err != nil {
		t.Fatalf("GET host state: %v", err)
	}
	defer resp.Body.Close()
	var state protocol.RoomState
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		t.Fatalf("decode host state: %v", err)
	}
	return state
}

func metricNamed(metrics []protocol.StreamMetrics, name string) bool {
	for _, m := range metrics {
		if m.Name == name {
			return true
		}
	}
	return false
}

func participantPresent(participants []protocol.ParticipantInfo, id string) bool {
	for _, p := range participants {
		if p.ID == id {
			return true
		}
	}
	return false
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}
