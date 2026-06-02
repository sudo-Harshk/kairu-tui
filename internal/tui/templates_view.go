package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"kairu-tui/internal/templates"
)

func renderTemplateManagerView(m model) string {
	footer := "[Tab/Arrows] Browse   [Enter] Apply   [Ctrl+T] Save current form   [Ctrl+R] Rename   [Ctrl+D] Delete   [Ctrl+Z] Undo delete   [Ctrl+Y] Duplicate   [Ctrl+P/Esc] Back   [?] Help   [q] Quit"
	errorLine := renderAppError(m)
	statusLine := renderNotificationStatus(m)
	if errorLine != "" {
		footer = fmt.Sprintf("%s\n%s", errorLine, footer)
	}
	if statusLine != "" {
		footer = fmt.Sprintf("%s\n%s", statusLine, footer)
	}

	listLines := []string{"Templates:"}
	if len(m.templates) == 0 {
		listLines = append(listLines, "  No templates saved yet.")
	} else {
		for i, template := range m.templates {
			prefix := "  "
			if i == m.templateIndex {
				prefix = "> "
			}
			tagStr := ""
			if len(template.Tags) > 0 {
				tagStr = fmt.Sprintf(" [%s]", strings.Join(template.Tags, ", "))
			}
			listLines = append(listLines, fmt.Sprintf("%s%s (%s)%s", prefix, template.Name, template.Duration, tagStr))
		}
	}

	preview := m.currentTemplateDetails()
	if len(m.templates) > 0 {
		preview = "Selected Template:\n" + preview
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(32).Render(strings.Join(listLines, "\n")),
		"    ",
		lipgloss.NewStyle().Width(40).Render(preview),
	)
	block := renderBanner(m.config) + "\n\n" +
		"╭─────────────────────────────────────╮\n" +
		"│  Session Templates                 │\n" +
		"╰─────────────────────────────────────╯\n\n" +
		body + "\n\n" +
		footer
	return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
}

func (m model) currentTemplate() (templates.SessionTemplate, bool) {
	if len(m.templates) == 0 || m.templateIndex < 0 || m.templateIndex >= len(m.templates) {
		return templates.SessionTemplate{}, false
	}
	return m.templates[m.templateIndex], true
}

func (m model) applySelectedTemplate() model {
	template, ok := m.currentTemplate()
	if !ok {
		return m
	}
	if strings.TrimSpace(template.Task) != "" {
		m.textInput.SetValue(template.Task)
		m.textInput.CursorEnd()
	}
	if strings.TrimSpace(template.Duration) != "" {
		m.durationInput.SetValue(template.Duration)
		m.durationInput.CursorEnd()
	}
	m.noteInput.SetValue(strings.TrimSpace(template.Note))
	m.tagInput.SetValue(strings.Join(template.Tags, ", "))
	m.inputError = ""
	m.appError = ""
	return m
}

func (m model) cycleTemplate(delta int) model {
	if len(m.templates) == 0 {
		return m
	}
	if m.templateIndex < 0 || m.templateIndex >= len(m.templates) {
		m.templateIndex = 0
	} else {
		m.templateIndex = (m.templateIndex + delta + len(m.templates)) % len(m.templates)
	}
	return m.applySelectedTemplate()
}

func (m *model) saveCurrentTemplate() error {
	task := strings.TrimSpace(m.textInput.Value())
	if task == "" {
		return fmt.Errorf("task name is required before saving a template")
	}
	template := templates.SessionTemplate{
		Name:     task,
		Task:     task,
		Duration: strings.TrimSpace(m.durationInput.Value()),
		Note:     strings.TrimSpace(m.noteInput.Value()),
		Tags:     parseTags(m.tagInput.Value()),
	}
	if strings.TrimSpace(template.Duration) == "" {
		return fmt.Errorf("duration is required before saving a template")
	}
	replaced := false
	for i, existing := range m.templates {
		if strings.EqualFold(strings.TrimSpace(existing.Name), task) {
			m.templates[i] = template
			m.templateIndex = i
			replaced = true
			break
		}
	}
	if !replaced {
		m.templates = append([]templates.SessionTemplate{template}, m.templates...)
		m.templateIndex = 0
	}
	if err := templates.SaveSessionTemplates(m.templateFile, m.templates); err != nil {
		return err
	}
	m.notificationStatus = fmt.Sprintf("Saved template: %s", task)
	return nil
}

