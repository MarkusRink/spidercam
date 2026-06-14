package daemon

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/markus/spidercam/internal/signaling"
)

func TestHostSPAServesRootHTTP(t *testing.T) {
	h := signaling.NewHostMux(hostFS, hostUIRoot, nil, signaling.HostServices{})
	srv := httptest.NewServer(h)
	defer srv.Close()

	for _, path := range []string{"/", "/index.html", "/missing-route"} {
		t.Run(path, func(t *testing.T) {
			resp, err := http.Get(srv.URL + path)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("GET %s: status=%d location=%q", path, resp.StatusCode, resp.Header.Get("Location"))
			}
		})
	}

	resp, err := http.Get(srv.URL + "/api/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/health: status=%d", resp.StatusCode)
	}
}
