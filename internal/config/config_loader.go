package config

import (
	"cf-observer/internal/audit"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	yaml "gopkg.in/yaml.v3"
)

func LoadConfigFile(override string) (map[string]Host, error) {
	var configPath string

	if override != "" {
		configPath = override
	} else {
		baseDir, err := os.UserConfigDir()
		if err != nil {
			return map[string]Host{}, err
		}

		configPath = filepath.Join(baseDir, "codeforge-observer", "config.yaml")
	}

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

	for key, host := range cfg.Hosts {
		if host.APIContractPath == "" {
			cfg.Hosts[key] = host
			continue
		}

		contractPath := host.APIContractPath
		if !filepath.IsAbs(contractPath) {
			contractPath = filepath.Join(filepath.Dir(configPath), contractPath)
		}

		contract, err := audit.LoadOpenAPIDocument(contractPath)
		if err != nil {
			return map[string]Host{}, fmt.Errorf("load api contract for host %q: %w", key, err)
		}

		host.APIContract = contract
		cfg.Hosts[key] = host
	}

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
			UpstreamRaw: host.UpstreamRaw,
			Upstream:    u,
			APIContract: host.APIContract,
			// ResourceContract: host.ResourceContract,
		}
	}

	return c.Hosts, nil
}
