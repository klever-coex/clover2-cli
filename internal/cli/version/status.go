package version

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/klever-coex/clover2-cli/internal/cliapp"
	core "github.com/klever-coex/clover2-cli/internal/version"
)

type storeRow struct {
	Reference string `json:"reference"`
	Version   string `json:"version"`
	Drift     bool   `json:"drift"`
}

type statusResult struct {
	canon string
	rows  []storeRow
}

func (r statusResult) Text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "canonical: %s\n", r.canon)

	for _, row := range r.rows {
		suffix := ""
		if row.Drift {
			suffix = "  (drift)"
		}

		fmt.Fprintf(&b, "%s: %s%s\n", row.Reference, row.Version, suffix)
	}

	return strings.TrimSuffix(b.String(), "\n")
}

func (r statusResult) Data() any {
	inSync := true
	for _, row := range r.rows {
		inSync = inSync && !row.Drift
	}

	return map[string]any{
		"version": r.canon,
		"in_sync": inSync,
		"stores":  r.rows,
	}
}

func statusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show canonical version, stores and drift",
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

			res := statusResult{canon: canon.String()}
			for _, s := range stores {
				v, err := s.Read()
				if err != nil {
					return err
				}

				res.rows = append(res.rows, storeRow{
					Reference: s.Reference(),
					Version:   v.String(),
					Drift:     v.String() != canon.String(),
				})
			}

			return cliapp.Emit(cmd, res)
		},
	}

	cliapp.PayloadFlags(cmd)
	return cmd
}
