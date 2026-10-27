package agentapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	base  string
	token string
	http  *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		base:  strings.TrimSuffix(baseURL, "/"),
		token: token,
		http:  &http.Client{},
	}
}

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

func idRoute(pattern, id string) string {
	return strings.Replace(pattern, "{id}", url.PathEscape(id), 1)
}

func (c *Client) DockerImages(ctx context.Context) (DockerImagesList, error) {
	var out DockerImagesList
	return out, c.get(ctx, DockerImagesRoute, &out)
}

func (c *Client) DockerImageInfo(ctx context.Context, id string) (DockerImageInfo, error) {
	var out DockerImageInfo
	return out, c.get(ctx, idRoute(DockerImageInfoPattern, id), &out)
}

func (c *Client) DockerContainers(ctx context.Context) (DockerContainersList, error) {
	var out DockerContainersList
	return out, c.get(ctx, DockerContainersRoute, &out)
}

func (c *Client) DockerContainerInfo(ctx context.Context, id string) (DockerContainerInfo, error) {
	var out DockerContainerInfo
	return out, c.get(ctx, idRoute(DockerContainerInfoPattern, id), &out)
}

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
