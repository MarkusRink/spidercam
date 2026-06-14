package signaling

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/markus/spidercam/internal/protocol"
	"github.com/markus/spidercam/internal/room"
)

type HostHub struct {
	room              *room.Room
	processor         StreamProcessor
	useFixtureDevices bool

	mu      sync.Mutex
	clients map[*wsClient]struct{}
}

func NewHostHub(r *room.Room, useFixtureDevices bool) *HostHub {
	return &HostHub{
		room:              r,
		useFixtureDevices: useFixtureDevices,
		clients:           make(map[*wsClient]struct{}),
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
		devices, err := ListCaptureDevices(h.useFixtureDevices)
		if err != nil {
			return
		}
		if capture, changed, err := EnsureDefaultCapture(h.room, devices); err == nil && changed {
			_ = client.sendJSON(protocol.CaptureDevicesUpdatedMsg{
				Type:    "capture-devices-updated",
				Capture: capture,
			})
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
	devices, err := ListCaptureDevices(h.useFixtureDevices)
	if err != nil {
		return
	}
	capture, err := ApplyCaptureSelection(h.room, devices, selection)
	if err != nil {
		_ = client.sendJSON(protocol.CaptureDevicesUpdatedMsg{
			Type:    "capture-devices-updated",
			Capture: h.room.State().Capture,
			Error:   err.Error(),
		})
		return
	}
	_ = client.sendJSON(protocol.CaptureDevicesUpdatedMsg{
		Type:    "capture-devices-updated",
		Capture: capture,
	})
}
