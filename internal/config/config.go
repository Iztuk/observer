package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
)

type Config struct {
	RunTime RunTimeConfig   `yaml:"runtime"`
	Hosts   map[string]Host `yaml:"hosts"`
}

type Host struct {
	UpstreamRaw     string   `yaml:"upstream"`
	Upstream        *url.URL `yaml:"-"`
	APIContractPath string   `yaml:"api_contract"`
}

type RunTimeConfig struct {
	Listen      string      `yaml:"listen"`
	DefaultHost *Host       `yaml:"default_host"`
	AuditConfig AuditConfig `yaml:"audit_config"`
}

type AuditConfig struct {
	Enabled   bool `yaml:"enabled"`
	QueueSize int  `yaml:"queue_size"`
	Workers   int  `yaml:"worker"`
}

type ProcessConfig struct {
	PidFile  string
	LogFile  string
	SockFile string
}

var AppRunTimeConfig RunTimeConfig

var AppProcessConfig ProcessConfig = ProcessConfig{
	PidFile:  "/tmp/observer.pid",
	LogFile:  "/tmp/observer.log",
	SockFile: "/tmp/observer.sock",
}

const defaultConfigYAML = `# Observer
runtime:
  listen: ":8080"

  default_host: null

  audit_config:
    enabled: true
    queue_size: 1000
    worker: 4

hosts:
`

func ConfigDir() (string, error) {
	baseDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("get user config dir: %w", err)
	}

	return filepath.Join(baseDir, "observer"), nil
}

func ConfigFilePath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "config.yaml"), nil
}

func InitConfigDir(force bool) error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	configPath := filepath.Join(dir, "config.yaml")

	if !force {
		_, err = os.Stat(configPath)
		if err == nil {
			return fmt.Errorf("config file already exists: %s", configPath)
		}
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("check config file: %w", err)
		}
	}

	if err := os.WriteFile(configPath, []byte(defaultConfigYAML), 0o600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}
