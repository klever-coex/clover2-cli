package docker

import (
	"context"
	"net/http"
	"time"

	"github.com/docker/docker/client"

	"github.com/klever-coex/clover2-cli/internal/agentapi"
	"github.com/klever-coex/clover2-cli/internal/httpx"
)

type Handler struct {
	cli *client.Client
}

func NewHandler(cli *client.Client) *Handler {
	return &Handler{cli: cli}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET "+agentapi.DockerImagesRoute, h.listImages)
	mux.HandleFunc("GET "+agentapi.DockerImageInfoPattern, h.imageInfo)
	mux.HandleFunc("GET "+agentapi.DockerContainersRoute, h.listContainers)
	mux.HandleFunc("GET "+agentapi.DockerContainerInfoPattern, h.containerInfo)
}

func (h *Handler) available(w http.ResponseWriter) bool {
	if h.cli == nil {
		httpx.WriteError(w,
			http.StatusServiceUnavailable,
			"service_unavailable",
			"Docker is not available on this host")
		return false
	}

	return true
}

const probeTimeout = 2 * time.Second

func (h *Handler) IsHealthy(ctx context.Context) bool {
	if h.cli == nil {
		return false
	}

	probeCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	_, err := h.cli.Info(probeCtx)
	if err != nil {
		return false
	}

	return true
}

func (h *Handler) fail(w http.ResponseWriter, err error, msg string) {
	switch {
	case client.IsErrConnectionFailed(err):
		httpx.WriteError(w,
			http.StatusServiceUnavailable,
			"service_unavailable",
			"Docker is not available on this host")
	case client.IsErrNotFound(err):
		httpx.WriteError(w,
			http.StatusNotFound,
			"not_found",
			"No such docker object")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "internal_server_error", msg)
	}
}
