package signaling

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/markus/spidercam/internal/fixtures"
	"github.com/markus/spidercam/internal/preview"
	"github.com/markus/spidercam/internal/protocol"
	"github.com/markus/spidercam/internal/room"
	"github.com/markus/spidercam/internal/scenario"
)

func TestHostStateRESTHasParticipants(t *testing.T) {
	rm := room.New("http://127.0.0.1:1234/")
	state, err := fixtures.LoadRoutingState()
	if err != nil {
		t.Fatalf("load routing fixture: %v", err)
	}
	rm.SetState(state)

	mux := http.NewServeMux()
	RegisterHostREST(mux, rm, nil, true, nil)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/host/state")
	if err != nil {
		t.Fatalf("GET host state: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got protocol.RoomState
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Participants) == 0 {
		t.Fatal("expected participants in host state")
	}
}

func TestHostWebSocketReceivesHostState(t *testing.T) {
	rm := room.New("http://127.0.0.1:1234/")
	state, err := fixtures.LoadRoutingState()
	if err != nil {
		t.Fatalf("load routing fixture: %v", err)
	}
	rm.SetState(state)

	engine := scenario.New(rm)
	hostHub := NewHostHub(rm, true)
	engine.OnState(hostHub.BroadcastState)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	engine.Start(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/ws", hostHub.HandleUpgrade)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/v1/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial ws: %v", err)
	}
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var msg protocol.HostStateMsg
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("read host-state: %v", err)
	}
	if msg.Type != "host-state" {
		t.Fatalf("type = %q, want host-state", msg.Type)
	}
	if len(msg.State.Participants) == 0 {
		t.Fatal("expected participants in ws host-state")
	}
}

func TestPreviewWebSocketInit(t *testing.T) {
	keyframe, err := fixtures.LoadPreviewKeyframe()
	if err != nil {
		t.Fatalf("load keyframe: %v", err)
	}
	stream, err := preview.New(preview.Config{
		Width:        preview.DefaultWidth,
		Height:       preview.DefaultHeight,
		FPS:          preview.DefaultFPS,
		Mock:         true,
		MockKeyframe: keyframe,
	})
	if err != nil {
		t.Fatalf("preview stream: %v", err)
	}
	defer stream.Close()
	previewHub := NewPreviewHub(stream)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go previewHub.PublishLoop(ctx)
	preview.RunMockCompositor(ctx, stream, room.New("http://127.0.0.1:1234/"), nil)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/ws/preview", previewHub.HandleUpgrade)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/v1/ws/preview"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial preview ws: %v", err)
	}
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var init protocol.PreviewStreamInitMsg
	if err := conn.ReadJSON(&init); err != nil {
		t.Fatalf("read preview-stream-init: %v", err)
	}
	if init.Type != "preview-stream-init" {
		t.Fatalf("type = %q, want preview-stream-init", init.Type)
	}
	if init.Codec != preview.DefaultCodec {
		t.Fatalf("codec = %q, want %q", init.Codec, preview.DefaultCodec)
	}
}
