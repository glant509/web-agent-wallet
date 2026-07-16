package tool

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type definitionFile struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

func LoadDefinition(path string) (Definition, error) {
	content, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return Definition{}, fmt.Errorf("read tool definition %q: %w", path, err)
	}

	var file definitionFile
	if err := json.Unmarshal(content, &file); err != nil {
		return Definition{}, fmt.Errorf("decode tool definition %q: %w", path, err)
	}
	if file.Name == "" {
		return Definition{}, fmt.Errorf("tool definition %q is missing name", path)
	}

	return Definition{
		Name:        file.Name,
		Description: file.Description,
		InputSchema: file.InputSchema,
	}, nil
}
