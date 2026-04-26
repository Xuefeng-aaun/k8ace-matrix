package matrix

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func Load(path string) (*Matrix, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read matrix: %w", err)
	}

	var m Matrix
	if err := yaml.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("unmarshal matrix yaml: %w", err)
	}

	return &m, nil
}
