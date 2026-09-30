// Package agentapi is the contract between the on-robot agent (server) and
// the CLI (client): route paths, payload DTOs and the error envelope. Both
// sides import this package, so the wire format cannot drift apart.
package agentapi

import "fmt"

// APIVersion is the contract version. It is bumped on breaking payload or
// semantic changes; clients refuse to talk to agents with a different
// version and instruct the user to update the agent.
const APIVersion = 1

// Route paths served by the agent.
const (
	HealthRoute = "/api/v1/health"
	InfoRoute   = "/api/v1/info"
	ExecRoute   = "/api/v1/exec"
)

// MaxExecRequestBytes caps the exec request body.
const MaxExecRequestBytes = 1 << 20 // 1 MiB

// MaxOutputBytes caps buffered exec stdout/stderr before truncation.
const MaxOutputBytes = 1 << 20 // 1 MiB

// MaxExecTimeoutS caps the per-command timeout accepted from a client.
const MaxExecTimeoutS = 300

// Capabilities reports what the machine provides. Commands check them before
// starting and fail with actionable errors instead of failing mid-way.
type Capabilities struct {
	SudoPasswordless bool `json:"sudo_passwordless"` // `sudo -n true` succeeds
	NetworkManager   bool `json:"networkmanager"`    // nmcli available
	Docker           bool `json:"docker"`            // docker available
	Avahi            bool `json:"avahi"`             // avahi-daemon publishing available
	Systemd          bool `json:"systemd"`           // systemd as PID 1
}

// Health is the liveness payload. It stays intentionally small so monitoring
// can poll it cheaply; Info carries the full picture.
type Health struct {
	APIVersion   int    `json:"api_version"`
	AgentVersion string `json:"agent_version"`
	RobotName    string `json:"robot_name"`
	UptimeS      int    `json:"uptime_s"`
}

// CheckCompat validates the agent API version against this client. A missing
// field (zero) means a pre-1.0 agent and is treated as version 1.
func (h Health) CheckCompat() error {
	v := h.APIVersion
	if v == 0 {
		v = 1
	}

	if v != APIVersion {
		return &APIError{
			Code:    "version_mismatch",
			Message: fmt.Sprintf("agent speaks API v%d, client expects v%d; update the agent on the drone (clover2 remote add --bootstrap)", v, APIVersion),
		}
	}

	return nil
}

type Info struct {
	APIVersion   int    `json:"api_version"`
	AgentVersion string `json:"agent_version"`
	OS           string `json:"os"`   // runtime.GOOS
	Arch         string `json:"arch"` // runtime.GOARCH
	Distro       string `json:"distro,omitempty"`
	Hostname     string `json:"hostname"`
	AgentUser    string `json:"agent_user"`

	Capabilities Capabilities `json:"capabilities"`

	Disk DiskInfo `json:"disk"`
	Mem  MemInfo  `json:"mem"`
}

type DiskInfo struct {
	TotalBytes uint64 `json:"total_bytes"`
	FreeBytes  uint64 `json:"free_bytes"`
}

type MemInfo struct {
	TotalBytes     uint64 `json:"total_bytes"`
	AvailableBytes uint64 `json:"available_bytes"`
}

type ExecRequest struct {
	// Argv is executed directly (no shell): argv[0] is resolved via PATH by
	// the agent, argv elements are passed verbatim.
	Argv     []string `json:"argv"`
	TimeoutS int      `json:"timeout_s,omitempty"`
	Workdir  string   `json:"workdir,omitempty"`
}

type ExecResult struct {
	ExitCode   int    `json:"exit_code"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	Truncated  bool   `json:"truncated"`
	DurationMS int64  `json:"duration_ms"`
}

// Error is the API error envelope: every non-2xx response carries
// {"error":{"code":"...","message":"..."}}.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorEnvelope struct {
	Error Error `json:"error"`
}

// APIError is a structured error response from the agent (or a client-side
// stand-in when the response body is not a valid envelope).
type APIError struct {
	Status  int // HTTP status; 0 for client-side failures
	Code    string
	Message string
}

func (e *APIError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = e.Code
	}

	if e.Status != 0 {
		return fmt.Sprintf("agent error (HTTP %d): %s", e.Status, msg)
	}

	return "agent error: " + msg
}
