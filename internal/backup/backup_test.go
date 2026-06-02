package backup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"kairu-tui/internal/entries"
	"kairu-tui/internal/notification"
	"kairu-tui/internal/templates"
)

func TestBackupAndRestoreProject(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dataFile := filepath.Join(dir, "entries.json")
	templateFile := filepath.Join(dir, "templates.json")
	configFile := filepath.Join(dir, "kairu.yaml")
	outboxFile := filepath.Join(dir, "notification_outbox.json")
	backupFile := filepath.Join(dir, "backup.json")

	entriesList := []entries.Entry{
		{Task: "Focus", Start: time.Date(2024, 6, 1, 9, 0, 0, 0, time.Local), End: time.Date(2024, 6, 1, 9, 25, 0, 0, time.Local), Duration: 1500, Type: "work"},
	}
	templatesList := []templates.SessionTemplate{{Name: "Deep Work", Task: "Focus", Duration: "25"}}
	configYAML := []byte("work_duration: 45\ntheme: ember\n")
	outbox := []notification.NotificationJob{{ID: "queued", Event: "session_end", Title: "Done", Body: "Session complete"}}

	if data, err := json.MarshalIndent(entriesList, "", "  "); err != nil {
		t.Fatalf("marshal entries failed: %v", err)
	} else if err := os.WriteFile(dataFile, data, 0644); err != nil {
		t.Fatalf("write entries failed: %v", err)
	}
	if err := templates.SaveSessionTemplates(templateFile, templatesList); err != nil {
		t.Fatalf("write templates failed: %v", err)
	}
	if err := os.WriteFile(configFile, configYAML, 0644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}
	if data, err := json.MarshalIndent(outbox, "", "  "); err != nil {
		t.Fatalf("marshal outbox failed: %v", err)
	} else if err := os.WriteFile(outboxFile, data, 0644); err != nil {
		t.Fatalf("write outbox failed: %v", err)
	}

	if err := BackupProject(dataFile, templateFile, configFile, outboxFile, backupFile); err != nil {
		t.Fatalf("BackupProject failed: %v", err)
	}

	if err := os.WriteFile(dataFile, []byte("[]"), 0644); err != nil {
		t.Fatalf("reset entries failed: %v", err)
	}
	if err := os.WriteFile(templateFile, []byte("[]"), 0644); err != nil {
		t.Fatalf("reset templates failed: %v", err)
	}
	if err := os.WriteFile(configFile, []byte("work_duration: 5\n"), 0644); err != nil {
		t.Fatalf("reset config failed: %v", err)
	}
	if err := os.WriteFile(outboxFile, []byte("[]"), 0644); err != nil {
		t.Fatalf("reset outbox failed: %v", err)
	}

	if err := RestoreProject(dataFile, templateFile, configFile, outboxFile, backupFile); err != nil {
		t.Fatalf("RestoreProject failed: %v", err)
	}

	gotEntries, err := entries.LoadEntries(dataFile)
	if err != nil {
		t.Fatalf("LoadEntries failed: %v", err)
	}
	if len(gotEntries) != 1 || gotEntries[0].Task != "Focus" {
		t.Fatalf("unexpected restored entries: %+v", gotEntries)
	}

	gotTemplates, err := templates.LoadSessionTemplates(templateFile)
	if err != nil {
		t.Fatalf("LoadSessionTemplates failed: %v", err)
	}
	if len(gotTemplates) != 1 || gotTemplates[0].Name != "Deep Work" {
		t.Fatalf("unexpected restored templates: %+v", gotTemplates)
	}

	gotConfig, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("read restored config failed: %v", err)
	}
	if !strings.Contains(string(gotConfig), "theme: ember") {
		t.Fatalf("unexpected restored config: %s", string(gotConfig))
	}

	gotOutbox, err := notification.LoadNotificationOutbox(outboxFile)
	if err != nil {
		t.Fatalf("LoadNotificationOutbox failed: %v", err)
	}
	if len(gotOutbox) != 1 || gotOutbox[0].ID != "queued" {
		t.Fatalf("unexpected restored outbox: %+v", gotOutbox)
	}
}
