package signaling

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/markus/spidercam/internal/fixtures"
	"github.com/markus/spidercam/internal/protocol"
	"github.com/markus/spidercam/internal/room"
)

type HostHub struct {
	room      *room.Room
	processor StreamProcessor

	mu      sync.Mutex
	clients map[*wsClient]struct{}
}

func NewHostHub(r *room.Room) *HostHub {
	return &HostHub{
		room:    r,
		clients: make(map[*wsClient]struct{}),
	}
}

func (h *HostHub) SetStreamProcessor(p StreamProcessor) {
	h.processor = p
}

func (h *HostHub) HandleUpgrade(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	client := &wsClient{conn: conn}
	h.mu.Lock()
	h.clients[client] = struct{}{}
	h.mu.Unlock()

	go func() {
		defer func() {
			h.mu.Lock()
			delete(h.clients, client)
			h.mu.Unlock()
			_ = conn.Close()
		}()
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			h.handleMessage(client, data)
		}
	}()
}

func (h *HostHub) BroadcastState(state protocol.RoomState) {
	msg := protocol.HostStateMsg{Type: "host-state", State: state}
	payload, err := json.Marshal(msg)
	if err != nil {
		return
	}
	h.mu.Lock()
	clients := make([]*wsClient, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.Unlock()
	for _, c := range clients {
		c.mu.Lock()
		_ = c.conn.WriteMessage(websocket.TextMessage, payload)
		c.mu.Unlock()
	}
}

func (h *HostHub) handleMessage(client *wsClient, data []byte) {
	var envelope struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return
	}

	switch envelope.Type {
	case "config":
		var msg protocol.ConfigMsg
		if err := json.Unmarshal(data, &msg); err != nil {
			return
		}
		_ = h.room.UpdateConfig(msg.Config)
	case "list-capture-devices":
		devices, err := fixtures.LoadCaptureDevices()
		if err != nil {
			return
		}
		_ = client.sendJSON(protocol.CaptureDevicesMsg{
			Type:    "capture-devices",
			Devices: devices,
		})
	case "set-capture-devices":
		var msg protocol.SetCaptureDevicesMsg
		if err := json.Unmarshal(data, &msg); err != nil {
			return
		}
		h.handleSetCaptureDevices(client, msg.Selection)
	case "set-stream-processing":
		var msg protocol.SetStreamProcessingMsg
		if err := json.Unmarshal(data, &msg); err != nil {
			return
		}
		applyStreamProcessing(h.room, h.processor, msg.ParticipantID, msg.Flags)
	case "copy-participant-url":
		_ = client.sendJSON(protocol.ParticipantURLMsg{
			Type: "participant-url",
			URL:  h.room.State().ParticipantURL,
		})
	}
}

func (h *HostHub) handleSetCaptureDevices(client *wsClient, selection protocol.CaptureSelection) {
	devices, err := fixtures.LoadCaptureDevices()
	if err != nil {
		return
	}
	mic, camera, sink, ok := resolveCaptureSelection(devices, selection)
	if !ok {
		_ = client.sendJSON(protocol.CaptureDevicesUpdatedMsg{
			Type:    "capture-devices-updated",
			Capture: h.room.State().Capture,
			Error:   "unknown device id",
		})
		return
	}
	h.room.SetCaptureSelection(selection.MicID, selection.CameraID, selection.SinkID)
	h.room.SetCaptureLabels(mic.Label, camera.Label, sink.Label)
	_ = client.sendJSON(protocol.CaptureDevicesUpdatedMsg{
		Type:    "capture-devices-updated",
		Capture: h.room.State().Capture,
	})
}

func resolveCaptureSelection(devices protocol.CaptureDevices, sel protocol.CaptureSelection) (mic, camera, sink protocol.DeviceInfo, ok bool) {
	for _, d := range devices.Mics {
		if d.ID == sel.MicID {
			mic = d
			break
		}
	}
	for _, d := range devices.Cameras {
		if d.ID == sel.CameraID {
			camera = d
			break
		}
	}
	for _, d := range devices.Sinks {
		if d.ID == sel.SinkID {
			sink = d
			break
		}
	}
	ok = mic.ID != "" && camera.ID != "" && sink.ID != ""
	return
}
