// Package agent wires the "clover2 agent" command group to the agent server
// in internal/agent.
package agent

import (
	"github.com/spf13/cobra"
)

// Command returns the "agent" command group.
func Command() *cobra.Command {
	group := &cobra.Command{Use: "agent", Short: "On-robot agent (localhost HTTP API)"}

	group.AddCommand(runCmd(), installCmd())

	return group
}
