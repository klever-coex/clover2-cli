package version

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/klever-coex/clover2-cli/internal/cliapp"
	core "github.com/klever-coex/clover2-cli/internal/version"
)

func bumpCmd() *cobra.Command {
	var base string

	cmd := &cobra.Command{
		Use:   "bump {major|minor|patch|rc}",
		Short: "Bump the canonical version (rc advances the rc series) and sync stores",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := requireProject()
			if err != nil {
				return err
			}

			var payload map[string]any
			switch args[0] {
			case "rc":
				payload, err = core.BumpRC(root, cfg, base)
			case "major", "minor", "patch":
				payload, err = core.Bump(root, cfg, args[0])
			default:
				return fmt.Errorf("invalid bump field '%s' (want major, minor, patch or rc)", args[0])
			}

			if err != nil {
				return err
			}

			return cliapp.Emit(cmd, cliapp.Payload(payload))
		},
	}

	cmd.Flags().StringVar(&base, "base", "minor", "base bump applied when cutting a fresh rc series")
	cmd.RegisterFlagCompletionFunc("base", cobra.FixedCompletions([]string{"major", "minor", "patch"}, cobra.ShellCompDirectiveNoFileComp))
	cliapp.PayloadFlags(cmd)

	return cmd
}
