package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

const configPath = "configs/config.yaml"

// 전체 설정 구조체 정의
type Config struct {
	Region  string        `yaml:"region"`
	Env     string        `yaml:"env"`
	Log     LoggerConfig  `yaml:"log"`
	Nats    NatsConfig    `yaml:"nats"`
	Valkey  ValkeyConfig  `yaml:"valkey"`
	Publish PublishConfig `yaml:"publish"`
}

type LoggerConfig struct {
	Level string `yaml:"level"` // debug, info, warn, error, etc.
}

type NatsConfig struct {
	ConnPoolCnt int `yaml:"connPoolCount"`
}

type ValkeyConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type PublishConfig struct {
	Worker int `yaml:"worker"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		data, err = os.ReadFile(configPath)
		if err != nil {
			return nil, err
		}
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
