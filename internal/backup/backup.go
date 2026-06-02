package backup

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"kairu-tui/internal/entries"
	"kairu-tui/internal/notification"
	"kairu-tui/internal/templates"
)

type ProjectBackup struct {
	Version            int                            `json:"version"`
	CreatedAt          time.Time                      `json:"created_at"`
	Entries            []entries.Entry                `json:"entries"`
	Templates          []templates.SessionTemplate    `json:"templates"`
	ConfigYAML         string                         `json:"config_yaml"`
	NotificationOutbox []notification.NotificationJob `json:"notification_outbox,omitempty"`
}

func LoadFileString(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

func BackupProject(dataFile, templateFile, configFile, outboxFile, backupPath string) error {
	entriesList, err := entries.LoadEntries(dataFile)
	if err != nil {
		return fmt.Errorf("failed to read entries: %w", err)
	}
	templatesList, err := templates.LoadSessionTemplates(templateFile)
	if err != nil {
		return fmt.Errorf("failed to read templates: %w", err)
	}
	configYAML, err := LoadFileString(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}
	outbox, err := notification.LoadNotificationOutbox(outboxFile)
	if err != nil {
		return fmt.Errorf("failed to read notification queue: %w", err)
	}
	projectBackup := ProjectBackup{
		Version:            1,
		CreatedAt:          time.Now().UTC(),
		Entries:            entriesList,
		Templates:          templatesList,
		ConfigYAML:         configYAML,
		NotificationOutbox: outbox,
	}
	data, err := json.MarshalIndent(projectBackup, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode backup: %w", err)
	}
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}
	return nil
}

func RestoreProject(dataFile, templateFile, configFile, outboxFile, backupPath string) error {
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}
	var projectBackup ProjectBackup
	if err := json.Unmarshal(data, &projectBackup); err != nil {
		return fmt.Errorf("failed to parse backup file: %w", err)
	}
	if projectBackup.Version != 1 {
		return fmt.Errorf("unsupported backup version: %d", projectBackup.Version)
	}
	if err := entries.ValidateEntries(projectBackup.Entries); err != nil {
		return fmt.Errorf("backup entries validation failed: %w", err)
	}
	entriesData, err := json.MarshalIndent(projectBackup.Entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode entries: %w", err)
	}
	if err := os.WriteFile(dataFile, entriesData, 0644); err != nil {
		return fmt.Errorf("failed to restore entries: %w", err)
	}
	templatesData, err := json.MarshalIndent(projectBackup.Templates, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode templates: %w", err)
	}
	if err := os.WriteFile(templateFile, templatesData, 0644); err != nil {
		return fmt.Errorf("failed to restore templates: %w", err)
	}
	if err := os.WriteFile(configFile, []byte(projectBackup.ConfigYAML), 0644); err != nil {
		return fmt.Errorf("failed to restore config: %w", err)
	}
	outboxData, err := json.MarshalIndent(projectBackup.NotificationOutbox, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode notification queue: %w", err)
	}
	if err := os.WriteFile(outboxFile, outboxData, 0644); err != nil {
		return fmt.Errorf("failed to restore notification queue: %w", err)
	}
	return nil
}
