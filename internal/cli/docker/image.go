package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/klever-coex/clover2-cli/internal/agentapi"
	"github.com/klever-coex/clover2-cli/internal/cliapp"
)

func imageCmd(flags *cliapp.AgentTargetFlags) *cobra.Command {
	group := &cobra.Command{
		Use:   "image",
		Short: "Image details",
	}

	info := &cobra.Command{
		Use:   "info <ID>",
		Short: "Full details of one image",
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

			info, err := client.DockerImageInfo(ctx, args[0])
			if err != nil {
				return agentErr(err)
			}

			return cliapp.Emit(cmd, infoResult{data: info, text: imageInfoText(info)})
		},
	}
	cliapp.PayloadFlags(info)
	group.AddCommand(info)

	return group
}

func imageInfoText(v agentapi.DockerImageInfo) string {
	var b strings.Builder
	row := func(k, val string) { fmt.Fprintf(&b, "%-13s %s\n", k+":", val) }

	row("id", v.ID)
	row("repo_tags", strings.Join(v.RepoTags, ", "))
	row("repo_digests", strings.Join(v.RepoDigests, ", "))
	row("created", v.Created)
	row("size", humanBytes(v.Size))
	row("arch", v.Architecture)
	row("os", v.OS)
	labelsBlock(&b, v.Labels)

	return strings.TrimSuffix(b.String(), "\n")
}
