package docker

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/klever-coex/clover2-cli/internal/cliapp"
)

func containersCmd(flags *cliapp.AgentTargetFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "containers",
		Short: "List container ids on the robot",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), cliapp.AgentTimeout)
			defer cancel()

			_, cfg, err := cliapp.Load()
			if err != nil {
				return err
			}

			client, err := cliapp.DialAgent(cfg, *flags)
			if err != nil {
				return err
			}

			list, err := client.DockerContainers(ctx)
			if err != nil {
				return agentErr(err)
			}

			return cliapp.Emit(cmd, idsResult{ids: list.IDs, data: list})
		},
	}
	cliapp.PayloadFlags(cmd)

	return cmd
}
