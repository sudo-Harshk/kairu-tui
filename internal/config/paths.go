package config

type Paths struct {
	DataFile     string
	TemplateFile string
	ConfigFile   string
	OutboxFile   string
	PetFile      string
}

func DefaultPaths() Paths {
	return Paths{
		DataFile:     "entries.json",
		TemplateFile: "templates.json",
		ConfigFile:   "kairu.yaml",
		OutboxFile:   "notification_outbox.json",
		PetFile:      "pet.json",
	}
}

func DefaultBackupFile() string {
	return "backup.json"
}
