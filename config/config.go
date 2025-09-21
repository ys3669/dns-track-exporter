package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

// Config represents the application configuration
type Config struct {
	Server     ServerConfig  `yaml:"server"`
	Monitoring MonitorConfig `yaml:"monitoring"`
	DNSServers []DNSServer   `yaml:"dns_servers"`
	Targets    []Target      `yaml:"targets"`
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Port int `yaml:"port"`
}

// MonitorConfig contains monitoring configuration
type MonitorConfig struct {
	Interval time.Duration `yaml:"interval"`
	Timeout  time.Duration `yaml:"timeout"`
}

// DNSServer represents a DNS server configuration
type DNSServer struct {
	Name    string `yaml:"name"`
	Address string `yaml:"address"`
}

// Target represents a DNS resolution target
type Target struct {
	FQDN        string   `yaml:"fqdn"`
	RecordTypes []string `yaml:"record_types"`
}

// LoadConfig loads configuration from YAML file
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set default values if not specified
	if config.Server.Port == 0 {
		config.Server.Port = 9653
	}
	if config.Monitoring.Interval == 0 {
		config.Monitoring.Interval = 30 * time.Second
	}
	if config.Monitoring.Timeout == 0 {
		config.Monitoring.Timeout = 10 * time.Second
	}

	return &config, nil
}

// GetListenAddress returns the server listen address
func (c *Config) GetListenAddress() string {
	return fmt.Sprintf(":%d", c.Server.Port)
}
