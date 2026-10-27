package cliapp

import (
	"os"
	"strings"
	"time"

	"github.com/klever-coex/clover2-cli/internal/agentapi"
)

const AgentTimeout = 10 * time.Second

type AgentTargetFlags struct {
	Agent     string // --agent: agent base URL
	TokenFile string // --token-file: bearer token file
}

func DialAgent(cfg Config, flags AgentTargetFlags) (*agentapi.Client, error) {
	endpoint := flags.Agent
	if endpoint == "" {
		endpoint = cfg.Agent.URL
	}

	if endpoint == "" {
		return nil, ExitErrorf(ExitBadArgument,
			"no agent address: pass --agent URL or set agent.url in tooling/tooling.json")
	}

	tokenFile := flags.TokenFile
	if tokenFile == "" {
		tokenFile = cfg.Agent.TokenFile
	}

	token := ""
	if tokenFile != "" {
		raw, err := os.ReadFile(tokenFile)
		if err != nil {
			return nil, ExitErrorf(ExitBadArgument, "agent token file: %v", err)
		}

		token = strings.TrimSpace(string(raw))
	}

	return agentapi.NewClient(endpoint, token), nil
}
