package agent

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	core "github.com/klever-coex/clover2-cli/internal/agent"
	"github.com/klever-coex/clover2-cli/internal/cliapp"
)

const (
	unitPath  = "/etc/systemd/system/clover2-agent.service"
	unitName  = "clover2-agent.service"
	listenLan = "0.0.0.0:7550"
)

func installCmd() *cobra.Command {
	var userName string

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install and start the agent systemd unit (run as root on the robot)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if os.Geteuid() != 0 {
				return cliapp.ExitErrorf(cliapp.ExitGeneric, "agent install must run as root (use sudo)")
			}

			name := userName
			if name == "" {
				name = os.Getenv("SUDO_USER")
			}
			if name == "" {
				var err error
				if name, err = firstLoginUser(); err != nil {
					return cliapp.ExitErrorf(cliapp.ExitBadArgument, "cannot pick a user: %v; pass --user", err)
				}
			}

			u, err := user.Lookup(name)
			if err != nil {
				return fmt.Errorf("user %q: %w", name, err)
			}

			return installAgent(u)
		},
	}

	cmd.Flags().StringVar(&userName, "user", "", "user to run the agent as (default: $SUDO_USER, else the first regular user)")

	return cmd
}

// installAgent writes the token and unit, then enables and starts the
// service. Idempotent: safe to re-run on upgrades and to repair.
func installAgent(u *user.User) error {
	uid, _ := strconv.Atoi(u.Uid)
	gid, _ := strconv.Atoi(u.Gid)

	// Token must be readable by the agent user: generate when missing
	// (fatal on unreadable/empty), then hand it over.
	if _, err := core.EnsureToken(core.DefaultTokenFile); err != nil {
		return err
	}

	if err := os.Chown(core.DefaultTokenFile, uid, gid); err != nil {
		return err
	}

	if err := os.WriteFile(unitPath, []byte(renderUnit(u.Username)), 0o644); err != nil {
		return err
	}

	// Best-effort systemctl: inside image builds there is no running PID 1,
	// enable still lays down the boot symlink there.
	for _, args := range [][]string{
		{"daemon-reload"},
		{"enable", unitName},
		{"restart", unitName},
	} {
		if out, err := exec.Command("systemctl", args...).CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: systemctl %s failed: %v: %s\n",
				strings.Join(args, " "), err, strings.TrimSpace(string(out)))
		}
	}

	fmt.Printf("agent installed (user %s, listen %s, token %s)\n", u.Username, listenLan, core.DefaultTokenFile)

	return nil
}

func renderUnit(userName string) string {
	return fmt.Sprintf(`[Unit]
Description=clover2 agent (management API)
After=network-online.target
Wants=network-online.target

[Service]
User=%s
ExecStart=/usr/bin/clover2 agent run --listen %s --token-file %s
Restart=on-failure
RestartSec=3
KillMode=mixed

[Install]
WantedBy=multi-user.target
`, userName, listenLan, core.DefaultTokenFile)
}

// firstLoginUser returns the first regular (uid>=1000) user from
// /etc/passwd — the robot owner on a stock image.
// ponytail: first uid>=1000; pass --user explicitly on multi-user boxes.
func firstLoginUser() (string, error) {
	data, err := os.ReadFile("/etc/passwd")
	if err != nil {
		return "", err
	}

	return parseFirstLoginUser(string(data))
}

func parseFirstLoginUser(passwd string) (string, error) {
	for _, line := range strings.Split(passwd, "\n") {
		f := strings.Split(line, ":")
		if len(f) < 4 {
			continue
		}

		uid, err := strconv.Atoi(f[2])
		if err != nil || uid < 1000 || uid >= 60000 {
			continue
		}

		return f[0], nil
	}

	return "", fmt.Errorf("no regular user (uid>=1000) found")
}
