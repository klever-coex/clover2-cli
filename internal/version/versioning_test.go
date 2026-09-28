package version

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Masterminds/semver/v3"

	"github.com/klever-coex/clover2-cli/internal/cliapp"
)

// setupProject creates a temp project: tooling.json with the canonical
// version, a ros store, and a git repo with stable tag v1.2.3 and rc tag
// v1.2.3-rc.1.
func setupProject(t *testing.T, canon string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, cliapp.ConfigDir), 0o755); err != nil {
		t.Fatal(err)
	}

	cfg, err := json.Marshal(map[string]any{
		"project": map[string]any{"name": "test", "version": canon},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(cliapp.ConfigPath(dir), cfg, 0o644); err != nil {
		t.Fatal(err)
	}

	os.WriteFile(filepath.Join(dir, "package.xml"),
		[]byte("<package format=\"3\">\n  <name>clover2</name>\n  <version>"+canon+"</version>\n</package>\n"), 0o644)

	g := func(args ...string) {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", dir, "-c", "user.email=t@t", "-c", "user.name=t"}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
	}
	g("init", "-q")
	g("add", "-A")
	g("commit", "-q", "--allow-empty", "-m", "init")
	g("tag", "v1.2.3")
	g("commit", "-q", "--allow-empty", "-m", "rc work")
	g("tag", "v1.2.3-rc.1")
	g("commit", "-q", "--allow-empty", "-m", "post-rc work")
	return dir
}

func mustCfg(t *testing.T, root string) cliapp.Config {
	t.Helper()
	cfg, err := cliapp.LoadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Version == "" {
		t.Fatalf("no canonical version in %s", cliapp.ConfigPath(root))
	}
	return cfg
}

func readCanon(t *testing.T, root string) string {
	t.Helper()
	return mustCfg(t, root).Version
}

func readStore(t *testing.T, root, file string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, file))
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`<version>([^<]*)</version>`).FindStringSubmatch(string(data))
	if m == nil {
		t.Fatalf("no version in %s", file)
	}
	return m[1]
}

func TestBumpPatch(t *testing.T) {
	root := setupProject(t, "1.2.3")
	payload, err := Bump(root, mustCfg(t, root), "patch")
	if err != nil {
		t.Fatal(err)
	}
	if payload["tag"] != "v1.2.4" || payload["stores_changed"] != true {
		t.Errorf("payload = %v", payload)
	}
	if canon := readCanon(t, root); canon != "1.2.4" {
		t.Errorf("canon = %q", canon)
	}
	if v := readStore(t, root, "package.xml"); v != "1.2.4" {
		t.Errorf("store = %q", v)
	}
}

func TestBumpRCSeries(t *testing.T) {
	root := setupProject(t, "1.2.3")

	// series exists: v1.2.3-rc.1 -> v1.2.3-rc.2, nothing written
	payload, err := BumpRC(root, mustCfg(t, root), "minor")
	if err != nil {
		t.Fatal(err)
	}
	if payload["tag"] != "v1.2.3-rc.2" || payload["stores_changed"] != false {
		t.Errorf("payload = %v", payload)
	}
	if v := readStore(t, root, "package.xml"); v != "1.2.3" {
		t.Errorf("store changed on rc advance: %q", v)
	}

	// stable tag exists and the rc series is done: fresh series from bumped base
	if err := exec.Command("git", "-C", root, "tag", "-d", "v1.2.3-rc.1").Run(); err != nil {
		t.Fatal(err)
	}
	payload, err = BumpRC(root, mustCfg(t, root), "minor")
	if err != nil {
		t.Fatal(err)
	}
	if payload["tag"] != "v1.3.0-rc.1" || payload["stores_changed"] != true {
		t.Errorf("payload = %v", payload)
	}
	if v := readStore(t, root, "package.xml"); v != "1.3.0" {
		t.Errorf("store = %q", v)
	}
}

