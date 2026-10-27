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

	"github.com/klever-coex/clover2-cli/internal/agent/docker"
	"github.com/klever-coex/clover2-cli/internal/agent/system"
	"github.com/klever-coex/clover2-cli/internal/agentapi"
	"github.com/klever-coex/clover2-cli/internal/cliapp"

	"github.com/docker/docker/client"
)

const (
	DefaultListen    = "127.0.0.1:7550"
	DefaultTokenFile = "/etc/clover2/agent.token"
	shutdownGrace    = 5 * time.Second
)

type Options struct {
	Token   []byte
	Version string
	Log     *slog.Logger
	Started time.Time
}

type server struct {
	opts          Options
	dockerHandler *docker.Handler
	system        *system.Handler
	caps          *capabilitySet
}

func New(opts Options, cli *client.Client) *server {
	if opts.Log == nil {
		opts.Log = slog.Default()
	}

	if opts.Started.IsZero() {
		opts.Started = time.Now()
	}

	s := &server{
		opts:          opts,
		dockerHandler: docker.NewHandler(cli),
	}
	s.caps = newCapabilitySet(s.dockerHandler.IsHealthy)
	s.system = system.NewHandler(opts.Version, opts.Started, s.caps.snapshot)

	return s
}

func (s *server) Handler() http.Handler {
	m := http.NewServeMux()
	s.system.Register(m)
	s.dockerHandler.Register(m)

	var h http.Handler = m
	if len(s.opts.Token) > 0 {
		h = s.withAuth(h)
	}

	h = s.withRecovery(h)

	return s.withLogging(h)
}

func Serve(ctx context.Context, listen, tokenFile string) error {
	var token []byte
	if tokenFile == "" {
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

	dockerCli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		slog.Warn("docker client init failed; docker endpoints will report 503", "err", err)
	}

	agentServer := New(Options{
		Token:   token,
		Version: cliapp.Version,
	}, dockerCli)

	srv := &http.Server{
		Addr:              listen,
		Handler:           agentServer.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}

	slog.Info("agent listening", "addr", listen, "api", "v"+strconv.Itoa(agentapi.APIVersion), "auth", len(token) > 0)

	go agentServer.caps.run(ctx)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Warn("graceful shutdown failed", "err", err)
		}
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

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
