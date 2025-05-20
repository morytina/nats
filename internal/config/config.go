package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

var Root *Config

const configPath = "configs/config.yaml"

// 전체 설정 구조체 정의
type Config struct {
	Region string       `yaml:"region"`
	Env    string       `yaml:"env"`
	Log    LoggerConfig `yaml:"log"`
}

// 로깅 관련 설정 구조체
type LoggerConfig struct {
	Level string `yaml:"level"` // debug, info, warn, error, etc.
}

func Init() error {
	loaded, err := loadConfig(configPath)
	if err != nil {
		panic("config load error: " + err.Error())
	}
	Root = loaded
	return nil
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
