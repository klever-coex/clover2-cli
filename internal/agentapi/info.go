package agentapi

type Capabilities struct {
	Docker bool `json:"docker"`
}

type Info struct {
	APIVersion   int    `json:"api_version"`
	AgentVersion string `json:"agent_version"`
	OS           string `json:"os"`
	Arch         string `json:"arch"`
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
