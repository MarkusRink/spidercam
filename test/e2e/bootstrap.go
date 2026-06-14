//go:build e2e

package e2e_test

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"
)

var (
	daemonBuildOnce sync.Once
	daemonBinary    string
	daemonBuildErr  error
)

type Daemon struct {
	ParticipantPort int
	HostPort        int
	cmd             *exec.Cmd
}

func (d *Daemon) ParticipantBase() string {
	return fmt.Sprintf("http://127.0.0.1:%d", d.ParticipantPort)
}

func (d *Daemon) HostBase() string {
	return fmt.Sprintf("http://127.0.0.1:%d", d.HostPort)
}

func (d *Daemon) ParticipantWS() string {
	return fmt.Sprintf("ws://127.0.0.1:%d/api/v1/ws", d.ParticipantPort)
}

func (d *Daemon) HostWS() string {
	return fmt.Sprintf("ws://127.0.0.1:%d/api/v1/ws", d.HostPort)
}

func (d *Daemon) HostPreviewWS() string {
	return fmt.Sprintf("ws://127.0.0.1:%d/api/v1/ws/preview", d.HostPort)
}

func StartMockDaemon(t *testing.T) *Daemon {
	t.Helper()

	participantPort := portFromEnv(t, "SPIDERCAM_E2E_PARTICIPANT_PORT", freePort(t))
	hostPort := portFromEnv(t, "SPIDERCAM_E2E_HOST_PORT", freePort(t))

	binary, err := daemonExecutable()
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(
		binary,
		"--mock",
		"--no-open-browser",
		"--participant-addr", fmt.Sprintf("127.0.0.1:%d", participantPort),
		"--host-addr", fmt.Sprintf("127.0.0.1:%d", hostPort),
	)
	cmd.Dir = repoRoot()
	if err := cmd.Start(); err != nil {
		t.Fatalf("start spidercamd: %v", err)
	}

	d := &Daemon{
		ParticipantPort: participantPort,
		HostPort:        hostPort,
		cmd:             cmd,
	}
	t.Cleanup(func() {
		_ = d.cmd.Process.Kill()
		_ = d.cmd.Wait()
	})

	waitHealthy(t, d.ParticipantBase()+"/api/health")
	waitHealthy(t, d.HostBase()+"/api/health")
	return d
}

func daemonExecutable() (string, error) {
	daemonBuildOnce.Do(func() {
		dir, err := os.MkdirTemp("", "spidercam-e2e-*")
		if err != nil {
			daemonBuildErr = err
			return
		}
		daemonBinary = filepath.Join(dir, "spidercamd")
		build := exec.Command("go", "build", "-o", daemonBinary, "./cmd/spidercamd")
		build.Dir = repoRoot()
		if out, err := build.CombinedOutput(); err != nil {
			daemonBuildErr = fmt.Errorf("build spidercamd: %w\n%s", err, out)
		}
	})
	if daemonBuildErr != nil {
		return "", daemonBuildErr
	}
	return daemonBinary, nil
}

func portFromEnv(t *testing.T, key string, fallback int) int {
	t.Helper()
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	port, err := strconv.Atoi(raw)
	if err != nil || port <= 0 || port > 65535 {
		t.Fatalf("%s=%q is not a valid port", key, raw)
	}
	return port
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func waitHealthy(t *testing.T, url string) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("daemon not healthy on %s", url)
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	return repoRoot()
}

func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
}
