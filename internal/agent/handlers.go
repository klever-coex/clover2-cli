package agent

import (
	"bufio"
	"context"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/klever-coex/clover2-cli/internal/agentapi"
)

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, agentapi.Health{
		APIVersion:   agentapi.APIVersion,
		AgentVersion: s.opts.Version,
		RobotName:    hostname(),
		UptimeS:      int(time.Since(s.opts.Started).Seconds()),
	})
}

func (s *server) handleInfo(w http.ResponseWriter, r *http.Request) {
	total, free := diskUsage("/")
	memTotal, memAvail := memInfo()

	writeJSON(w, http.StatusOK, agentapi.Info{
		APIVersion:   agentapi.APIVersion,
		AgentVersion: s.opts.Version,
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		Distro:       distro(),
		Hostname:     hostname(),
		AgentUser:    agentUser(),
		Capabilities: s.capabilities(r.Context()),
		Disk:         agentapi.DiskInfo{TotalBytes: total, FreeBytes: free},
		Mem:          agentapi.MemInfo{TotalBytes: memTotal, AvailableBytes: memAvail},
	})
}

// capabilities probes what the machine offers. Probes are cheap and run per
// info call: clients use them for actionable preflight errors.
func (s *server) capabilities(ctx context.Context) agentapi.Capabilities {
	return agentapi.Capabilities{
		SudoPasswordless: probeSudo(ctx),
		NetworkManager:   commandAvailable("nmcli"),
		Docker:           commandAvailable("docker"),
		Avahi:            commandAvailable("avahi-publish"),
		Systemd:          systemdRunning(),
	}
}

func probeSudo(ctx context.Context) bool {
	probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(probeCtx, "sudo", "-n", "true")
	return cmd.Run() == nil
}

func commandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func systemdRunning() bool {
	_, err := os.Stat("/run/systemd/system")
	return err == nil
}

func agentUser() string {
	if u, err := user.Current(); err == nil {
		return u.Username
	}

	return "unknown"
}

func distro() string {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return ""
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if name, ok := strings.CutPrefix(sc.Text(), "PRETTY_NAME="); ok {
			return strings.Trim(name, `"`)
		}
	}

	return ""
}

func diskUsage(path string) (total, free uint64) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, 0
	}

	bsize := uint64(st.Bsize)
	return st.Blocks * bsize, st.Bavail * bsize
}

// memInfo parses /proc/meminfo (kB fields). On failure it returns zeros and
// the payload simply omits meaningful memory numbers.
func memInfo() (total, available uint64) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		key, value, ok := strings.Cut(sc.Text(), ":")
		if !ok {
			continue
		}

		fields := strings.Fields(value)
		if len(fields) == 0 {
			continue
		}

		kb, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			continue
		}

		switch strings.TrimSpace(key) {
		case "MemTotal":
			total = kb * 1024
		case "MemAvailable":
			available = kb * 1024
		}
	}

	return total, available
}
