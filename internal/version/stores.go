package version

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// Each store edits its file surgically - no TOML/XML round-tripping,
// the version value is replaced in place.

type pyProjectStore struct{ path string }

func newPyProjectStore(path string) Store { return pyProjectStore{path} }

// findKey returns the value and line index of `key = "value"` inside the
// given TOML section (e.g. "[project]", "tool.poetry").
func (s pyProjectStore) findKey(header, key string) (string, int, bool) {
	lines := fileLines(s.path)
	inHeader := false

	for i, line := range lines {
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}

		if strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]") {
			inHeader = strings.Trim(t, "[]") == strings.Trim(header, "[]")
			continue
		}

		if !inHeader {
			continue
		}

		if m := tomlKeyRe(key).FindStringSubmatch(t); m != nil {
			return m[1], i, true
		}
	}

	return "", -1, false
}

// versionHeader picks the section carrying the version: [project] first,
// then [tool.poetry].
func (s pyProjectStore) versionHeader() string {
	if _, _, ok := s.findKey("[project]", "version"); ok {
		return "[project]"
	}

	return "tool.poetry"
}

func (s pyProjectStore) Name() string {
	if n, _, ok := s.findKey("[project]", "name"); ok && n != "" {
		return n
	}

	if n, _, ok := s.findKey("tool.poetry", "name"); ok && n != "" {
		return n
	}

	return filepath.Base(filepath.Dir(s.path))
}

func (s pyProjectStore) Read() (semver.Version, error) {
	raw, _, ok := s.findKey(s.versionHeader(), "version")
	if !ok {
		return semver.Version{}, fmt.Errorf("no version field in %s", s.path)
	}

	v, err := parseStrict(raw)
	if err != nil {
		return semver.Version{}, fmt.Errorf("invalid version in %s: %s", s.path, raw)
	}

	return v, nil
}

func (s pyProjectStore) Write(v semver.Version) error {
	_, idx, ok := s.findKey(s.versionHeader(), "version")
	if !ok {
		return fmt.Errorf("no version field in %s", s.path)
	}

	return replaceTomlValue(s.path, idx, "version", bare(v).String())
}

func (s pyProjectStore) Reference() string { return "pyproject::" + s.Name() }

var tomlHeaderRe = regexp.MustCompile(`^\[[^\]]*\]$`)

func tomlKeyRe(key string) *regexp.Regexp {
	return regexp.MustCompile(`^\s*` + key + `\s*=\s*"([^"]*)"`)
}

// replaceTomlValue rewrites `key = "..."` on line idx, keeping the rest of
// the line (comments, spacing) intact.
func replaceTomlValue(path string, idx int, key, newValue string) error {
	lines := fileLines(path)
	if idx < 0 || idx >= len(lines) {
		return fmt.Errorf("%s: line %d out of range", path, idx+1)
	}

	re := regexp.MustCompile(`^(\s*` + key + `\s*=\s*")([^"]*)(")`)
	if !re.MatchString(lines[idx]) {
		return fmt.Errorf("%s: cannot rewrite line %d", path, idx+1)
	}

	lines[idx] = re.ReplaceAllString(lines[idx], `${1}`+newValue+`${3}`)
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
}

// --- package.json ---

type npmStore struct{ path string }

func newNpmStore(path string) Store { return npmStore{path} }

func (s npmStore) Name() string {
	var data struct {
		Name string `json:"name"`
	}
	if json.Unmarshal(readFile(s.path), &data) == nil && data.Name != "" {
		return data.Name
	}

	return filepath.Base(filepath.Dir(s.path))
}

func (s npmStore) Read() (semver.Version, error) {
	var data struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(readFile(s.path), &data); err != nil {
		return semver.Version{}, fmt.Errorf("invalid JSON in %s: %w", s.path, err)
	}

	if data.Version == "" {
		return semver.Version{}, fmt.Errorf("no version field in %s", s.path)
	}

	v, err := parseStrict(data.Version)
	if err != nil {
		return semver.Version{}, fmt.Errorf("invalid version in %s: %s", s.path, data.Version)
	}

	return v, nil
}

var npmVersionRe = regexp.MustCompile(`("version"\s*:\s*")([^"]*)(")`)

