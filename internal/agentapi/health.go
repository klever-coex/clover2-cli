package agentapi

import "fmt"

type Health struct {
	APIVersion   int    `json:"api_version"`
	AgentVersion string `json:"agent_version"`
	RobotName    string `json:"robot_name"`
	UptimeS      int    `json:"uptime_s"`
}

func (h Health) CheckCompat() error {
	v := h.APIVersion
	if v == 0 {
		v = 1
	}

	if v != APIVersion {
		return &APIError{
			Code:    "version_mismatch",
			Message: fmt.Sprintf("agent speaks API v%d, client expects v%d; update the agent on the drone (clover2 remote add --bootstrap)", v, APIVersion),
		}
	}

	return nil
}
