package version

import (
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// Store edits one kind of version file. Write must be surgical
// (regex/line replacement), preserving all other formatting.
type Store interface {
	Name() string
	// Reference is "<store-type>::<name>", e.g. "ros::clover2".
	Reference() string
	Read() (semver.Version, error)
	Write(semver.Version) error
}

var storesByFile = map[string]func(path string) Store{
	"pyproject.toml": newPyProjectStore,
	"package.json":   newNpmStore,
	"galaxy.yml":     newGalaxyStore,
	"package.xml":    newRosStore,
}

var skipDirs = map[string]bool{
	"node_modules": true, ".git": true, "build": true, "dist": true,
	".venv": true, "venv": true, "install": true,
}

// Discover skips stores without a readable version (missing field, bad
// version) - matching the old CLI. Excludes are globs matched against the
// relative dir path or any of its parts.
func Discover(root string, exclude []string) ([]Store, error) {
	var stores []Store

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if path == root {
				return nil
			}
			if skipDirs[d.Name()] || excluded(path, root, exclude) {
				return filepath.SkipDir
			}

			return nil
		}

		newStore, ok := storesByFile[d.Name()]
		if !ok {
			return nil
		}

		s := newStore(path)
		v, err := s.Read()
		if err != nil {
			slog.Debug("skipping store", "path", path, "reason", err)
			return nil
		}

		slog.Debug("found store", "reference", s.Reference(), "version", v)
		stores = append(stores, s)
		return nil
	})

	return stores, err
}

func excluded(path, root string, exclude []string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}

	rel = filepath.ToSlash(rel)
	for _, glob := range exclude {
		if ok, _ := filepath.Match(glob, rel); ok {
			return true
		}

		for _, part := range strings.Split(rel, "/") {
			if ok, _ := filepath.Match(glob, part); ok {
				return true
			}
		}
	}

	return false
}

func FindStore(stores []Store, ref string) (Store, error) {
	for _, s := range stores {
		if s.Reference() == ref {
			return s, nil
		}
	}

	return nil, fmt.Errorf("store '%s' not found", ref)
}

func Sync(stores []Store, v semver.Version) error {
	for _, s := range stores {
		old, err := s.Read()
		if err != nil {
			return err
		}

		if err := s.Write(v); err != nil {
			return fmt.Errorf("%s: %w", s.Reference(), err)
		}

		slog.Info("updated store", "reference", s.Reference(), "from", old, "to", v)
	}

	return nil
}

// bare strips prerelease/build with a warning: stores keep bare versions only.
func bare(v semver.Version) semver.Version {
	if v.Prerelease() != "" || v.Metadata() != "" {
		slog.Warn("version carries a suffix; stores keep bare versions only", "version", v)
	}

	return mkVersion(v.Major(), v.Minor(), v.Patch())
}

func mkVersion(major, minor, patch uint64) semver.Version {
	v := semver.MustParse(fmt.Sprintf("%d.%d.%d", major, minor, patch))
	return *v
}

// bumpField clears prerelease/build on the way up (old CLI semantics).
func bumpField(v semver.Version, field string) (semver.Version, error) {
	switch field {
	case "major":
		return mkVersion(v.Major()+1, 0, 0), nil
	case "minor":
		return mkVersion(v.Major(), v.Minor()+1, 0), nil
	case "patch":
		return mkVersion(v.Major(), v.Minor(), v.Patch()+1), nil
	}

	return v, fmt.Errorf("invalid bump field '%s' (want major, minor or patch)", field)
}

func parseStrict(s string) (semver.Version, error) {
	v, err := semver.StrictNewVersion(s)
	if err != nil {
		return semver.Version{}, err
	}

	return *v, nil
}
