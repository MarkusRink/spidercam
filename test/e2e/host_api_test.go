//go:build e2e

package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/markus/spidercam/internal/protocol"
)

func TestHealth(t *testing.T) {
	d := StartMockDaemon(t)
	for _, base := range []string{d.ParticipantBase(), d.HostBase()} {
		resp, err := http.Get(base + "/api/health")
		if err != nil {
			t.Fatalf("GET %s/api/health: %v", base, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
		var body map[string]bool
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("decode health: %v", err)
		}
		if !body["ok"] {
			t.Fatalf("health ok = false")
		}
	}
}

func TestHostStateREST(t *testing.T) {
	d := StartMockDaemon(t)
	resp, err := http.Get(d.HostBase() + "/api/v1/host/state")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var state protocol.RoomState
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		t.Fatalf("decode state: %v", err)
	}
	if len(state.Metrics) == 0 {
		t.Fatal("expected metrics in host state")
	}
	if state.Reference.StreamID == "" {
		t.Fatal("expected reference stream id")
	}
}

func TestHostStateWS(t *testing.T) {
	d := StartMockDaemon(t)
	conn, _, err := websocket.DefaultDialer.Dial(d.HostWS(), nil)
	if err != nil {
		t.Fatalf("dial host ws: %v", err)
	}
	defer conn.Close()

	msg, err := readHostState(t, conn, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if len(msg.State.Metrics) == 0 {
		t.Fatal("expected metrics in host-state")
	}
}

func TestHostConfigREST(t *testing.T) {
	d := StartMockDaemon(t)
	hold := 450
	body, _ := json.Marshal(protocol.HostConfigPatch{VideoHoldMs: &hold})
	resp, err := http.Post(d.HostBase()+"/api/v1/host/config", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var result map[string]bool
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if !result["ok"] {
		t.Fatal("config post not ok")
	}
}

func TestHostConfigWS(t *testing.T) {
	d := StartMockDaemon(t)
	conn, _, err := websocket.DefaultDialer.Dial(d.HostWS(), nil)
	if err != nil {
		t.Fatalf("dial host ws: %v", err)
	}
	defer conn.Close()

	hold := 500
	if err := conn.WriteJSON(protocol.ConfigMsg{
		Type:   "config",
		Config: protocol.HostConfigPatch{VideoHoldMs: &hold},
	}); err != nil {
		t.Fatalf("send config: %v", err)
	}
	_, err = readHostState(t, conn, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCaptureDevicesREST(t *testing.T) {
	d := StartMockDaemon(t)
	resp, err := http.Get(d.HostBase() + "/api/v1/capture/devices")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var devices protocol.CaptureDevices
	if err := json.NewDecoder(resp.Body).Decode(&devices); err != nil {
		t.Fatal(err)
	}
	if len(devices.Mics) == 0 || len(devices.Cameras) == 0 || len(devices.Sinks) == 0 {
		t.Fatalf("incomplete capture devices: %+v", devices)
	}
}

func TestCaptureSelectionREST(t *testing.T) {
	d := StartMockDaemon(t)
	sel := protocol.CaptureSelection{
		MicID:    "pw:source:0",
		CameraID: "v4l2:/dev/video0",
		SinkID:   "pw:sink:0",
	}
	body, _ := json.Marshal(sel)
	resp, err := http.Post(d.HostBase()+"/api/v1/capture/selection", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var capture protocol.CaptureState
	if err := json.NewDecoder(resp.Body).Decode(&capture); err != nil {
		t.Fatal(err)
	}
	if capture.MicID != sel.MicID || capture.CameraID != sel.CameraID || capture.SinkID != sel.SinkID {
		t.Fatalf("capture = %+v, want selection applied", capture)
	}
}

func TestCaptureDevicesWS(t *testing.T) {
	d := StartMockDaemon(t)
	conn, _, err := websocket.DefaultDialer.Dial(d.HostWS(), nil)
	if err != nil {
		t.Fatalf("dial host ws: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteJSON(protocol.ListCaptureDevicesMsg{Type: "list-capture-devices"}); err != nil {
		t.Fatalf("send list-capture-devices: %v", err)
	}

	var devicesMsg protocol.CaptureDevicesMsg
	if err := readJSONUntil(t, conn, 5*time.Second, func(raw []byte) bool {
		var envelope struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return false
		}
		if envelope.Type != "capture-devices" {
			return false
		}
		return json.Unmarshal(raw, &devicesMsg) == nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(devicesMsg.Devices.Mics) == 0 {
		t.Fatal("expected mics in capture-devices response")
	}

	if err := conn.WriteJSON(protocol.SetCaptureDevicesMsg{
		Type: "set-capture-devices",
		Selection: protocol.CaptureSelection{
			MicID:    "pw:source:1",
			CameraID: "v4l2:/dev/video2",
			SinkID:   "pw:sink:1",
		},
	}); err != nil {
		t.Fatalf("send set-capture-devices: %v", err)
	}

	var updated protocol.CaptureDevicesUpdatedMsg
	if err := readJSONUntil(t, conn, 5*time.Second, func(raw []byte) bool {
		var envelope struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return false
		}
		if envelope.Type != "capture-devices-updated" {
			return false
		}
		return json.Unmarshal(raw, &updated) == nil
	}); err != nil {
		t.Fatal(err)
	}
	if updated.Capture.MicID != "pw:source:1" {
		t.Fatalf("capture mic = %q", updated.Capture.MicID)
	}
}

func TestStreamProcessingREST(t *testing.T) {
	d := StartMockDaemon(t)
	participantID := "a3f7c2e1-0000-4000-8000-000000000001"
	body, _ := json.Marshal(protocol.SetStreamProcessingMsg{
		Type:          "set-stream-processing",
		ParticipantID: participantID,
		Flags: protocol.StreamProcessingFlags{
			AecEnabled:     true,
			DenoiseEnabled: true,
		},
	})
	resp, err := http.Post(d.HostBase()+"/api/v1/host/stream-processing", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d body = %s", resp.StatusCode, data)
	}
}

func readHostState(t *testing.T, conn *websocket.Conn, timeout time.Duration) (protocol.HostStateMsg, error) {
	t.Helper()
	var msg protocol.HostStateMsg
	err := readJSONUntil(t, conn, timeout, func(raw []byte) bool {
		var envelope struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return false
		}
		if envelope.Type != "host-state" {
			return false
		}
		return json.Unmarshal(raw, &msg) == nil
	})
	return msg, err
}

func readJSONUntil(t *testing.T, conn *websocket.Conn, timeout time.Duration, match func([]byte) bool) error {
	t.Helper()
	type result struct {
		err error
	}
	ch := make(chan result, 1)
	go func() {
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				ch <- result{err: err}
				return
			}
			if match(data) {
				ch <- result{}
				return
			}
		}
	}()
	select {
	case res := <-ch:
		if res.err != nil {
			return res.err
		}
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timed out after %s", timeout)
	}
}
