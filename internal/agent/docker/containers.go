package docker

import (
	"net/http"
	"strings"

	"github.com/docker/docker/api/types/container"

	"github.com/klever-coex/clover2-cli/internal/agentapi"
	"github.com/klever-coex/clover2-cli/internal/httpx"
)

func (h *Handler) listContainers(w http.ResponseWriter, r *http.Request) {
	if !h.available(w) {
		return
	}

	containers, err := h.cli.ContainerList(r.Context(), container.ListOptions{All: true})
	if err != nil {
		h.fail(w, err, "Fail to read containers")
		return
	}

	ids := make([]string, 0, len(containers))
	for _, c := range containers {
		ids = append(ids, c.ID)
	}

	httpx.WriteJSON(w, http.StatusOK, agentapi.DockerContainersList{IDs: ids})
}

func (h *Handler) containerInfo(w http.ResponseWriter, r *http.Request) {
	if !h.available(w) {
		return
	}

	insp, err := h.cli.ContainerInspect(r.Context(), r.PathValue("id"))
	if err != nil {
		h.fail(w, err, "Unable to inspect container")
		return
	}

	state := ""
	if insp.State != nil {
		state = insp.State.Status
	}

	image := ""
	var labels map[string]string
	if insp.Config != nil {
		image = insp.Config.Image
		labels = insp.Config.Labels
	}

	httpx.WriteJSON(w, http.StatusOK, agentapi.DockerContainerInfo{
		ID:       insp.ID,
		Name:     strings.TrimPrefix(insp.Name, "/"),
		Image:    image,
		ImageID:  insp.Image,
		Created:  insp.Created,
		State:    state,
		Platform: insp.Platform,
		Labels:   labels,
	})
}
