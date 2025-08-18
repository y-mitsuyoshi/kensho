package configs

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Document struct {
	Prompt        string            `yaml:"prompt"`
	JSONStructure map[string]string `yaml:"json_structure"`
}

type Config struct {
	Documents map[string]Document `yaml:"documents"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}
