package system

import (
	"bufio"
	"os"
	"os/user"
	"strconv"
	"strings"
	"syscall"
)

func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}

	return h
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
