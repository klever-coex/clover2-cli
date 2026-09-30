package version

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/klever-coex/clover2-cli/internal/cliapp"
	core "github.com/klever-coex/clover2-cli/internal/version"
)

func initCmd() *cobra.Command {
	var from string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Seed the canonical version from an existing store (one-time migration)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if from == "" {
				return fmt.Errorf("--from <store::name> is required, e.g. --from ros::clover2")
			}

			// init may create the config, so a missing tooling.json is fine:
			// use --root, else an existing project found upwards, else cwd.
			if cliapp.RootOverride != "" {
				cfg, err := cliapp.LoadDir(cliapp.RootOverride)
				if err != nil {
					return err
				}
				return core.InitCanon(cliapp.RootOverride, cfg, from)
			}

			root, cfg, err := cliapp.Load()
			if err == nil && root != "" {
				return core.InitCanon(root, cfg, from)
			}

			cwd, err := os.Getwd()
			if err != nil {
				return err
			}

			cfg, err = cliapp.LoadDir(cwd)
			if err != nil {
				return err
			}

			return core.InitCanon(cwd, cfg, from)
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "reference of the source store, e.g. ros::clover2")
	return cmd
}
