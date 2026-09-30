package agent

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"time"
)

const mdnsService = "_clover2._tcp"

// announceMDNS publishes the agent service via avahi-publish, a small
// subprocess that stays alive while the agent runs. Best-effort: without
// avahi the agent still serves and only warns once. The child is tied to
// ctx, so agent shutdown reaps it (KillMode=mixed on the unit does the rest).
func announceMDNS(ctx context.Context, port int, txt map[string]string) {
	if !commandAvailable("avahi-publish") {
		slog.Warn("avahi-publish not found; mDNS announce disabled (install avahi-utils to enable 'remote scan')")
		return
	}

	name, err := os.Hostname()
	if err != nil || name == "" {
		name = "clover2-agent"
	}

	go func() {
		backoff := 5 * time.Second
		for {
			args := []string{"-s", name, mdnsService, strconv.Itoa(port)}
			for k, v := range txt {
				args = append(args, k+"="+v)
			}

			err := exec.CommandContext(ctx, "avahi-publish", args...).Run()
			if ctx.Err() != nil {
				return
			}

			slog.Warn("avahi-publish exited; retrying", "err", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}

			if backoff < 60*time.Second {
				backoff *= 2
			}
		}
	}()
}
