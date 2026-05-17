package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	yaml "gopkg.in/yaml.v3"
)

func LoadConfigFile() (map[string]Host, error) {
	var configPath string

	baseDir, err := os.UserConfigDir()
	if err != nil {
		return map[string]Host{}, err
	}

	configPath = filepath.Join(baseDir, "observer", "config.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return map[string]Host{}, err
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return map[string]Host{}, err
	}

	err = cfg.Validate()
	if err != nil {
		return map[string]Host{}, err
	}

	AppRunTimeConfig = cfg.RunTime

	return cfg.ValidateHostUrls()
}

func (c *Config) Validate() error {
	if c.RunTime.Listen == "" {
		return fmt.Errorf("listen is required")
	}

	for hostName, host := range c.Hosts {
		if hostName == "" {
			return fmt.Errorf("host name cannot be empty")
		}
		if host.UpstreamRaw == "" {
			return fmt.Errorf("host %q is missing upstream", hostName)
		}
	}

	if c.RunTime.AuditConfig.QueueSize < 0 {
		return fmt.Errorf("audit queue_size cannot be negative")
	}
	if c.RunTime.AuditConfig.Workers < 0 {
		return fmt.Errorf("audit workers cannot be negative")
	}

	return nil
}

func (c *Config) ValidateHostUrls() (map[string]Host, error) {
	for key, host := range c.Hosts {
		u, err := url.Parse(host.UpstreamRaw)
		if err != nil {
			return map[string]Host{}, fmt.Errorf("parse upstream for host %q: %w", key, err)
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return map[string]Host{}, fmt.Errorf("host %q upstream must use http or https", key)
		}
		if u.Host == "" {
			return map[string]Host{}, fmt.Errorf("host %q upstream must include a host", key)
		}

		c.Hosts[key] = Host{
			UpstreamRaw:     host.UpstreamRaw,
			Upstream:        u,
			APIContractPath: host.APIContractPath,
		}
	}

	return c.Hosts, nil
}
