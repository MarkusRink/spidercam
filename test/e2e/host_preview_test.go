//go:build e2e

package e2e_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/markus/spidercam/internal/protocol"
)

const previewFlagKeyframe byte = 0x01

func TestPreviewStreamInitAndKeyframe(t *testing.T) {
	d := StartMockDaemon(t)
	conn, _, err := websocket.DefaultDialer.Dial(d.HostPreviewWS(), nil)
	if err != nil {
		t.Fatalf("dial preview ws: %v", err)
	}
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read preview-stream-init: %v", err)
	}
	var init protocol.PreviewStreamInitMsg
	if err := json.Unmarshal(data, &init); err != nil {
		t.Fatalf("decode init: %v", err)
	}
	if init.Type != "preview-stream-init" {
		t.Fatalf("type = %q, want preview-stream-init", init.Type)
	}
	if init.Codec == "" || init.Width <= 0 || init.Height <= 0 || init.FPS <= 0 {
		t.Fatalf("invalid init: %+v", init)
	}

	chunk, err := readPreviewKeyframeChunk(t, conn, 15*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunk) < 13 {
		t.Fatalf("chunk too short: %d bytes", len(chunk))
	}
	if chunk[0]&previewFlagKeyframe == 0 {
		t.Fatalf("flags %#x missing keyframe bit", chunk[0])
	}
}

func readPreviewKeyframeChunk(t *testing.T, conn *websocket.Conn, timeout time.Duration) ([]byte, error) {
	t.Helper()
	type result struct {
		data []byte
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		for {
			mt, data, err := conn.ReadMessage()
			if err != nil {
				ch <- result{err: err}
				return
			}
			if mt == websocket.BinaryMessage && len(data) > 0 && data[0]&previewFlagKeyframe != 0 {
				ch <- result{data: data}
				return
			}
		}
	}()
	select {
	case res := <-ch:
		return res.data, res.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("timed out waiting for keyframe chunk after %s", timeout)
	}
}
