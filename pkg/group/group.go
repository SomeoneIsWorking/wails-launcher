package group

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"wails-launcher/pkg/config"
	"wails-launcher/pkg/service"
)

// Manager handles group operations
type Manager struct {
	groups map[string]config.GroupConfig
}

// NewManager creates a new group manager
func NewManager(groups map[string]config.GroupConfig) *Manager {
	if groups == nil {
		groups = make(map[string]config.GroupConfig)
	}
	return &Manager{groups: groups}
}

// GetGroups returns all groups
func (m *Manager) GetGroups() map[string]config.GroupConfig {
	// Return a copy
	result := make(map[string]config.GroupConfig)
	for id, group := range m.groups {
		groupCopy := config.GroupConfig{
			Name:     group.Name,
			Env:      make(config.ServiceEnv),
			Services: make(map[string]config.ServiceConfig),
		}
		for k, v := range group.Env {
			groupCopy.Env[k] = v
		}
		for sid, sconfig := range group.Services {
			groupCopy.Services[sid] = sconfig
		}
		result[id] = groupCopy
	}
	return result
}

// AddGroup adds a new group
func (m *Manager) AddGroup(name string, env config.ServiceEnv) string {
	groupId := service.GenerateID()
	group := config.GroupConfig{
		Name:     name,
		Env:      env,
		Services: make(map[string]config.ServiceConfig),
	}
	m.groups[groupId] = group
	return groupId
}

// UpdateGroup updates a group
func (m *Manager) UpdateGroup(id string, name string, env config.ServiceEnv) {
	if group, exists := m.groups[id]; exists {
		group.Name = name
		group.Env = env
		m.groups[id] = group
	}
}

// AddServiceToGroup adds a service to a group
func (m *Manager) AddServiceToGroup(groupId string, serviceConfig config.ServiceConfig) string {
	if group, exists := m.groups[groupId]; exists {
		serviceId := service.GenerateID()
		group.Services[serviceId] = serviceConfig
		m.groups[groupId] = group
		return serviceId
	}
	return ""
}

// UpdateServiceInGroup updates a service in a group
func (m *Manager) UpdateServiceInGroup(groupId string, serviceId string, serviceConfig config.ServiceConfig) {
	if group, exists := m.groups[groupId]; exists {
		if _, sExists := group.Services[serviceId]; sExists {
			group.Services[serviceId] = serviceConfig
			m.groups[groupId] = group
		}
	}
}

// DeleteServiceFromGroup deletes a service from a group
func (m *Manager) DeleteServiceFromGroup(groupId string, serviceId string) bool {
	if group, exists := m.groups[groupId]; exists {
		if _, sExists := group.Services[serviceId]; sExists {
			delete(group.Services, serviceId)
			m.groups[groupId] = group
			return true
		}
	}
	return false
}

// ImportSLN imports projects from a .sln file and creates a group
func (m *Manager) ImportSLN(slnPath string) error {
	file, err := os.Open(slnPath)
	if err != nil {
		return err
	}
	defer file.Close()

	groupName := strings.TrimSuffix(filepath.Base(slnPath), filepath.Ext(slnPath))

	group := config.GroupConfig{
		Name:     groupName,
		Env:      make(config.ServiceEnv),
		Services: make(map[string]config.ServiceConfig),
	}

	scanner := bufio.NewScanner(file)
	projectRegex := regexp.MustCompile(`Project\("\{[^}]+\}"\) = "([^"]+)", "([^"]+\.csproj)", "\{[^}]+\}"`)

	for scanner.Scan() {
		line := scanner.Text()
		matches := projectRegex.FindStringSubmatch(line)
		if len(matches) == 3 {
			projectName := matches[1]
			projectPath := matches[2]

			// Convert Windows backslashes to forward slashes
			projectPath = strings.ReplaceAll(projectPath, "\\", "/")

			// Convert relative path to absolute
			if !filepath.IsAbs(projectPath) {
				projectPath = filepath.Join(filepath.Dir(slnPath), projectPath)
			}

			serviceId := service.GenerateID()
			serviceConfig := config.ServiceConfig{
				Name: projectName,
				Path: filepath.Dir(projectPath), // Directory containing the .csproj
				Env:  make(config.ServiceEnv),
				Type: "dotnet",
			}

			// Only add projects that have a Program.cs file
			programPath := filepath.Join(serviceConfig.Path, "Program.cs")
			if _, err := os.Stat(programPath); err == nil {
				group.Services[serviceId] = serviceConfig
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	groupId := service.GenerateID()
	m.groups[groupId] = group
	return nil
}

// ImportProject imports a single project into a group
func (m *Manager) ImportProject(groupId string, path string, projectType string) (string, error) {
	dir := filepath.Dir(path)
	name := filepath.Base(dir) // Default to folder name

	if projectType == "npm" && strings.HasSuffix(path, "package.json") {
		// Try to read package.json
		data, err := os.ReadFile(path)
		if err == nil {
			var pkg struct {
				Name string `json:"name"`
			}
			if err := json.Unmarshal(data, &pkg); err == nil && pkg.Name != "" {
				name = pkg.Name
			}
		}
	} else if projectType == "dotnet" {
		// For .csproj, filename is usually the project name
		name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}

	serviceConfig := config.ServiceConfig{
		Name: name,
		Path: dir,
		Env:  make(config.ServiceEnv),
		Type: projectType,
	}

	return m.AddServiceToGroup(groupId, serviceConfig), nil
}

// EnrichedServiceConfig includes the service config and its inherited environment
type EnrichedServiceConfig struct {
	Config       config.ServiceConfig
	InheritedEnv config.ServiceEnv
}

// GetGroupServices returns all services in all groups with their inherited environments
func (m *Manager) GetGroupServices() map[string]EnrichedServiceConfig {
	result := make(map[string]EnrichedServiceConfig)
	for _, group := range m.groups {
		for serviceId, serviceConfig := range group.Services {
			result[serviceId] = EnrichedServiceConfig{
				Config:       serviceConfig,
				InheritedEnv: group.Env,
			}
		}
	}
	return result
}

// FindGroupByService finds the group containing a service
func (m *Manager) FindGroupByService(serviceId string) (string, bool) {
	for groupId, group := range m.groups {
		if _, exists := group.Services[serviceId]; exists {
			return groupId, true
		}
	}
	return "", false
}
