package version

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/spf13/cobra"

	"github.com/klever-coex/clover2-cli/internal/cliapp"
)

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

// statusResult: table text for humans, structured object for --json.
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

			canon, err := requireCanon(cfg)
			if err != nil {
				return err
			}

			stores, err := Discover(root, cfg.Exclude)
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
				return initCanon(cliapp.RootOverride, cfg, from)
			}

			root, cfg, err := cliapp.Load()
			if err == nil && root != "" {
				return initCanon(root, cfg, from)
			}

			cwd, err := os.Getwd()
			if err != nil {
				return err
			}

			cfg, err = cliapp.LoadDir(cwd)
			if err != nil {
				return err
			}

			return initCanon(cwd, cfg, from)
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "reference of the source store, e.g. ros::clover2")
	return cmd
}

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

			target := bare(*parsed)
			if err := writeCanonVersion(root, target.String()); err != nil {
				return err
			}

			stores, err := Discover(root, cfg.Exclude)
			if err != nil {
				return err
			}

			changed, err := changedStores(stores, target)
			if err != nil {
				return err
			}

			if err := Sync(stores, target); err != nil {
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

func syncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Write the canonical version into all stores",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := requireProject()
			if err != nil {
				return err
			}

			canon, err := requireCanon(cfg)
			if err != nil {
				return err
			}

			stores, err := Discover(root, cfg.Exclude)
			if err != nil {
				return err
			}

			changed, err := changedStores(stores, canon)
			if err != nil {
				return err
			}

			if err := Sync(stores, canon); err != nil {
				return err
			}

			return cliapp.Emit(cmd, cliapp.Payload{"updated": len(changed), "stores": changed})
		},
	}

	cliapp.PayloadFlags(cmd)
	return cmd
}

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
				payload, err = BumpRC(root, cfg, base)
			case "major", "minor", "patch":
				payload, err = Bump(root, cfg, args[0])
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

			payload, err := Compose(root, cfg, ref, mode, latestRC, latestStable)
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

// changedStores returns sorted references whose version differs from canon.
func changedStores(stores []Store, canon semver.Version) ([]string, error) {
	changed := []string{}
	for _, s := range stores {
		v, err := s.Read()
		if err != nil {
			return nil, err
		}

		if v.String() != canon.String() {
			changed = append(changed, s.Reference())
		}
	}

	sort.Strings(changed)
	return changed, nil
}
