package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	ErrEmptyPath   = errors.New("config path is empty")
	ErrInvalidEnv  = errors.New("invalid environment")
	ErrEnvRequired = errors.New("env is required")
)

type GoogleDrive struct {
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
	TokenPath    string `yaml:"token_path"`
}

type YandexDisk struct {
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
	TokenPath    string `yaml:"token_path"`
}

type Config struct {
	Env         string      `yaml:"env"`
	GoogleDrive GoogleDrive `yaml:"google_drive"`
	YandexDisk  YandexDisk  `yaml:"yandex_disk"`
	LocalDir    string      `yaml:"local_dir"`
}

func Load(path string) (*Config, error) {
	if strings.TrimSpace(path) == "" {
		return nil, ErrEmptyPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var config Config

	if err = yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err = config.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &config, nil
}

func (c *Config) Validate() error {
	switch c.Env {
	case "local", "dev", "prod":
	case "":
		return ErrEnvRequired
	default:
		return fmt.Errorf("%w: %q, allowed: local, dev, prod", ErrInvalidEnv, c.Env)
	}

	if strings.TrimSpace(c.LocalDir) == "" {
		return errors.New("local_dir is required")
	}

	return nil
}
