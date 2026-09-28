package version

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Masterminds/semver/v3"
)

// Golden tests: Read parses the testdata files, Write replaces the version
// in place, and the rest of the file must stay byte-identical.
func TestStoresRead(t *testing.T) {
	dir := "testdata"
	cases := []struct {
		file      string
		name      string
		reference string
		version   string
	}{
		{"pyproject.toml", "clover2-core", "pyproject::clover2-core", "1.2.3"},
		{"package.json", "@clover2/ui", "npm::@clover2/ui", "0.9.0"},
		{"galaxy.yml", "fleet", "galaxy::fleet", "2.0.0"},
		{"package.xml", "clover2-drone", "ros::clover2-drone", "1.2.3"},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			s := storesByFile[tc.file](filepath.Join(dir, tc.file))
			if s.Name() != tc.name {
				t.Errorf("name = %q, want %q", s.Name(), tc.name)
			}
			if s.Reference() != tc.reference {
				t.Errorf("reference = %q, want %q", s.Reference(), tc.reference)
			}
			v, err := s.Read()
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			if v.String() != tc.version {
				t.Errorf("version = %q, want %q", v.String(), tc.version)
			}
		})
	}
}

func TestStoresWritePreservesFormatting(t *testing.T) {
	cases := map[string]string{
		"pyproject.toml": "# comment preserved on write\n[project]\nname = \"clover2-core\"\nversion = \"9.9.9\"  # trailing comment kept\ndescription = \"Core package\"\n",
		"package.json":   "{\n  \"name\": \"@clover2/ui\",\n  \"version\": \"9.9.9\",\n  \"scripts\": { \"build\": \"vite build\" }\n}\n",
		"galaxy.yml":     "namespace: clover2\nname: fleet\nversion: 9.9.9\nauthors:\n  - klever\n",
		"package.xml":    "<?xml version=\"1.0\"?>\n<package format=\"3\">\n  <name>clover2-drone</name>\n  <version>9.9.9</version>\n  <description>ROS package</description>\n</package>\n",
	}
	for file, want := range cases {
		t.Run(file, func(t *testing.T) {
			dir := t.TempDir()
			src, err := os.ReadFile(filepath.Join("testdata", file))
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, file)
			if err := os.WriteFile(path, src, 0o644); err != nil {
				t.Fatal(err)
			}
			s := storesByFile[file](path)
			v := parseStrict9(t)
			if err := s.Write(v); err != nil {
				t.Fatalf("write: %v", err)
			}
			got, _ := os.ReadFile(path)
			if string(got) != want {
				t.Errorf("file after write:\n--- got ---\n%s\n--- want ---\n%s", got, want)
			}
		})
	}
}

func TestNpmWriteOnlyFirstVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "package.json")
	src := "{\n  \"name\": \"x\",\n  \"version\": \"1.0.0\",\n  \"overrides\": { \"left-pad\": { \"version\": \"2.0.0\" } }\n}\n"
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := newNpmStore(path).Write(parseStrict9(t)); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	want := "{\n  \"name\": \"x\",\n  \"version\": \"9.9.9\",\n  \"overrides\": { \"left-pad\": { \"version\": \"2.0.0\" } }\n}\n"
	if string(got) != want {
		t.Errorf("nested version rewritten:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestDiscoverySkipsBadAndMissingVersions(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "sub", "node_modules"), 0o755)
	os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[project]\nname = \"x\"\nversion = \"1.0.0\"\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "sub", "package.xml"),
		[]byte("<package>\n  <name>y</name>\n  <version>2.0.0</version>\n</package>\n"), 0o644)
	// no version field: skipped
	os.WriteFile(filepath.Join(dir, "sub", "package.json"), []byte("{\"name\": \"z\"}"), 0o644)
	// inside skipped dir: not found
	os.WriteFile(filepath.Join(dir, "sub", "node_modules", "package.xml"),
		[]byte("<package>\n  <name>n</name>\n  <version>3.0.0</version>\n</package>\n"), 0o644)

	stores, err := Discover(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(stores) != 2 {
		t.Fatalf("got %d stores, want 2", len(stores))
	}
	if stores[0].Reference() != "pyproject::x" || stores[1].Reference() != "ros::y" {
		t.Errorf("unexpected stores: %s, %s", stores[0].Reference(), stores[1].Reference())
	}
}

func TestDiscoveryExclude(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "external", "foo"), 0o755)
	os.WriteFile(filepath.Join(dir, "external", "foo", "package.xml"),
		[]byte("<package>\n  <name>e</name>\n  <version>1.0.0</version>\n</package>\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "package.xml"),
		[]byte("<package>\n  <name>main</name>\n  <version>1.0.0</version>\n</package>\n"), 0o644)

	stores, err := Discover(dir, []string{"external/*"})
	if err != nil {
		t.Fatal(err)
	}
	if len(stores) != 1 || stores[0].Reference() != "ros::main" {
		t.Errorf("exclude failed: %d stores", len(stores))
	}
}

func parseStrict9(t *testing.T) semver.Version {
	v, err := parseStrict("9.9.9")
	if err != nil {
		t.Fatal(err)
	}
	return v
}
