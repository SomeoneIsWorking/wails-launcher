package config

import (
	"encoding/json"
	"os"

	"nuxt-launcher/pkg/process"
)

// ServiceEnv represents environment variables
type ServiceEnv = process.ServiceEnv

// ServiceConfig represents service configuration
type ServiceConfig struct {
	Name string     `json:"name"`
	Path string     `json:"path"`
	Env  ServiceEnv `json:"env"`
}

// GroupConfig represents group configuration
type GroupConfig struct {
	Name     string                   `json:"name"`
	Env      ServiceEnv               `json:"env"`
	Services map[string]ServiceConfig `json:"services"`
}

// Config represents the overall configuration
type Config struct {
	Groups map[string]GroupConfig `json:"groups"`
}

// Load loads configuration from services.json
func Load() (*Config, error) {
	data, err := os.ReadFile("services.json")
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Groups: make(map[string]GroupConfig)}, nil
		}
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	if config.Groups == nil {
		config.Groups = make(map[string]GroupConfig)
	}

	return &config, nil
}

// Save saves configuration to services.json
func (c *Config) Save() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile("services.json", data, 0644)
}

// MigrateFromOldFormat migrates old flat service format to new grouped format
func MigrateFromOldFormat(oldConfigs map[string]ServiceConfig) *Config {
	config := &Config{Groups: make(map[string]GroupConfig)}
	defaultGroup := GroupConfig{
		Name:     "Default",
		Env:      make(ServiceEnv),
		Services: oldConfigs,
	}
	config.Groups["default"] = defaultGroup
	return config
}
