package cliapp

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	ConfigDir  = "tooling"
	ConfigFile = "tooling.json"
	RootEnv    = "CLOVER2_CLI_ROOT"
)

// Config is the typed view of tooling/tooling.json. Unknown keys are ignored.
type Config struct {
	Name     string   // [project] name
	Version  string   // [project] version: the canonical version
	Exclude  []string // [version] exclude: glob list for store scanning
	Artifact struct {
		Endpoint string
		Bucket   string
		Secure   *bool // nil means true
	}
}

// Load finds the project root (--root, $CLOVER2_CLI_ROOT, or the nearest parent
// directory containing tooling/tooling.json) and parses its config. An empty
// root with a zero Config means "no project" — commands that need one error out.
func Load() (string, Config, error) {
	if RootOverride != "" {
		return loadExplicit(RootOverride)
	}

	if env := os.Getenv(RootEnv); env != "" {
		return loadExplicit(env)
	}

	dir, err := os.Getwd()
	if err != nil {
		return "", Config{}, err
	}

	for {
		if isProject(dir) {
			cfg, err := parse(ConfigPath(dir))
			return dir, cfg, err
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", Config{}, nil
		}

		dir = parent
	}
}

func loadExplicit(dir string) (string, Config, error) {
	if !isProject(dir) {
		return "", Config{}, ExitErrorf(ExitNoProject, "'%s' has no %s/%s", dir, ConfigDir, ConfigFile)
	}

	cfg, err := parse(ConfigPath(dir))
	return dir, cfg, err
}

// LoadDir parses the config under dir; a missing file yields a zero Config.
// Used by commands that may create the config (version init).
func LoadDir(dir string) (Config, error) {
	if !isProject(dir) {
		return Config{}, nil
	}

	return parse(ConfigPath(dir))
}

func isProject(dir string) bool {
	_, err := os.Stat(ConfigPath(dir))
	return err == nil
}

func parse(path string) (Config, error) {
	var raw struct {
		Project struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"project"`
		Version struct {
			Exclude []string `json:"exclude"`
		} `json:"version"`
		Artifact struct {
			Endpoint string `json:"endpoint"`
			Bucket   string `json:"bucket"`
			Secure   *bool  `json:"secure"`
		} `json:"artifact"`
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return Config{}, fmt.Errorf("%s: invalid JSON: %w", path, err)
	}

	return Config{
		Name:    raw.Project.Name,
		Version: raw.Project.Version,
		Exclude: raw.Version.Exclude,
	}, nil
}

// ConfigPath returns the config file path under a project root.
func ConfigPath(root string) string {
	return filepath.Join(root, ConfigDir, ConfigFile)
}

var ErrNoProject = errors.New("no tooling/tooling.json found (use --root or set " + RootEnv + ")")
