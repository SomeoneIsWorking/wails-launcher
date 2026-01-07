package group

import (
	"bufio"
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

// GetGroupServices returns all services in all groups with merged environments
func (m *Manager) GetGroupServices() map[string]config.ServiceConfig {
	result := make(map[string]config.ServiceConfig)
	for _, group := range m.groups {
		for serviceId, serviceConfig := range group.Services {
			// Merge group env with service env
			mergedEnv := make(config.ServiceEnv)
			for k, v := range group.Env {
				mergedEnv[k] = v
			}
			for k, v := range serviceConfig.Env {
				mergedEnv[k] = v
			}
			result[serviceId] = config.ServiceConfig{
				Name: serviceConfig.Name,
				Path: serviceConfig.Path,
				Env:  mergedEnv,
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
