package agent

import (
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	group := &cobra.Command{Use: "agent", Short: "On-robot agent (localhost HTTP API)"}

	group.AddCommand(runCmd())
	group.AddCommand(installCmd())

	return group
}
