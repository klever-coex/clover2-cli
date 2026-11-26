package agentapi

import "fmt"

type ExecRequest struct {
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

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorEnvelope struct {
	Error Error `json:"error"`
}

type APIError struct {
	Status  int
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
