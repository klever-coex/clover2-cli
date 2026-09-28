package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/klever-coex/clover2-cli/internal/agent"
	"github.com/klever-coex/clover2-cli/internal/artifact"
	"github.com/klever-coex/clover2-cli/internal/cliapp"
	"github.com/klever-coex/clover2-cli/internal/version"
	"github.com/spf13/cobra"
)

func main() {
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

	if err := root.Execute(); err != nil {
		code := cliapp.ExitGeneric
		var coded *cliapp.Error
		if errors.As(err, &coded) {
			code = coded.Code
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(int(code))
	}
}
