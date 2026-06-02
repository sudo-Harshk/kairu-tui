package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultPaths(t *testing.T) {
	paths := DefaultPaths()
	if paths.DataFile == "" || paths.ConfigFile == "" || paths.TemplateFile == "" {
		t.Error("Expected paths to be non-empty")
	}
}

func TestEnsureDirsExist(t *testing.T) {
	tempDir := t.TempDir()
	paths := Paths{
		DataFile:     filepath.Join(tempDir, "data", "entries.json"),
		ConfigFile:   filepath.Join(tempDir, "config", "kairu.yaml"),
		TemplateFile: filepath.Join(tempDir, "data", "templates.json"),
	}

	err := paths.EnsureDirsExist()
	if err != nil {
		t.Fatalf("EnsureDirsExist failed: %v", err)
	}

	if _, err := os.Stat(filepath.Dir(paths.DataFile)); os.IsNotExist(err) {
		t.Error("Expected data directory to exist")
	}
	if _, err := os.Stat(filepath.Dir(paths.ConfigFile)); os.IsNotExist(err) {
		t.Error("Expected config directory to exist")
	}
}
