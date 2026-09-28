package version

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"

	"github.com/klever-coex/clover2-cli/internal/cliapp"
)

const tagPrefix = "v"

func parseTag(tag string) (semver.Version, error) {
	if !strings.HasPrefix(tag, tagPrefix) {
		return semver.Version{}, fmt.Errorf("tag '%s' must start with 'v'", tag)
	}

	v, err := parseStrict(strings.TrimPrefix(tag, tagPrefix))
	if err != nil {
		return semver.Version{}, fmt.Errorf("tag '%s' is not a valid semver version", tag)
	}

	return v, nil
}

func initCanon(root string, cfg cliapp.Config, from string) error {
	if cfg.Version != "" {
		return fmt.Errorf("canonical version already set to %s (use 'version set' to change)", cfg.Version)
	}

	stores, err := Discover(root, cfg.Exclude)
	if err != nil {
		return err
	}

	src, err := FindStore(stores, from)
	if err != nil {
		return err
	}

	v, err := src.Read()
	if err != nil {
		return err
	}

	return writeCanonVersion(root, bare(v).String())
}

func requireCanon(cfg cliapp.Config) (semver.Version, error) {
	if cfg.Version == "" {
		return semver.Version{}, fmt.Errorf("no canonical version in tooling.json (project.version); run 'clover2 version init'")
	}
	return parseStrict(cfg.Version)
}

// writeCanonVersion sets project.version in tooling/tooling.json, preserving
// every other key (including unknown ones).
func writeCanonVersion(root, version string) error {
	path := cliapp.ConfigPath(root)

	var data map[string]any
	if raw, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(raw, &data); err != nil {
			return fmt.Errorf("%s: invalid JSON: %w", path, err)
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	if data == nil {
		data = map[string]any{}
	}

	project, _ := data["project"].(map[string]any)
	if project == nil {
		project = map[string]any{}
		data["project"] = project
	}

	project["version"] = version

	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	if err := os.WriteFile(path, append(out, '\n'), 0o644); err != nil {
		return err
	}

	slog.Info("canonical version set", "version", version, "file", path)
	return nil
}

// Bump bumps the canonical version and syncs all stores.
func Bump(root string, cfg cliapp.Config, field string) (map[string]any, error) {
	current, err := requireCanon(cfg)
	if err != nil {
		return nil, err
	}

	target, err := bumpField(current, field)
	if err != nil {
		return nil, err
	}

	if err := writeCanonVersion(root, target.String()); err != nil {
		return nil, err
	}

	stores, err := Discover(root, cfg.Exclude)
	if err != nil {
		return nil, err
	}

	if err := Sync(stores, target); err != nil {
		return nil, err
	}

	return map[string]any{
		"base":           target.String(),
		"tag":            tagPrefix + target.String(),
		"stores_changed": true,
	}, nil
}

func BumpRC(root string, cfg cliapp.Config, baseField string) (map[string]any, error) {
	current, err := requireCanon(cfg)
	if err != nil {
		return nil, err
	}

	rcNames, stableExists := []string{}, false
	tags, repoErr := gitTags(root)
	switch {
	case repoErr == nil:
		for _, name := range tags {
			if strings.HasPrefix(name, tagPrefix+current.String()+"-rc.") {
				rcNames = append(rcNames, name)
			}

			if name == tagPrefix+current.String() {
				stableExists = true
			}
		}
	case repoErr == errNotRepo:
		slog.Warn("not a git repo; treating as a fresh rc base", "version", current)
	default:
		return nil, repoErr
	}

	if len(rcNames) > 0 {
		next := 1
		for _, name := range rcNames {
			var n int
			if _, err := fmt.Sscanf(name, tagPrefix+current.String()+"-rc.%d", &n); err == nil && n >= next {
				next = n + 1
			}
		}

		return map[string]any{
			"base":           current.String(),
			"tag":            fmt.Sprintf("%s%s-rc.%d", tagPrefix, current, next),
			"stores_changed": false,
		}, nil
	}

	if stableExists {
		target, err := bumpField(current, baseField)
		if err != nil {
			return nil, err
		}

		if err := writeCanonVersion(root, target.String()); err != nil {
			return nil, err
		}

		stores, err := Discover(root, cfg.Exclude)
		if err != nil {
			return nil, err
		}

		if err := Sync(stores, target); err != nil {
			return nil, err
		}

		return map[string]any{
			"base":           target.String(),
			"tag":            fmt.Sprintf("%s%s-rc.1", tagPrefix, target),
			"stores_changed": true,
		}, nil
	}

	return map[string]any{
		"base":           current.String(),
		"tag":            fmt.Sprintf("%s%s-rc.1", tagPrefix, current),
		"stores_changed": false,
	}, nil
}

func Compose(root string, cfg cliapp.Config, ref, mode string, latestRC, latestStable bool) (map[string]any, error) {
	if _, err := gitHead(root); err != nil {
		return nil, fmt.Errorf("'%s': %w", root, err)
	}

	if latestRC || latestStable {
		return latestTag(root, latestRC)
	}

	base, err := requireCanon(cfg)
	if err != nil {
		return nil, err
	}

	if ref != "" && !strings.HasPrefix(ref, "refs/") {
		if !strings.HasPrefix(ref, tagPrefix) {
			return nil, fmt.Errorf("--ref must be a full git ref or a v-prefixed tag")
		}
		ref = "refs/tags/" + ref
	}

	if ref != "" && strings.HasPrefix(ref, "refs/tags/") {
		tagName := strings.TrimPrefix(ref, "refs/tags/")
		version, err := parseTag(tagName)
		if err != nil {
			return nil, err
		}

		tags, _ := gitTags(root)
		if !contains(tags, tagName) {
			return nil, fmt.Errorf("tag '%s' not found in repository", tagName)
		}

		hash, err := gitCommitOf(root, tagName)
		if err != nil {
			return nil, err
		}

		buildMode := "release"
		if version.Prerelease() != "" {
			buildMode = "pre-release"
		}

		return map[string]any{
			"base_version": base.String(),
			"git_hash":     short(hash),
			"build_mode":   buildMode,
			"version":      version.String(),
		}, nil
	}

	var hash string
	if ref != "" {
		h, err := gitCommitOf(root, ref)
		if err != nil {
			slog.Warn("ref not found; using HEAD", "ref", ref)
		} else {
			hash = h
		}
	}

	if hash == "" {
		h, err := gitHead(root)
		if err != nil {
			return nil, err
		}
		hash = h
	}

	if mode == "" {
		mode = "develop"
		if ref == "refs/heads/master" {
			mode = "master"
		}
	}

	version, err := versionForMode(root, base, mode, short(hash))
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"base_version": base.String(),
		"git_hash":     short(hash),
		"build_mode":   mode,
		"version":      version,
	}, nil
}