func TestComposeDevelop(t *testing.T) {
	root := setupProject(t, "1.2.3")
	payload, err := Compose(root, mustCfg(t, root), "", "", false, false)
	if err != nil {
		t.Fatal(err)
	}
	v := payload["version"].(string)
	if !strings.HasPrefix(v, "1.2.3-dev.1+") {
		t.Errorf("version = %q", v)
	}
	if payload["build_mode"] != "develop" {
		t.Errorf("build_mode = %v", payload["build_mode"])
	}
}

func TestComposeLatestTags(t *testing.T) {
	root := setupProject(t, "1.2.3")

	payload, err := Compose(root, mustCfg(t, root), "", "", true, false)
	if err != nil {
		t.Fatal(err)
	}
	if payload["tag"] != "v1.2.3-rc.1" || payload["version"] != "1.2.3" {
		t.Errorf("latest-rc payload = %v", payload)
	}

	payload, err = Compose(root, mustCfg(t, root), "", "", false, true)
	if err != nil {
		t.Fatal(err)
	}
	if payload["tag"] != "v1.2.3" || payload["version"] != "1.2.3" {
		t.Errorf("latest-stable payload = %v", payload)
	}
}

func TestComposeRefTag(t *testing.T) {
	root := setupProject(t, "1.2.3")
	payload, err := Compose(root, mustCfg(t, root), "v1.2.3-rc.1", "", false, false)
	if err != nil {
		t.Fatal(err)
	}
	if payload["version"] != "1.2.3-rc.1" || payload["build_mode"] != "pre-release" {
		t.Errorf("payload = %v", payload)
	}
	if h := payload["git_hash"].(string); len(h) != 7 {
		t.Errorf("git_hash = %q", h)
	}
}

func TestComposeReleaseModeRejectsNonVTag(t *testing.T) {
	root := setupProject(t, "1.2.3")
	if err := exec.Command("git", "-C", root, "tag", "not-a-version").Run(); err != nil {
		t.Fatal(err)
	}
	_, err := Compose(root, mustCfg(t, root), "", "release", false, false)
	if err == nil || !strings.Contains(err.Error(), "must start with 'v'") {
		t.Errorf("err = %v, want non-v tag rejection", err)
	}
}

func TestChangedStores(t *testing.T) {
	dir := t.TempDir()
	write := func(file, v string) Store {
		path := filepath.Join(dir, file)
		os.WriteFile(path, []byte("<package>\n  <name>"+file+"</name>\n  <version>"+v+"</version>\n</package>\n"), 0o644)
		return newRosStore(path)
	}
	stores := []Store{write("b.xml", "2.0.0"), write("a.xml", "1.0.0")}

	changed, err := changedStores(stores, parseStrict2(t, "1.0.0"))
	if err != nil {
		t.Fatal(err)
	}
	if len(changed) != 1 || changed[0] != "ros::b.xml" {
		t.Errorf("changed = %v, want [ros::b.xml]", changed)
	}
}

func parseStrict2(t *testing.T, s string) semver.Version {
	t.Helper()
	v, err := semver.StrictNewVersion(s)
	if err != nil {
		t.Fatal(err)
	}
	return *v
}

func TestWriteCanonVariants(t *testing.T) {
	// creates the file when it does not exist
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, cliapp.ConfigDir), 0o755)
	if err := writeCanonVersion(dir, "3.2.1"); err != nil {
		t.Fatal(err)
	}
	if cfg, err := cliapp.LoadDir(dir); err != nil || cfg.Version != "3.2.1" {
		t.Errorf("created config: cfg = %+v, err = %v", cfg, err)
	}

	// preserves unrelated keys, including unknown ones
	dir2 := t.TempDir()
	os.MkdirAll(filepath.Join(dir2, cliapp.ConfigDir), 0o755)
	src := `{"project": {"name": "t"}, "custom": {"keep": true}}`
	if err := os.WriteFile(cliapp.ConfigPath(dir2), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeCanonVersion(dir2, "3.2.1"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(cliapp.ConfigPath(dir2))
	for _, want := range []string{`"version": "3.2.1"`, `"name": "t"`, `"custom"`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("missing %s in:\n%s", want, data)
		}
	}
}
