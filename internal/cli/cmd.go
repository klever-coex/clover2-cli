// Package cli assembles the clover2 root command from the command groups;
// each group lives in its own subpackage, one file per command.
package cli

import (
	"github.com/spf13/cobra"

	"github.com/klever-coex/clover2-cli/internal/cli/agent"
	"github.com/klever-coex/clover2-cli/internal/cli/artifact"
	"github.com/klever-coex/clover2-cli/internal/cli/version"
	"github.com/klever-coex/clover2-cli/internal/cliapp"
)

// NewCommand builds the root "clover2" command.
func NewCommand() *cobra.Command {
	var verbose int

	root := &cobra.Command{
		Use:           "clover2",
		Short:         "Clover2 platform manager: release pipeline and robot fleet",
		Version:       cliapp.Version,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	root.PersistentFlags().CountVarP(&verbose, "verbose", "v", "increase verbosity (-v: warnings, -vv: info, -vvv: debug)")
	root.PersistentFlags().StringVar(&cliapp.RootOverride, "root", "",
		"project root (must contain tooling/tooling.json; default: $CLOVER2_ROOT, else nearest parent)")
	root.SetVersionTemplate("clover2 {{.Version}}\n")
	root.PersistentPreRun = func(*cobra.Command, []string) { cliapp.SetLogLevel(verbose) }
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return cliapp.ExitErrorf(cliapp.ExitBadArgument, "%v", err)
	})
	root.AddCommand(version.Command(), artifact.Command(), agent.Command())

	return root
}