func latestTag(root string, rc bool) (map[string]any, error) {
	tags, err := gitTags(root)
	if err != nil {
		return nil, err
	}
	var best struct {
		name string
		v    semver.Version
	}
	for _, name := range tags {
		if !strings.HasPrefix(name, tagPrefix) {
			continue
		}

		if rc && !strings.Contains(name, "-rc.") {
			continue
		}

		if !rc && strings.Contains(name, "-") {
			continue
		}

		v, err := parseTag(name)
		if err != nil {
			continue
		}

		if best.name == "" || v.Compare(&best.v) > 0 {
			best.name, best.v = name, v
		}
	}
	if best.name == "" {
		what := "stable"
		if rc {
			what = "rc"
		}

		return nil, fmt.Errorf("no %s tags found", what)
	}

	version := best.v
	if rc {
		version = bare(version)
	}

	hash, err := gitCommitOf(root, best.name)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"tag":     best.name,
		"version": version.String(),
		"commit":  hash,
	}, nil
}

func versionForMode(root string, base semver.Version, mode, hash string) (string, error) {
	if mode == "develop" || mode == "master" {
		build := hash
		if gitIsDirty(root) {
			build += ".dirty"
		}

		return fmt.Sprintf("%s-dev.%d+%s", base, gitCommitsBehind(root), build), nil
	}

	tag := gitExactTag(root)
	if tag != "" {
		version, err := parseTag(tag)
		if err != nil {
			return "", err
		}

		if (mode == "release") == (version.Prerelease() == "") {
			return version.String(), nil
		}

		slog.Warn("tag does not match mode; using bare base version", "tag", tag, "mode", mode)
	} else {
		slog.Warn("no tag on HEAD; build uses the bare base version", "mode", mode)
	}

	return base.String(), nil
}

func short(hash string) string {
	if len(hash) > 7 {
		return hash[:7]
	}

	return hash
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}

	return false
}
