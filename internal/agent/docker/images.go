package docker

import (
	"net/http"

	"github.com/docker/docker/api/types/image"

	"github.com/klever-coex/clover2-cli/internal/agentapi"
	"github.com/klever-coex/clover2-cli/internal/httpx"
)

func (h *Handler) listImages(w http.ResponseWriter, r *http.Request) {
	if !h.available(w) {
		return
	}

	images, err := h.cli.ImageList(r.Context(), image.ListOptions{All: true})
	if err != nil {
		h.fail(w, err, "Unable to fetch list of images")
		return
	}

	ids := make([]string, 0, len(images))
	for _, img := range images {
		ids = append(ids, img.ID)
	}

	httpx.WriteJSON(w, http.StatusOK, agentapi.DockerImagesList{IDs: ids})
}

func (h *Handler) imageInfo(w http.ResponseWriter, r *http.Request) {
	if !h.available(w) {
		return
	}

	insp, err := h.cli.ImageInspect(r.Context(), r.PathValue("id"))
	if err != nil {
		h.fail(w, err, "Unable to inspect image")
		return
	}

	var labels map[string]string
	if insp.Config != nil {
		labels = insp.Config.Labels
	}

	httpx.WriteJSON(w, http.StatusOK, agentapi.DockerImageInfo{
		ID:           insp.ID,
		RepoTags:     insp.RepoTags,
		RepoDigests:  insp.RepoDigests,
		Created:      insp.Created,
		Size:         insp.Size,
		Architecture: insp.Architecture,
		OS:           insp.Os,
		Labels:       labels,
	})
}
