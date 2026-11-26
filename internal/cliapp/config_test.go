package cliapp

import (
	"os"
	"path/filepath"
	"testing"
)

func writeToolingJSON(t *testing.T, root, content string) {
	t.Helper()
	dir := filepath.Join(root, ConfigDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ConfigFile), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadDirParsesArtifactSection(t *testing.T) {
	root := t.TempDir()
	writeToolingJSON(t, root, `{
	  "project": {"name": "clover2", "version": "1.2.3-rc.1"},
	  "version": {"exclude": ["third_party/*"]},
	  "artifact": {"endpoint": "https://minio.example.com:9000", "bucket": "drone-artifacts", "secure": false}
	}`)

	cfg, err := LoadDir(root)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Name != "clover2" || cfg.Version != "1.2.3-rc.1" {
		t.Errorf("project section not parsed: %+v", cfg)
	}
	if len(cfg.Exclude) != 1 || cfg.Exclude[0] != "third_party/*" {
		t.Errorf("version.exclude not parsed: %v", cfg.Exclude)
	}
	if cfg.Artifact.Endpoint != "https://minio.example.com:9000" {
		t.Errorf("artifact.endpoint = %q", cfg.Artifact.Endpoint)
	}
	if cfg.Artifact.Bucket != "drone-artifacts" {
		t.Errorf("artifact.bucket = %q", cfg.Artifact.Bucket)
	}
	if cfg.Artifact.Secure == nil || *cfg.Artifact.Secure != false {
		t.Errorf("artifact.secure = %v, want false", cfg.Artifact.Secure)
	}
}

func TestLoadDirMissingFileYieldsZeroConfig(t *testing.T) {
	cfg, err := LoadDir(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Artifact.Endpoint != "" || cfg.Artifact.Bucket != "" || cfg.Artifact.Secure != nil {
		t.Errorf("expected zero artifact config, got %+v", cfg.Artifact)
	}
}
