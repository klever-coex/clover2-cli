// Package agent implements the on-robot agent: an HTTP API used by the CLI
// and tooling to query and operate the machine.
package agent

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/klever-coex/clover2-cli/internal/cliapp"
)

const (
	DefaultListen = "127.0.0.1:7550"
	authPrefix    = "Bearer "
)

// DefaultTokenPath picks a per-user location so a per-user systemd template
// unit (clover2-agent@<user>) works without root-owned file conflicts.
func DefaultTokenPath() string {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "clover2", "agent.token")
	}
	return "/etc/clover2/agent.token"
}

var started = time.Now()

// Serve runs the agent HTTP server until ctx is cancelled: it loads (or
// creates) the bearer token from tokenFile, then blocks serving requests.
func Serve(ctx context.Context, listen, tokenFile string) error {
	token, err := EnsureToken(tokenFile)
	if err != nil {
		return err
	}

	slog.Info("agent token ready", "file", tokenFile)

	if _, _, err := net.SplitHostPort(listen); err != nil {
		return fmt.Errorf("bad listen address '%s': %w", listen, err)
	}

	srv := &http.Server{
		Addr:              listen,
		Handler:           mux(token),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	slog.Info("agent listening", "addr", srv.Addr, "api", "v1")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

func mux(token []byte) http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /api/v1/health", withAuth(token, handleHealth))

	return m
}

func withAuth(token []byte, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, authPrefix) ||
			subtle.ConstantTimeCompare([]byte(strings.TrimPrefix(auth, authPrefix)), token) != 1 {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		next(w, r)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	hostname, _ := os.Hostname()
	writeJSON(w, http.StatusOK, map[string]any{
		"agent_version": cliapp.Version,
		"robot_name":    hostname,
		"uptime_s":      int(time.Since(started).Seconds()),
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// EnsureToken reads the token file or creates it (32 random bytes, 0600).
func EnsureToken(path string) ([]byte, error) {
	if token, err := os.ReadFile(path); err == nil && len(token) > 0 {
		return []byte(token), nil
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, err
	}

	token := []byte(hex.EncodeToString(raw))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}

	if err := os.WriteFile(path, token, 0o600); err != nil {
		return nil, err
	}

	return token, nil
}
