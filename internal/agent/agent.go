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
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/klever-coex/clover2-cli/internal/cliapp"
)

const (
	defaultListen = "127.0.0.1:7550"
	authPrefix    = "Bearer "
)

// defaultTokenPath picks a per-user location so the systemd template unit
// (clover2-agent@<user>) works without root-owned file conflicts.
func defaultTokenPath() string {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "clover2", "agent.token")
	}
	return "/etc/clover2/agent.token"
}

var started = time.Now()

func Command() *cobra.Command {
	group := &cobra.Command{Use: "agent", Short: "On-robot agent (localhost HTTP API)"}

	group.AddCommand(runCmd())

	return group
}

func runCmd() *cobra.Command {
	var (
		listen    string
		tokenFile string
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the agent daemon (systemd unit: packaging/clover2-agent.service)",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := ensureToken(tokenFile)
			if err != nil {
				return err
			}

			slog.Info("agent token ready", "file", tokenFile)

			if _, _, err := net.SplitHostPort(listen); err != nil {
				return fmt.Errorf("bad --listen '%s': %w", listen, err)
			}

			srv := &http.Server{
				Addr:              listen,
				Handler:           mux(token),
				ReadHeaderTimeout: 5 * time.Second,
			}

			ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

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
		},
	}

	cmd.Flags().StringVar(&listen, "listen", defaultListen, "listen address (localhost by default; remote access goes through SSH tunnels)")
	cmd.Flags().StringVar(&tokenFile, "token-file", defaultTokenPath(), "bearer token file (generated 0600 if missing)")

	return cmd
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

// ensureToken reads the token file or creates it (32 random bytes, 0600).
func ensureToken(path string) ([]byte, error) {
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
