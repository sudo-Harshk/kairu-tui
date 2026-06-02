package templates

import (
	"encoding/json"
	"errors"
	"os"
)

type SessionTemplate struct {
	Name     string   `json:"name"`
	Task     string   `json:"task"`
	Duration string   `json:"duration"`
	Note     string   `json:"note,omitempty"`
	Tags     []string `json:"tags,omitempty"`
}

func LoadSessionTemplates(path string) ([]SessionTemplate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []SessionTemplate{}, nil
		}
		return nil, err
	}
	var templates []SessionTemplate
	if err := json.Unmarshal(data, &templates); err != nil {
		return nil, err
	}
	return templates, nil
}

func SaveSessionTemplates(path string, templates []SessionTemplate) error {
	data, err := json.MarshalIndent(templates, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
