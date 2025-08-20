package kensho

import (
	"embed"
	"fmt"

	"gopkg.in/yaml.v3"
)

//go:embed document_types.yml
var defaultConfigFile embed.FS

// loadDefaultConfig loads the configuration from the embedded document_types.yml file.
func loadDefaultConfig() (*Config, error) {
	data, err := defaultConfigFile.ReadFile("document_types.yml")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded config file: %w", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal embedded config: %w", err)
	}

	return &config, nil
}
