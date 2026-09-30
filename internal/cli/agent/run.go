// Package agent wires the "clover2 agent" command group to the agent server
// in internal/agent.
package agent

import (
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	core "github.com/klever-coex/clover2-cli/internal/agent"
)

// Command returns the "agent" command group.
func Command() *cobra.Command {
	group := &cobra.Command{Use: "agent", Short: "On-robot agent (localhost HTTP API)"}

	group.AddCommand(runCmd())

	return group
}

func runCmd() *cobra.Command {
	var (
		listen    string
		tokenFile string
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the agent daemon (systemd unit: packaging/clover2-agent.service)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			return core.Serve(ctx, listen, tokenFile)
		},
	}

	cmd.Flags().StringVar(&listen, "listen", core.DefaultListen, "listen address (localhost by default; remote access goes through SSH tunnels)")
	cmd.Flags().StringVar(&tokenFile, "token-file", core.DefaultTokenPath(), "bearer token file (generated 0600 if missing)")

	return cmd
}
