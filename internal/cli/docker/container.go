package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/klever-coex/clover2-cli/internal/agentapi"
	"github.com/klever-coex/clover2-cli/internal/cliapp"
)

func containerCmd(flags *cliapp.AgentTargetFlags) *cobra.Command {
	group := &cobra.Command{
		Use:   "container",
		Short: "Container details",
	}

	info := &cobra.Command{
		Use:   "info <ID>",
		Short: "Full details of one container",
		Args:  cobra.ExactArgs(1),
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

			info, err := client.DockerContainerInfo(ctx, args[0])
			if err != nil {
				return agentErr(err)
			}

			return cliapp.Emit(cmd, infoResult{data: info, text: containerInfoText(info)})
		},
	}
	cliapp.PayloadFlags(info)
	group.AddCommand(info)

	return group
}

func containerInfoText(v agentapi.DockerContainerInfo) string {
	var b strings.Builder
	row := func(k, val string) { fmt.Fprintf(&b, "%-13s %s\n", k+":", val) }

	row("id", v.ID)
	row("name", v.Name)
	row("image", v.Image)
	row("image_id", v.ImageID)
	row("created", v.Created)
	row("state", v.State)
	row("platform", v.Platform)
	labelsBlock(&b, v.Labels)

	return strings.TrimSuffix(b.String(), "\n")
}
