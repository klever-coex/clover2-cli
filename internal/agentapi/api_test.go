package agentapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientHealthAndInfo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer tok" {
			t.Errorf("Authorization = %q", got)
		}

		switch r.URL.Path {
		case HealthRoute:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(Health{APIVersion: APIVersion, AgentVersion: "0.1.0", RobotName: "drone-1", UptimeS: 5})
		case InfoRoute:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(Info{APIVersion: APIVersion, Hostname: "drone-1", OS: "linux", Arch: "arm64",
				Capabilities: Capabilities{Docker: true, SudoPasswordless: true}})
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok")

	h, err := c.Health(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if h.AgentVersion != "0.1.0" || h.RobotName != "drone-1" {
		t.Errorf("health = %+v", h)
	}

	info, err := c.Info(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !info.Capabilities.Docker || info.OS != "linux" {
		t.Errorf("info = %+v", info)
	}
}

func TestClientExecRoundtrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ExecRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if len(req.Argv) != 2 || req.Argv[0] != "uname" {
			t.Errorf("argv = %v", req.Argv)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ExecResult{ExitCode: 0, Stdout: "Linux\n", DurationMS: 12})
	}))
	defer srv.Close()

	res, err := NewClient(srv.URL, "").Exec(context.Background(), ExecRequest{Argv: []string{"uname", "-a"}})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExitCode != 0 || res.Stdout != "Linux\n" || res.DurationMS != 12 {
		t.Errorf("result = %+v", res)
	}
}

func TestClientErrorEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"bad token"}}`))
	}))
	defer srv.Close()

	_, err := NewClient(srv.URL, "wrong").Health(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want *APIError, got %v", err)
	}
	if apiErr.Status != http.StatusUnauthorized || apiErr.Code != "unauthorized" || apiErr.Message != "bad token" {
		t.Errorf("apiErr = %+v", apiErr)
	}
	if !strings.Contains(apiErr.Error(), "401") || !strings.Contains(apiErr.Error(), "bad token") {
		t.Errorf("Error() = %q", apiErr.Error())
	}
}

func TestClientNonEnvelopeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("<html>gateway exploded</html>"))
	}))
	defer srv.Close()

	_, err := NewClient(srv.URL, "").Health(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want *APIError, got %v", err)
	}
	if apiErr.Code != "bad_response" || !strings.Contains(apiErr.Message, "gateway exploded") {
		t.Errorf("apiErr = %+v", apiErr)
	}
}

func TestCheckCompat(t *testing.T) {
	cases := []struct {
		got     int
		wantErr bool
	}{
		{got: 0, wantErr: false}, // pre-1.0 agent: treated as v1
		{got: 1, wantErr: false},
		{got: 2, wantErr: true},
	}

	for _, tc := range cases {
		err := (Health{APIVersion: tc.got}).CheckCompat()
		if (err != nil) != tc.wantErr {
			t.Errorf("APIVersion %d: err = %v, wantErr %v", tc.got, err, tc.wantErr)
		}
	}
}
