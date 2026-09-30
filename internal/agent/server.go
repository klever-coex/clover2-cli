// Package agent implements the on-robot agent: an HTTP API used by the CLI
// and tooling to query and operate the machine. The wire contract lives in
// internal/agentapi and is shared with the CLI client.
package agent

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/klever-coex/clover2-cli/internal/agentapi"
	"github.com/klever-coex/clover2-cli/internal/cliapp"
)

const (
	// DefaultListen is the default listen address. Remote access (LAN) is a
	// deliberate choice: the systemd unit installed by "agent install"
	// passes --listen 0.0.0.0:7550 explicitly.
	DefaultListen = "127.0.0.1:7550"
	// DefaultTokenFile is the token location used by the systemd unit
	// installed via "clover2 agent install".
	DefaultTokenFile = "/etc/clover2/agent.token"
	shutdownGrace    = 5 * time.Second
)

// Options configures an agent server.
type Options struct {
	// Token enables bearer auth when non-empty. An empty token serves
	// WITHOUT auth — the caller must warn loudly (Serve does).
	Token []byte
	// Version is reported by health/info payloads.
	Version string
	// Log receives request and lifecycle logs; nil means slog.Default().
	Log *slog.Logger
	// Started overrides the process start time (tests).
	Started time.Time
}

type server struct {
	opts Options
}

// New builds an agent server.
func New(opts Options) *server {
	if opts.Log == nil {
		opts.Log = slog.Default()
	}

	if opts.Started.IsZero() {
		opts.Started = time.Now()
	}

	return &server{opts: opts}
}

// Handler builds the full HTTP handler: request logging and panic recovery
// around the whole mux; auth wraps the mux only when a token is configured.
func (s *server) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET "+agentapi.HealthRoute, s.handleHealth)
	m.HandleFunc("GET "+agentapi.InfoRoute, s.handleInfo)
	m.HandleFunc("POST "+agentapi.ExecRoute, s.handleExec)

	var h http.Handler = m
	if len(s.opts.Token) > 0 {
		h = s.withAuth(h)
	}

	h = s.withRecovery(h)

	return s.withLogging(h)
}

// Serve is the CLI entry point: it resolves the token file (empty means
// serve without auth), announces the agent over mDNS and blocks until ctx is
// cancelled or the listener fails.
func Serve(ctx context.Context, listen, tokenFile string) error {
	var token []byte
	if tokenFile == "" {
		// Always visible regardless of the CLI log level: a security
		// warning nobody sees is an open API.
		fmt.Fprintln(os.Stderr, "warning: running WITHOUT auth: anyone on the network can execute commands via this agent; pass --token-file to enable")
	} else {
		t, err := EnsureToken(tokenFile)
		if err != nil {
			return err
		}

		token = t
		slog.Info("agent token ready", "file", tokenFile)
	}

	if _, _, err := net.SplitHostPort(listen); err != nil {
		return fmt.Errorf("bad listen address '%s': %w", listen, err)
	}

	srv := &http.Server{
		Addr:              listen,
		Handler:           New(Options{Token: token, Version: cliapp.Version}).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		// Cancelling ctx (SIGTERM) cancels every in-flight request, which
		// structurally aborts in-flight exec children.
		BaseContext: func(net.Listener) context.Context { return ctx },
	}

	// Best-effort mDNS announce; skip when the address carries no port.
	if _, portStr, err := net.SplitHostPort(listen); err == nil {
		if port, err := strconv.Atoi(portStr); err == nil && port > 0 {
			announceMDNS(ctx, port, map[string]string{"version": cliapp.Version})
		}
	}

	slog.Info("agent listening", "addr", listen, "api", "v"+strconv.Itoa(agentapi.APIVersion), "auth", len(token) > 0)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

// hostname is the robot name reported by health/info.
func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}

	return h
}

// EnsureToken reads the token file or creates it (32 random bytes hex, 0600).
// An existing but empty or unreadable file is a fatal error: silently
// regenerating would lock out every client holding the old token.
func EnsureToken(path string) ([]byte, error) {
	if token, err := os.ReadFile(path); err == nil {
		if len(token) == 0 {
			return nil, fmt.Errorf("token file %s is empty; remove it and restart to generate a new one", path)
		}

		return token, nil
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("token file %s is not readable: %w", path, err)
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, err
	}

	token := []byte(hex.EncodeToString(raw))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	if err := os.WriteFile(path, token, 0o600); err != nil {
		return nil, err
	}

	return token, nil
}
