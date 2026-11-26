package agent

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/klever-coex/clover2-cli/internal/agentapi"
)

const capabilitiesRefreshInterval = 15 * time.Second

type probe func(ctx context.Context) bool

type capability struct {
	probe  probe
	cached atomic.Bool
}

func (c *capability) refresh(ctx context.Context) {
	c.cached.Store(c.probe(ctx))
}

type capabilitySet struct {
	docker capability
}

func newCapabilitySet(dockerProbe probe) *capabilitySet {
	return &capabilitySet{
		docker: capability{probe: dockerProbe},
	}
}

func (cs *capabilitySet) refresh(ctx context.Context) {
	cs.docker.refresh(ctx)
}

func (cs *capabilitySet) run(ctx context.Context) {
	cs.refresh(ctx)

	ticker := time.NewTicker(capabilitiesRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cs.refresh(ctx)
		}
	}
}

func (cs *capabilitySet) snapshot() agentapi.Capabilities {
	return agentapi.Capabilities{
		Docker: cs.docker.cached.Load(),
	}
}
