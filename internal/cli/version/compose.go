package version

import (
	"github.com/spf13/cobra"

	"github.com/klever-coex/clover2-cli/internal/cliapp"
	core "github.com/klever-coex/clover2-cli/internal/version"
)

func composeCmd() *cobra.Command {
	var ref, mode string
	var latestRC, latestStable bool

	cmd := &cobra.Command{
		Use:   "compose",
		Short: "Compose the full version from the git context",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := requireProject()
			if err != nil {
				return err
			}

			payload, err := core.Compose(root, cfg, ref, mode, latestRC, latestStable)
			if err != nil {
				return err
			}

			return cliapp.Emit(cmd, cliapp.Payload(payload))
		},
	}

	cmd.Flags().StringVar(&ref, "ref", "", "git ref: refs/tags/vX, refs/heads/... or bare vX tag")
	cmd.Flags().StringVar(&mode, "mode", "", "build mode: develop, master, release, pre-release")
	cmd.Flags().BoolVar(&latestRC, "latest-rc", false, "report the newest rc tag")
	cmd.Flags().BoolVar(&latestStable, "latest-stable", false, "report the newest stable tag")
	cliapp.PayloadFlags(cmd)

	return cmd
}
