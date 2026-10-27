package docker

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/klever-coex/clover2-cli/internal/agentapi"
	"github.com/klever-coex/clover2-cli/internal/cliapp"
)

// idsResult renders id lists as one short id per line for humans.
type idsResult struct {
	ids  []string
	data any
}

func (r idsResult) Text() string {
	out := make([]string, 0, len(r.ids))
	for _, id := range r.ids {
		out = append(out, shortID(id))
	}

	return strings.Join(out, "\n")
}

func (r idsResult) Data() any { return r.data }

// infoResult renders full documents as aligned key/value rows.
type infoResult struct {
	text string
	data any
}

func (r infoResult) Text() string { return r.text }
func (r infoResult) Data() any    { return r.data }

// shortID strips the sha256: prefix and cuts to 12 chars, docker-style.
func shortID(id string) string {
	if rest, ok := strings.CutPrefix(id, "sha256:"); ok {
		id = rest
	}

	if len(id) > 12 {
		id = id[:12]
	}

	return id
}

func labelsBlock(b *strings.Builder, labels map[string]string) {
	if len(labels) == 0 {
		return
	}

	fmt.Fprintf(b, "%-13s\n", "labels:")

	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		fmt.Fprintf(b, "  %s: %s\n", k, labels[k])
	}
}

func humanBytes(n int64) string {
	const unit = 1024
	switch {
	case n >= unit*unit*unit:
		return fmt.Sprintf("%.1f GiB", float64(n)/(unit*unit*unit))
	case n >= unit*unit:
		return fmt.Sprintf("%.1f MiB", float64(n)/(unit*unit))
	case n >= unit:
		return fmt.Sprintf("%.1f KiB", float64(n)/unit)
	default:
		return fmt.Sprintf("%d B", n)
	}
}

// agentErr maps agent API errors onto CLI exit codes.
func agentErr(err error) error {
	var apiErr *agentapi.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.Code {
		case "not_found":
			return cliapp.ExitErrorf(cliapp.ExitNotFound, "%s", apiErr.Message)
		case "service_unavailable":
			return cliapp.ExitErrorf(cliapp.ExitNetwork, "%s", apiErr.Message)
		default:
			return cliapp.ExitErrorf(cliapp.ExitNetwork, "agent error: %s", apiErr.Message)
		}
	}

	return cliapp.ExitErrorf(cliapp.ExitNetwork, "%v", err)
}
