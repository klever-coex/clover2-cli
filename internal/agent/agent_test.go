package agent

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHealthAuth(t *testing.T) {
	token := []byte("secret-token")
	srv := httptest.NewServer(mux(token))
	defer srv.Close()

	// no token -> 401
	resp, err := http.Get(srv.URL + "/api/v1/health")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("no token: status = %d, want 401", resp.StatusCode)
	}

	// wrong token -> 401
	req, _ := http.NewRequest("GET", srv.URL+"/api/v1/health", nil)
	req.Header.Set("Authorization", "Bearer nope")
	if resp, err = http.DefaultClient.Do(req); err != nil {
		t.Fatal(err)
	}
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
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"agent_version", "robot_name", "uptime_s"} {
		if _, ok := payload[key]; !ok {
			t.Errorf("health payload missing %q: %v", key, payload)
		}
	}
}

func TestEnsureToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "agent.token")

	token, err := ensureToken(path)
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
	token2, err := ensureToken(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(token2) != string(token) {
		t.Error("token regenerated on second call")
	}
}
