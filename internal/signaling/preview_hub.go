package signaling

import (
	"context"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/markus/spidercam/internal/preview"
	"github.com/markus/spidercam/internal/protocol"
)

type PreviewHub struct {
	stream *preview.Stream

	mu      sync.Mutex
	clients map[*previewClient]struct{}
}

type previewClient struct {
	conn *websocket.Conn
}

func NewPreviewHub(stream *preview.Stream) *PreviewHub {
	return &PreviewHub{
		stream:  stream,
		clients: make(map[*previewClient]struct{}),
	}
}

func (h *PreviewHub) HandleUpgrade(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	client := &previewClient{conn: conn}
	h.addClient(client)
	h.stream.ForceKeyframe()
	_ = client.sendJSON(h.stream.InitMessage())

	go func() {
		defer func() {
			h.removeClient(client)
			_ = conn.Close()
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
}

func (h *PreviewHub) PublishLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case chunk, ok := <-h.stream.Chunks():
			if !ok {
				return
			}
			h.broadcastBinary(chunk)
		}
	}
}

func (h *PreviewHub) BroadcastCut(msg protocol.PreviewCutMsg) {
	h.broadcastJSON(msg)
}

func (h *PreviewHub) addClient(c *previewClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c] = struct{}{}
}

func (h *PreviewHub) removeClient(c *previewClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, c)
}

func (h *PreviewHub) broadcastBinary(data []byte) {
	h.mu.Lock()
	clients := make([]*previewClient, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.Unlock()
	for _, c := range clients {
		_ = c.sendBinary(data)
	}
}

func (h *PreviewHub) broadcastJSON(v any) {
	h.mu.Lock()
	clients := make([]*previewClient, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.Unlock()
	for _, c := range clients {
		_ = c.sendJSON(v)
	}
}

func (c *previewClient) sendJSON(v any) error {
	return c.conn.WriteJSON(v)
}

func (c *previewClient) sendBinary(data []byte) error {
	return c.conn.WriteMessage(websocket.BinaryMessage, data)
}
