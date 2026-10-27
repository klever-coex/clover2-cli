package system

import (
	"net/http"
	"runtime"
	"time"

	"github.com/klever-coex/clover2-cli/internal/agentapi"
	"github.com/klever-coex/clover2-cli/internal/httpx"
)

type Handler struct {
	version string
	started time.Time
	caps    func() agentapi.Capabilities
}

func NewHandler(version string, started time.Time, caps func() agentapi.Capabilities) *Handler {
	if caps == nil {
		caps = func() agentapi.Capabilities { return agentapi.Capabilities{} }
	}

	return &Handler{version: version, started: started, caps: caps}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET "+agentapi.HealthRoute, h.health)
	mux.HandleFunc("GET "+agentapi.InfoRoute, h.info)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, agentapi.Health{
		APIVersion:   agentapi.APIVersion,
		AgentVersion: h.version,
		RobotName:    hostname(),
		UptimeS:      int(time.Since(h.started).Seconds()),
	})
}

func (h *Handler) info(w http.ResponseWriter, r *http.Request) {
	total, free := diskUsage("/")
	memTotal, memAvail := memInfo()

	httpx.WriteJSON(w, http.StatusOK, agentapi.Info{
		APIVersion:   agentapi.APIVersion,
		AgentVersion: h.version,
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		Distro:       distro(),
		Hostname:     hostname(),
		AgentUser:    agentUser(),
		Capabilities: h.caps(),
		Disk:         agentapi.DiskInfo{TotalBytes: total, FreeBytes: free},
		Mem:          agentapi.MemInfo{TotalBytes: memTotal, AvailableBytes: memAvail},
	})
}