func (s npmStore) Write(v semver.Version) error {
	text := string(readFile(s.path))
	loc := npmVersionRe.FindStringSubmatchIndex(text)
	if loc == nil {
		return fmt.Errorf("no version field in %s", s.path)
	}

	// splice the first match only: replace exactly the captured value (group 2),
	// nested "version" keys stay untouched
	var b strings.Builder
	b.WriteString(text[:loc[4]])
	b.WriteString(bare(v).String())
	b.WriteString(text[loc[5]:])

	return os.WriteFile(s.path, []byte(b.String()), 0o644)
}

func (s npmStore) Reference() string { return "npm::" + s.Name() }

// --- galaxy.yml (ansible collection) ---

type galaxyStore struct{ path string }

func newGalaxyStore(path string) Store { return galaxyStore{path} }

var (
	galaxyVersionRe = regexp.MustCompile(`(?m)^(version\s*:\s*)("[^"]*"|'[^']*'|[^\s"'#]+)(\s*(?:#.*)?)$`)
	galaxyNameRe    = regexp.MustCompile(`(?m)^name\s*:\s*("[^"]*"|'[^']*'|[^\s"'#]+)`)
)

// yamlQuoteStyle returns the quote character of the original value (""
// when unquoted) so Write can preserve it.
func yamlQuoteStyle(s string) string {
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') {
		return s[:1]
	}

	return ""
}

func unquoteYaml(s string) string {
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}

	return s
}

func (s galaxyStore) Name() string {
	if m := galaxyNameRe.FindStringSubmatch(string(readFile(s.path))); m != nil {
		return unquoteYaml(m[1])
	}

	return filepath.Base(filepath.Dir(s.path))
}

func (s galaxyStore) Read() (semver.Version, error) {
	m := galaxyVersionRe.FindStringSubmatch(string(readFile(s.path)))
	if m == nil {
		return semver.Version{}, fmt.Errorf("no version field in %s", s.path)
	}

	raw := unquoteYaml(m[2])
	v, err := parseStrict(raw)
	if err != nil {
		return semver.Version{}, fmt.Errorf("invalid version in %s: %s", s.path, raw)
	}

	return v, nil
}

func (s galaxyStore) Write(v semver.Version) error {
	text := string(readFile(s.path))
	m := galaxyVersionRe.FindStringSubmatch(text)
	if m == nil {
		return fmt.Errorf("no version field in %s", s.path)
	}

	q := yamlQuoteStyle(m[2])
	newText := galaxyVersionRe.ReplaceAllString(text, "${1}"+q+bare(v).String()+q+"${3}")
	return os.WriteFile(s.path, []byte(newText), 0o644)
}

func (s galaxyStore) Reference() string { return "galaxy::" + s.Name() }

// --- package.xml (ROS) ---

type rosStore struct{ path string }

func newRosStore(path string) Store { return rosStore{path} }

var (
	rosVersionRe = regexp.MustCompile(`(?m)^(\s*)<version>([^<]*)</version>\s*$`)
	rosNameRe    = regexp.MustCompile(`<name>([^<]*)</name>`)
)

func (s rosStore) Name() string {
	if m := rosNameRe.FindStringSubmatch(string(readFile(s.path))); m != nil {
		return strings.TrimSpace(m[1])
	}

	return filepath.Base(filepath.Dir(s.path))
}

func (s rosStore) Read() (semver.Version, error) {
	m := rosVersionRe.FindStringSubmatch(string(readFile(s.path)))
	if m == nil {
		return semver.Version{}, fmt.Errorf("no version field in %s", s.path)
	}

	v, err := parseStrict(strings.TrimSpace(m[2]))
	if err != nil {
		return semver.Version{}, fmt.Errorf("invalid version in %s: %s", s.path, m[2])
	}

	return v, nil
}

func (s rosStore) Write(v semver.Version) error {
	text := string(readFile(s.path))
	if !rosVersionRe.MatchString(text) {
		return fmt.Errorf("no version field in %s", s.path)
	}

	newText := rosVersionRe.ReplaceAllString(text, `${1}<version>`+bare(v).String()+`</version>`)
	return os.WriteFile(s.path, []byte(newText), 0o644)
}

func (s rosStore) Reference() string { return "ros::" + s.Name() }

func fileLines(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	return strings.Split(string(data), "\n")
}

func readFile(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	return data
}
