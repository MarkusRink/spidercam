package signaling

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/markus/spidercam/internal/room"
)

type HostServices struct {
	Hub             *HostHub
	PreviewHub      *PreviewHub
	StreamProcessor StreamProcessor
}

type ParticipantServices struct {
	Hub *ParticipantHub
}

func NewHostMux(static fs.FS, uiRoot string, r *room.Room, svc HostServices) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", handleHealth)
	RegisterHostREST(mux, r, svc.StreamProcessor)
	mux.HandleFunc("GET /api/v1/ws", svc.Hub.HandleUpgrade)
	mux.HandleFunc("GET /api/v1/ws/preview", svc.PreviewHub.HandleUpgrade)
	mux.Handle("/{path...}", spaHandler(static, uiRoot))
	return mux
}

func NewParticipantMux(static fs.FS, uiRoot string, svc ParticipantServices) http.Handler {
	mux := http.NewServeMux()
	RegisterParticipantREST(mux)
	mux.HandleFunc("GET /api/v1/ws", svc.Hub.HandleUpgrade)
	mux.Handle("/{path...}", spaHandler(static, uiRoot))
	return mux
}

func spaHandler(root fs.FS, uiRoot string) http.Handler {
	sub, err := fs.Sub(root, uiRoot)
	if err != nil {
		sub = root
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		name := strings.TrimPrefix(path.Clean("/"+strings.TrimPrefix(r.URL.Path, "/")), "/")
		if name == "" || name == "." {
			http.ServeFileFS(w, r, sub, "index.html")
			return
		}
		if f, err := sub.Open(name); err == nil {
			info, statErr := f.Stat()
			_ = f.Close()
			if statErr == nil && !info.IsDir() {
				http.ServeFileFS(w, r, sub, name)
				return
			}
		}
		http.ServeFileFS(w, r, sub, "index.html")
	})
}
