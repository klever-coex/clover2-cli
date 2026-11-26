package version

import (
	"github.com/Masterminds/semver/v3"
	"github.com/spf13/cobra"

	"github.com/klever-coex/clover2-cli/internal/cliapp"
	core "github.com/klever-coex/clover2-cli/internal/version"
)

func setCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set VERSION",
		Short: "Set the canonical version and sync all stores",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := requireProject()
			if err != nil {
				return err
			}

			parsed, err := semver.StrictNewVersion(args[0])
			if err != nil {
				return cliapp.ExitErrorf(cliapp.ExitBadArgument, "invalid version '%s'", args[0])
			}

			target := core.Bare(*parsed)
			if err := core.WriteCanonVersion(root, target.String()); err != nil {
				return err
			}

			stores, err := core.Discover(root, cfg.Exclude)
			if err != nil {
				return err
			}

			changed, err := core.ChangedStores(stores, target)
			if err != nil {
				return err
			}

			if err := core.Sync(stores, target); err != nil {
				return err
			}

			return cliapp.Emit(cmd, cliapp.Payload{
				"version": target.String(),
				"updated": len(changed),
				"stores":  changed,
			})
		},
	}

	cliapp.PayloadFlags(cmd)
	return cmd
}
