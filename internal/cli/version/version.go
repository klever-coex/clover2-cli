// Package version wires the "clover2 version" command group to the
// versioning core in internal/version.
package version

import (
	"github.com/spf13/cobra"

	"github.com/klever-coex/clover2-cli/internal/cliapp"
)

// Command returns the "version" command group.
func Command() *cobra.Command {
	group := &cobra.Command{
		Use:   "version",
		Short: "Project version management (canonical version in tooling.json)",
	}

	group.AddCommand(
		statusCmd(),
		initCmd(),
		setCmd(),
		syncCmd(),
		bumpCmd(),
		composeCmd(),
	)

	return group
}

func requireProject() (string, cliapp.Config, error) {
	root, cfg, err := cliapp.Load()
	if err != nil {
		return "", cfg, err
	}

	if root == "" {
		return "", cfg, cliapp.ExitErrorf(cliapp.ExitNoProject, "%v", cliapp.ErrNoProject)
	}

	return root, cfg, nil
}
