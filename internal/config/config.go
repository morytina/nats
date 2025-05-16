package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type LoggerConfig struct {
	Region string `yaml:"region"`
	Env    string `yaml:"env"`
	Level  string `yaml:"level"` // "debug", "info", etc.
}

func LoadConfig(path string) (*LoggerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg LoggerConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
