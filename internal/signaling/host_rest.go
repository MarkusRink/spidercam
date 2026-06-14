package signaling

import (
	"net/http"

	"github.com/markus/spidercam/internal/fixtures"
	"github.com/markus/spidercam/internal/protocol"
	"github.com/markus/spidercam/internal/room"
)

func RegisterHostREST(mux *http.ServeMux, r *room.Room, proc StreamProcessor) {
	mux.HandleFunc("GET /api/v1/host/state", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, r.State())
	})

	mux.HandleFunc("POST /api/v1/host/config", func(w http.ResponseWriter, req *http.Request) {
		patch, ok := readJSONBody[protocol.HostConfigPatch](req)
		if !ok {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid json"})
			return
		}
		if err := r.UpdateConfig(patch); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	})

	mux.HandleFunc("GET /api/v1/capture/devices", func(w http.ResponseWriter, _ *http.Request) {
		devices, err := fixtures.LoadCaptureDevices()
		if err != nil {
			http.Error(w, "fixtures unavailable", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, devices)
	})

	mux.HandleFunc("POST /api/v1/capture/selection", func(w http.ResponseWriter, req *http.Request) {
		selection, ok := readJSONBody[protocol.CaptureSelection](req)
		if !ok {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
			return
		}
		devices, err := fixtures.LoadCaptureDevices()
		if err != nil {
			http.Error(w, "fixtures unavailable", http.StatusInternalServerError)
			return
		}
		mic, camera, sink, valid := resolveCaptureSelection(devices, selection)
		if !valid {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "unknown device id"})
			return
		}
		r.SetCaptureSelection(selection.MicID, selection.CameraID, selection.SinkID)
		r.SetCaptureLabels(mic.Label, camera.Label, sink.Label)
		writeJSON(w, http.StatusOK, r.State().Capture)
	})

	mux.HandleFunc("POST /api/v1/host/stream-processing", func(w http.ResponseWriter, req *http.Request) {
		body, ok := readJSONBody[protocol.SetStreamProcessingMsg](req)
		if !ok || body.ParticipantID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid json"})
			return
		}
		if !applyStreamProcessing(r, proc, body.ParticipantID, body.Flags) {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "unknown participant"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	})
}
