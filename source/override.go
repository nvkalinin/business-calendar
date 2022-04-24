package source

import (
	"fmt"
	"github.com/nvkalinin/business-calendar/store"
	"gopkg.in/yaml.v3"
	"os"
)

// Override - источник, который берет данные из YAML-файла.
type Override struct {
	Path string
}

type overrides map[int]store.Months // Ключ - год.

func (o *Override) GetYear(y int) (store.Months, error) {
	// Админ может менять файл, поэтому читаем его при каждом вызове.
	f, err := os.ReadFile(o.Path)
	if err != nil {
		return nil, fmt.Errorf("cannot read overrides yaml: %w", err)
	}

	ov := overrides{}
	if err := yaml.Unmarshal(f, &ov); err != nil {
		return nil, fmt.Errorf("cannot parse overrides yaml: %w", err)
	}

	months, _ := ov[y]
	return months, nil
}
