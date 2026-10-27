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
	RootEnv    = "CLOVER2_ROOT"
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
	Agent struct {
		URL       string // remote agent base URL, e.g. http://192.168.1.10:7550
		TokenFile string // bearer token file for the agent
	}
}

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
		Agent struct {
			URL       string `json:"url"`
			TokenFile string `json:"token_file"`
		} `json:"agent"`
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
		Artifact: struct {
			Endpoint string
			Bucket   string
			Secure   *bool
		}{
			Endpoint: raw.Artifact.Endpoint,
			Bucket:   raw.Artifact.Bucket,
			Secure:   raw.Artifact.Secure,
		},
		Agent: struct {
			URL       string
			TokenFile string
		}{
			URL:       raw.Agent.URL,
			TokenFile: raw.Agent.TokenFile,
		},
	}, nil
}

func ConfigPath(root string) string {
	return filepath.Join(root, ConfigDir, ConfigFile)
}

var ErrNoProject = errors.New("no tooling/tooling.json found (use --root or set " + RootEnv + ")")
