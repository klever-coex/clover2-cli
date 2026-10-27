package docker

import (
	"github.com/spf13/cobra"

	"github.com/klever-coex/clover2-cli/internal/cliapp"
)

func Command() *cobra.Command {
	var flags cliapp.AgentTargetFlags

	group := &cobra.Command{
		Use:   "docker",
		Short: "Docker on a remote robot (via the agent API)",
	}
	group.PersistentFlags().StringVar(&flags.Agent, "agent", "",
		"agent base URL, e.g. http://192.168.1.10:7550 (default: agent.url in tooling.json)")
	group.PersistentFlags().StringVar(&flags.TokenFile, "token-file", "",
		"agent bearer token file (default: agent.token_file in tooling.json)")

	group.AddCommand(
		imagesCmd(&flags),
		containersCmd(&flags),
		imageCmd(&flags),
		containerCmd(&flags),
	)

	return group
}