func (m *model) renameSelectedTemplate() error {
	template, ok := m.currentTemplate()
	if !ok {
		return fmt.Errorf("no template selected")
	}
	newName := strings.TrimSpace(m.textInput.Value())
	if newName == "" {
		return fmt.Errorf("task name is required before renaming a template")
	}
	template.Name = newName
	template.Task = newName
	template.Duration = strings.TrimSpace(m.durationInput.Value())
	template.Note = strings.TrimSpace(m.noteInput.Value())
	template.Tags = parseTags(m.tagInput.Value())
	m.templates[m.templateIndex] = template
	if err := templates.SaveSessionTemplates(m.templateFile, m.templates); err != nil {
		return err
	}
	m.notificationStatus = fmt.Sprintf("Renamed template to: %s", newName)
	return nil
}

func (m *model) deleteSelectedTemplate() error {
	if len(m.templates) == 0 || m.templateIndex < 0 || m.templateIndex >= len(m.templates) {
		return fmt.Errorf("no template selected")
	}
	removed := m.templates[m.templateIndex]
	m.lastDeletedTemplate = &deletedTemplateState{
		template:  removed,
		index:     m.templateIndex,
		expiresAt: time.Now().Add(10 * time.Second),
	}
	m.templates = append(m.templates[:m.templateIndex], m.templates[m.templateIndex+1:]...)
	if len(m.templates) == 0 {
		m.templateIndex = 0
	} else if m.templateIndex >= len(m.templates) {
		m.templateIndex = len(m.templates) - 1
	}
	if err := templates.SaveSessionTemplates(m.templateFile, m.templates); err != nil {
		return err
	}
	if len(m.templates) > 0 {
		updated := m.applySelectedTemplate()
		*m = updated
	}
	m.notificationStatus = fmt.Sprintf("Deleted template: %s (Ctrl+Z to undo)", removed.Name)
	return nil
}

func (m *model) undoLastTemplateDelete() error {
	if m.lastDeletedTemplate == nil {
		return fmt.Errorf("no deleted template to undo")
	}
	if time.Now().After(m.lastDeletedTemplate.expiresAt) {
		m.lastDeletedTemplate = nil
		return fmt.Errorf("deleted template can no longer be restored")
	}
	restore := *m.lastDeletedTemplate
	if restore.index < 0 {
		restore.index = 0
	}
	if restore.index > len(m.templates) {
		restore.index = len(m.templates)
	}
	m.templates = append(m.templates, templates.SessionTemplate{})
	copy(m.templates[restore.index+1:], m.templates[restore.index:])
	m.templates[restore.index] = restore.template
	m.templateIndex = restore.index
	if err := templates.SaveSessionTemplates(m.templateFile, m.templates); err != nil {
		return err
	}
	m.lastDeletedTemplate = nil
	updated := m.applySelectedTemplate()
	*m = updated
	m.notificationStatus = fmt.Sprintf("Restored template: %s", restore.template.Name)
	return nil
}

func (m *model) duplicateSelectedTemplate() error {
	template, ok := m.currentTemplate()
	if !ok {
		return fmt.Errorf("no template selected")
	}
	dup := template
	dup.Name = template.Name + " Copy"
	dup.Task = template.Task + " Copy"
	m.templates = append([]templates.SessionTemplate{dup}, m.templates...)
	m.templateIndex = 0
	if err := templates.SaveSessionTemplates(m.templateFile, m.templates); err != nil {
		return err
	}
	m.notificationStatus = fmt.Sprintf("Duplicated template: %s", template.Name)
	return nil
}

func (m model) currentTemplateDetails() string {
	template, ok := m.currentTemplate()
	if !ok {
		return "No templates saved yet."
	}
	tags := "none"
	if len(template.Tags) > 0 {
		tags = strings.Join(template.Tags, ", ")
	}
	note := template.Note
	if strings.TrimSpace(note) == "" {
		note = "none"
	}
	return fmt.Sprintf("Task: %s\nDuration: %s\nNote: %s\nTags: %s", template.Task, template.Duration, note, tags)
}
