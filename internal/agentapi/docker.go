package agentapi

// List payloads carry identifiers only; full details come from the
// per-id routes (DockerImageInfoPattern, DockerContainerInfoPattern).

type DockerImagesList struct {
	IDs []string `json:"ids" yaml:"ids"`
}

type DockerContainersList struct {
	IDs []string `json:"ids" yaml:"ids"`
}

// DockerImageInfo is the full image document (GET DockerImageInfoPattern).
type DockerImageInfo struct {
	ID           string            `json:"id" yaml:"id"`
	RepoTags     []string          `json:"repo_tags" yaml:"repo_tags"`
	RepoDigests  []string          `json:"repo_digests" yaml:"repo_digests"`
	Created      string            `json:"created" yaml:"created"`
	Size         int64             `json:"size" yaml:"size"`
	Architecture string            `json:"architecture" yaml:"architecture"`
	OS           string            `json:"os" yaml:"os"`
	Labels       map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

// DockerContainerInfo is the full container document (GET DockerContainerInfoPattern).
type DockerContainerInfo struct {
	ID       string            `json:"id" yaml:"id"`
	Name     string            `json:"name" yaml:"name"`
	Image    string            `json:"image" yaml:"image"`
	ImageID  string            `json:"image_id" yaml:"image_id"`
	Created  string            `json:"created" yaml:"created"`
	State    string            `json:"state" yaml:"state"`
	Platform string            `json:"platform" yaml:"platform"`
	Labels   map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}
