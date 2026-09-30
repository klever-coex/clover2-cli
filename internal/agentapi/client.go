package agentapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Client talks to an agent over HTTP. The zero value is not usable; build one
// with NewClient. A token may be empty for agents running without auth.
type Client struct {
	base  string
	token string
	http  *http.Client
}

// NewClient builds a client for an agent at baseURL (e.g.
// "http://192.168.11.1:7550"). token may be empty.
func NewClient(baseURL, token string) *Client {
	return &Client{
		base:  strings.TrimSuffix(baseURL, "/"),
		token: token,
		http:  &http.Client{},
	}
}

// Health fetches the liveness payload and checks API compatibility.
func (c *Client) Health(ctx context.Context) (Health, error) {
	var h Health
	if err := c.get(ctx, HealthRoute, &h); err != nil {
		return Health{}, err
	}

	if err := h.CheckCompat(); err != nil {
		return Health{}, err
	}

	return h, nil
}

// Info fetches the machine description and checks API compatibility.
func (c *Client) Info(ctx context.Context) (Info, error) {
	var info Info
	if err := c.get(ctx, InfoRoute, &info); err != nil {
		return Info{}, err
	}

	if err := (Health{APIVersion: info.APIVersion}).CheckCompat(); err != nil {
		return Info{}, err
	}

	return info, nil
}

// Exec runs argv on the agent machine. The context controls the overall call
// deadline; req.TimeoutS bounds the command on the agent side.
func (c *Client) Exec(ctx context.Context, req ExecRequest) (ExecResult, error) {
	var res ExecResult
	err := c.post(ctx, ExecRoute, req, &res)

	return res, err
}

func (c *Client) get(ctx context.Context, route string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+route, nil)
	if err != nil {
		return err
	}

	return c.do(req, out)
}

func (c *Client) post(ctx context.Context, route string, in, out any) error {
	body, err := json.Marshal(in)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+route, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	return c.do(req, out)
}

func (c *Client) do(req *http.Request, out any) error {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("agent unreachable at %s: %w", c.base, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxOutputBytes*4))
	if err != nil {
		return fmt.Errorf("reading agent response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return decodeAPIError(resp.StatusCode, data)
	}

	if out == nil {
		return nil
	}

	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("invalid agent response: %w", err)
	}

	return nil
}

// decodeAPIError extracts the error envelope; a non-envelope body falls back
// to a structured error carrying the raw text.
func decodeAPIError(status int, data []byte) error {
	var env errorEnvelope
	if err := json.Unmarshal(data, &env); err == nil && env.Error.Code != "" {
		return &APIError{Status: status, Code: env.Error.Code, Message: env.Error.Message}
	}

	return &APIError{
		Status:  status,
		Code:    "bad_response",
		Message: strings.TrimSpace(string(data)),
	}
}
