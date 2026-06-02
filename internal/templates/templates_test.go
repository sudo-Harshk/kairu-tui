package templates

import (
	"path/filepath"
	"testing"
)

func TestLoadAndSaveSessionTemplates(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "templates.json")

	templates := []SessionTemplate{
		{Name: "Deep Work", Task: "Deep Work", Duration: "25", Note: "Focus block", Tags: []string{"deep work", "writing"}},
	}
	if err := SaveSessionTemplates(path, templates); err != nil {
		t.Fatalf("SaveSessionTemplates failed: %v", err)
	}

	got, err := LoadSessionTemplates(path)
	if err != nil {
		t.Fatalf("LoadSessionTemplates failed: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 template, got %d", len(got))
	}
	if got[0].Name != "Deep Work" || got[0].Duration != "25" {
		t.Fatalf("unexpected template contents: %+v", got[0])
	}
}
