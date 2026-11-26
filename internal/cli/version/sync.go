package version

import (
	"github.com/spf13/cobra"

	"github.com/klever-coex/clover2-cli/internal/cliapp"
	core "github.com/klever-coex/clover2-cli/internal/version"
)

func syncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Write the canonical version into all stores",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := requireProject()
			if err != nil {
				return err
			}

			canon, err := core.RequireCanon(cfg)
			if err != nil {
				return err
			}

			stores, err := core.Discover(root, cfg.Exclude)
			if err != nil {
				return err
			}

			changed, err := core.ChangedStores(stores, canon)
			if err != nil {
				return err
			}

			if err := core.Sync(stores, canon); err != nil {
				return err
			}

			return cliapp.Emit(cmd, cliapp.Payload{"updated": len(changed), "stores": changed})
		},
	}

	cliapp.PayloadFlags(cmd)
	return cmd
}
