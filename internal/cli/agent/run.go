package agent

import (
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	core "github.com/klever-coex/clover2-cli/internal/agent"
)

func runCmd() *cobra.Command {
	var (
		listen    string
		tokenFile string
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the agent daemon (systemd unit installed via 'agent install')",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			return core.Serve(ctx, listen, tokenFile)
		},
	}

	cmd.Flags().StringVar(&listen, "listen", core.DefaultListen, "listen address (localhost by default; remote access goes through SSH tunnels)")
	cmd.Flags().StringVar(&tokenFile, "token-file", "", "bearer token file (generated 0600 if missing; empty = serve WITHOUT auth)")

	return cmd
}
