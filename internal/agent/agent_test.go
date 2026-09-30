package agent

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klever-coex/clover2-cli/internal/agentapi"
)

func newTestServer(t *testing.T, token string) *httptest.Server {
	t.Helper()

	opts := Options{Version: "test"}
	if token != "" {
		opts.Token = []byte(token)
	}

	srv := httptest.NewServer(New(opts).Handler())
	t.Cleanup(srv.Close)

	return srv
}

func TestHealthAuth(t *testing.T) {
	srv := newTestServer(t, "secret-token")

	// no token -> 401 envelope
	resp, err := http.Get(srv.URL + agentapi.HealthRoute)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("no token: status = %d, want 401", resp.StatusCode)
	}

	// wrong token -> 401
	req, _ := http.NewRequest(http.MethodGet, srv.URL+agentapi.HealthRoute, nil)
	req.Header.Set("Authorization", "Bearer nope")
	if resp, err = http.DefaultClient.Do(req); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("wrong token: status = %d, want 401", resp.StatusCode)
	}

	// correct token -> 200 with the health payload
	req.Header.Set("Authorization", "Bearer secret-token")
	if resp, err = http.DefaultClient.Do(req); err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("ok token: status = %d, want 200", resp.StatusCode)
	}

	var payload agentapi.Health
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.APIVersion != agentapi.APIVersion {
		t.Errorf("api_version = %d, want %d", payload.APIVersion, agentapi.APIVersion)
	}
	if payload.AgentVersion != "test" || payload.RobotName == "" {
		t.Errorf("health payload = %+v", payload)
	}
}

func TestHealthTokenlessServesWithoutAuth(t *testing.T) {
	srv := newTestServer(t, "")

	resp, err := http.Get(srv.URL + agentapi.HealthRoute)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("tokenless: status = %d, want 200", resp.StatusCode)
	}
}

func TestInfoPayload(t *testing.T) {
	srv := newTestServer(t, "")

	req, _ := http.NewRequest(http.MethodGet, srv.URL+agentapi.InfoRoute, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("info: status = %d, want 200", resp.StatusCode)
	}

	var payload agentapi.Info
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}

	if payload.APIVersion != agentapi.APIVersion {
		t.Errorf("api_version = %d", payload.APIVersion)
	}
	if payload.AgentVersion != "test" || payload.Hostname == "" {
		t.Errorf("info = %+v", payload)
	}
	if payload.OS == "" || payload.Arch == "" {
		t.Errorf("info os/arch empty: %+v", payload)
	}
	if payload.Mem.TotalBytes == 0 {
		// /proc/meminfo must exist on linux CI runners; zero means the
		// parser broke.
		t.Errorf("mem.total_bytes = 0; meminfo parser broken?")
	}
}

func TestExecHandler(t *testing.T) {
	srv := newTestServer(t, "")

	post := func(t *testing.T, body any) (*http.Response, agentapi.ExecResult) {
		t.Helper()
		raw, _ := json.Marshal(body)
		resp, err := http.Post(srv.URL+agentapi.ExecRoute, "application/json", bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		var res agentapi.ExecResult
		_ = json.NewDecoder(resp.Body).Decode(&res)

		return resp, res
	}

	// happy path
	resp, res := post(t, agentapi.ExecRequest{Argv: []string{"echo", "hello"}})
	if resp.StatusCode != http.StatusOK || res.ExitCode != 0 || strings.TrimSpace(res.Stdout) != "hello" {
		t.Errorf("echo: status=%d res=%+v", resp.StatusCode, res)
	}

	// nonzero exit passes through
	resp, res = post(t, agentapi.ExecRequest{Argv: []string{"sh", "-c", "echo err >&2; exit 3"}})
	if resp.StatusCode != http.StatusOK || res.ExitCode != 3 || strings.TrimSpace(res.Stderr) != "err" {
		t.Errorf("exit3: status=%d res=%+v", resp.StatusCode, res)
	}

	// empty argv -> 400
	resp, _ = post(t, agentapi.ExecRequest{Argv: nil})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("empty argv: status = %d, want 400", resp.StatusCode)
	}

	// timeout -> exit code -1
	resp, res = post(t, agentapi.ExecRequest{Argv: []string{"sleep", "5"}, TimeoutS: 1})
	if resp.StatusCode != http.StatusOK || res.ExitCode != -1 {
		t.Errorf("timeout: status=%d res=%+v", resp.StatusCode, res)
	}
	if res.DurationMS < 900 || res.DurationMS > 3000 {
		t.Errorf("timeout duration = %dms, want ~1000", res.DurationMS)
	}

	// unknown executable -> 400 exec_start
	resp, _ = post(t, agentapi.ExecRequest{Argv: []string{"definitely-not-a-command-xyz"}})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("not found: status = %d, want 400", resp.StatusCode)
	}

	// output truncation
	big := agentapi.ExecRequest{Argv: []string{"sh", "-c", "head -c 2097152 /dev/zero | tr '\\0' 'x'"}}
	resp, res = post(t, big)
	if resp.StatusCode != http.StatusOK || !res.Truncated {
		t.Errorf("truncation: status=%d res.truncated=%v", resp.StatusCode, res.Truncated)
	}
	if len(res.Stdout) > agentapi.MaxOutputBytes {
		t.Errorf("stdout len = %d, cap %d", len(res.Stdout), agentapi.MaxOutputBytes)
	}
}

func TestExecRequestBodyLimit(t *testing.T) {
	srv := newTestServer(t, "")

	big := strings.Repeat("a", agentapi.MaxExecRequestBytes+1024)
	resp, err := http.Post(srv.URL+agentapi.ExecRoute, "application/json", strings.NewReader(`{"argv":["true"],"workdir":"`+big+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("oversized body: status = %d, want 400", resp.StatusCode)
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	panicMux := http.NewServeMux()
	panicMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { panic("boom") })

	rec := httptest.NewRecorder()
	New(Options{Version: "test"}).withRecovery(panicMux).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("panic: status = %d, want 500", rec.Code)
	}
}

func TestEnsureToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "agent.token")

	token, err := EnsureToken(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(token) != 64 { // 32 bytes hex
		t.Errorf("token length = %d", len(token))
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("token file mode = %v, want 0600", info.Mode().Perm())
	}

	// second call reads the same token
	token2, err := EnsureToken(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(token2) != string(token) {
		t.Error("token regenerated on second call")
	}
}

func TestEnsureTokenFatalOnBadFile(t *testing.T) {
	t.Run("empty file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "agent.token")
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			t.Fatal(err)
		}

		if _, err := EnsureToken(path); err == nil {
			t.Error("empty token file must be a fatal error, got nil")
		}
	})

	t.Run("unreadable file", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("running as root: permission check meaningless")
		}

		path := filepath.Join(t.TempDir(), "agent.token")
		if err := os.WriteFile(path, []byte("token"), 0o000); err != nil {
			t.Fatal(err)
		}

		if _, err := EnsureToken(path); err == nil {
			t.Error("unreadable token file must be a fatal error, got nil")
		}
	})
}
