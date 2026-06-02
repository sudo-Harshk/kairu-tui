package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"kairu-tui/internal/backup"
	"kairu-tui/internal/config"
	"kairu-tui/internal/entries"
	"kairu-tui/internal/tui"
)

func main() {
	paths := config.DefaultPaths()

	exportPath := flag.String("export", "", "Export entries.json to the provided file path")
	importPath := flag.String("import", "", "Import entries from the provided file path into entries.json")
	backupPath := flag.String("backup", "", "Backup entries, templates, config, and notification queue to the provided file path")
	restorePath := flag.String("restore", "", "Restore entries, templates, config, and notification queue from the provided file path")
	flag.Parse()

	if *exportPath != "" && *importPath != "" {
		fmt.Println("Error: --export and --import cannot be used together.")
		os.Exit(1)
	}
	if *backupPath != "" && *restorePath != "" {
		fmt.Println("Error: --backup and --restore cannot be used together.")
		os.Exit(1)
	}
	if *backupPath != "" && (*exportPath != "" || *importPath != "") {
		fmt.Println("Error: --backup cannot be combined with --export or --import.")
		os.Exit(1)
	}
	if *restorePath != "" && (*exportPath != "" || *importPath != "") {
		fmt.Println("Error: --restore cannot be combined with --export or --import.")
		os.Exit(1)
	}

	if *exportPath != "" {
		if err := entries.ExportEntries(paths.DataFile, *exportPath); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		fmt.Println("Export complete:", *exportPath)
		return
	}
	if *importPath != "" {
		if err := entries.ImportEntries(paths.DataFile, *importPath); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		fmt.Println("Import complete:", *importPath)
		return
	}
	if *backupPath != "" {
		if err := backup.BackupProject(paths.DataFile, paths.TemplateFile, paths.ConfigFile, paths.OutboxFile, *backupPath); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		fmt.Println("Backup complete:", *backupPath)
		return
	}
	if *restorePath != "" {
		if err := backup.RestoreProject(paths.DataFile, paths.TemplateFile, paths.ConfigFile, paths.OutboxFile, *restorePath); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		fmt.Println("Restore complete:", *restorePath)
		return
	}

	// Create and run the TUI program
	p := tea.NewProgram(tui.New(paths), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
