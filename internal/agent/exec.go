package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/klever-coex/clover2-cli/internal/agentapi"
)

const defaultExecTimeout = 30 * time.Second

func (s *server) handleExec(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, agentapi.MaxExecRequestBytes)

	var req agentapi.ExecRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body: "+err.Error())
		return
	}

	if len(req.Argv) == 0 {
		writeError(w, http.StatusBadRequest, "bad_request", "argv must contain at least one element")
		return
	}

	timeout := defaultExecTimeout
	if req.TimeoutS > 0 {
		timeout = time.Duration(min(req.TimeoutS, agentapi.MaxExecTimeoutS)) * time.Second
	}

	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, req.Argv[0], req.Argv[1:]...)
	if req.Workdir != "" {
		if st, err := os.Stat(req.Workdir); err != nil || !st.IsDir() {
			writeError(w, http.StatusBadRequest, "bad_request", "workdir does not exist: "+req.Workdir)
			return
		}

		cmd.Dir = req.Workdir
	}

	out := &cappedBuffer{cap: agentapi.MaxOutputBytes}
	errOut := &cappedBuffer{cap: agentapi.MaxOutputBytes}
	cmd.Stdout = out
	cmd.Stderr = errOut

	start := time.Now()
	runErr := cmd.Run()

	res := agentapi.ExecResult{
		Stdout:     out.String(),
		Stderr:     errOut.String(),
		Truncated:  out.truncated || errOut.truncated,
		DurationMS: time.Since(start).Milliseconds(),
	}

	switch {
	case runErr == nil:
		res.ExitCode = 0
	case ctx.Err() != nil:
		// Timed out on the client-supplied deadline, or the agent is
		// shutting down (BaseContext cancel). -1 = terminated by us.
		res.ExitCode = -1
		res.Stderr += "\ncancelled (timeout or agent shutdown)"
	case errors.Is(runErr, exec.ErrNotFound):
		writeError(w, http.StatusBadRequest, "exec_start", "executable not found in PATH: "+req.Argv[0])
		return
	default:
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			res.ExitCode = exitErr.ExitCode()
		} else {
			res.ExitCode = -1
		}
	}

	writeJSON(w, http.StatusOK, res)
}

// cappedBuffer collects command output up to cap bytes and flags truncation.
// It always reports success to the writer so the child is not killed by an
// EPIPE mid-run.
type cappedBuffer struct {
	buf       bytes.Buffer
	cap       int
	truncated bool
}

func (c *cappedBuffer) Write(p []byte) (int, error) {
	if c.buf.Len()+len(p) > c.cap {
		if room := c.cap - c.buf.Len(); room > 0 {
			c.buf.Write(p[:room])
		}

		c.truncated = true
		return len(p), nil
	}

	return c.buf.Write(p)
}

func (c *cappedBuffer) String() string { return c.buf.String() }
