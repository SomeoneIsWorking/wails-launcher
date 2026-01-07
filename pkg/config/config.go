package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"wails-launcher/pkg/process"
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

// getConfigPath returns the path to the services.json file
func getConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "wails-launcher", "services.json"), nil
}

// Load loads configuration from services.json
func Load() (*Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Try to migrate from old location
			oldData, oldErr := os.ReadFile("services.json")
			if oldErr == nil {
				// Create directory and copy file
				dir := filepath.Dir(configPath)
				if mkdirErr := os.MkdirAll(dir, 0755); mkdirErr != nil {
					return nil, mkdirErr
				}
				if writeErr := os.WriteFile(configPath, oldData, 0644); writeErr != nil {
					return nil, writeErr
				}
				data = oldData
			} else {
				// Create directory if needed
				dir := filepath.Dir(configPath)
				os.MkdirAll(dir, 0755)
				return &Config{Groups: make(map[string]GroupConfig)}, nil
			}
		} else {
			return nil, err
		}
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
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
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
