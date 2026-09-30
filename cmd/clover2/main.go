package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/klever-coex/clover2-cli/internal/cli"
	"github.com/klever-coex/clover2-cli/internal/cliapp"
)

func main() {
	if err := cli.NewCommand().Execute(); err != nil {
		code := cliapp.ExitGeneric
		var coded *cliapp.Error
		if errors.As(err, &coded) {
			code = coded.Code
		}

		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(int(code))
	}
}
