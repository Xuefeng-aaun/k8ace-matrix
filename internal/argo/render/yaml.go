package render

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

func MarshalYAML(v any) ([]byte, error) {
	b, err := yaml.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal yaml: %w", err)
	}
	return b, nil
}
