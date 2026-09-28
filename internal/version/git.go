package version

import (
	"fmt"
	"os/exec"
	"strings"
)

// git runs git in dir (git itself searches parent dirs for the repo root).
func git(dir string, args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if strings.Contains(msg, "not a git repository") {
			return "", errNotRepo
		}

		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}

	return strings.TrimSpace(string(out)), nil
}

var errNotRepo = fmt.Errorf("not a git repository")

func gitTags(dir string) ([]string, error) {
	out, err := git(dir, "for-each-ref", "refs/tags", "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}

	if out == "" {
		return nil, nil
	}

	return strings.Split(out, "\n"), nil
}

func gitCommitOf(dir, ref string) (string, error) {
	return git(dir, "rev-parse", ref+"^{commit}")
}

func gitHead(dir string) (string, error) {
	return git(dir, "rev-parse", "HEAD")
}

// gitCommitsBehind counts commits since the newest v* tag (git describe
// --long); 0 when there are no tags.
func gitCommitsBehind(dir string) int {
	out, err := git(dir, "describe", "--tags", "--long", "--match", "v*")
	if err != nil {
		return 0
	}

	parts := strings.Split(out, "-")
	if len(parts) < 3 {
		return 0
	}

	var n int
	if _, err := fmt.Sscanf(parts[len(parts)-2], "%d", &n); err != nil {
		return 0
	}

	return n
}

// gitExactTag returns the tag directly on HEAD (any name - a non-v tag must
// surface as an error upstream, matching the old CLI), or "".
func gitExactTag(dir string) string {
	out, err := git(dir, "describe", "--tags", "--exact-match", "HEAD")
	if err != nil {
		return ""
	}

	return out
}

func gitIsDirty(dir string) bool {
	out, err := git(dir, "status", "--porcelain")
	return err == nil && out != ""
}
