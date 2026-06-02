package config

import (
	"io"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
)

type Paths struct {
	DataFile          string
	TemplateFile      string
	ConfigFile        string
	OutboxFile        string
	PetFile           string
	TypingRecordsFile string
}

func DefaultPaths() Paths {
	return Paths{
		DataFile:          filepath.Join(xdg.DataHome, "kairu", "entries.json"),
		TemplateFile:      filepath.Join(xdg.DataHome, "kairu", "templates.json"),
		ConfigFile:        filepath.Join(xdg.ConfigHome, "kairu", "kairu.yaml"),
		OutboxFile:        filepath.Join(xdg.DataHome, "kairu", "notification_outbox.json"),
		PetFile:           filepath.Join(xdg.DataHome, "kairu", "pet.json"),
		TypingRecordsFile: filepath.Join(xdg.DataHome, "kairu", "typing_records.json"),
	}
}

func DefaultBackupFile() string {
	return filepath.Join(xdg.DataHome, "kairu", "backup.json")
}

// EnsureDirsExist ensures that the directories containing the paths exist.
func (p Paths) EnsureDirsExist() error {
	dirs := []string{
		filepath.Dir(p.DataFile),
		filepath.Dir(p.ConfigFile),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}

// MigrateFromCWD checks if old CWD files exist and moves them to XDG paths.
func MigrateFromCWD(paths Paths) error {
	migrationMap := map[string]string{
		"entries.json":             paths.DataFile,
		"templates.json":           paths.TemplateFile,
		"kairu.yaml":               paths.ConfigFile,
		"notification_outbox.json": paths.OutboxFile,
		"pet.json":                  paths.PetFile,
		"typing_records.json":       paths.TypingRecordsFile,
		"backup.json":               DefaultBackupFile(),
	}

	for oldName, newPath := range migrationMap {
		// Check if old file exists in CWD
		if _, err := os.Stat(oldName); err == nil {
			// Old file exists. Now check if new file already exists.
			if _, errNew := os.Stat(newPath); os.IsNotExist(errNew) {
				// Destination does not exist. Migrate it.
				// Ensure destination directory exists
				dir := filepath.Dir(newPath)
				if errDir := os.MkdirAll(dir, 0755); errDir != nil {
					return errDir
				}
				// Copy the file
				if errCopy := copyFile(oldName, newPath); errCopy != nil {
					return errCopy
				}
				// Delete old file
				if errRemove := os.Remove(oldName); errRemove != nil {
					return errRemove
				}
			}
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
