package main

import (
	"context"
	"fmt"
	"sync"

	"wails-launcher/pkg/config"
	"wails-launcher/pkg/group"
	"wails-launcher/pkg/process"
	"wails-launcher/pkg/service"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// LogLevel represents the log level
type LogLevel = process.LogLevel

// ServiceStatus represents the service status
type ServiceStatus = process.ServiceStatus

// LogEntry represents a log entry
type LogEntry = process.LogEntry

// ServiceEnv represents environment variables
type ServiceEnv = config.ServiceEnv

// ServiceConfig represents service configuration
type ServiceConfig = config.ServiceConfig

// GroupConfig represents group configuration
type GroupConfig = config.GroupConfig

// ServiceInfo represents service information
type ServiceInfo = service.ServiceInfo

// App struct
type App struct {
	ctx      context.Context
	services map[string]*service.Service
	groups   *group.Manager
	config   *config.Config
	mu       sync.RWMutex
}

// EmitToFrontend emits an event to the frontend
func (a *App) EmitToFrontend(event string, serviceId string, data interface{}) {
	runtime.EventsEmit(a.ctx, "serviceEvent", map[string]interface{}{
		"type":      event,
		"serviceId": serviceId,
		"data":      data,
	})
}

// NewApp creates a new App application struct
func NewApp() *App {
	cfg, err := config.Load()
	if err != nil {
		// Handle error, maybe create empty config
		cfg = &config.Config{Groups: make(map[string]config.GroupConfig)}
	}

	app := &App{
		services: make(map[string]*service.Service),
		groups:   group.NewManager(cfg.Groups),
		config:   cfg,
	}
	app.loadServices()
	return app
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// loadServices loads services from configuration
func (a *App) loadServices() {
	groupServices := a.groups.GetGroupServices()
	for serviceId, serviceConfig := range groupServices {
		srv := service.NewService(serviceId, serviceConfig, a)
		a.services[serviceId] = srv
	}
}

// GetServices returns all services
func (a *App) GetServices() map[string]ServiceInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make(map[string]ServiceInfo)
	for id, srv := range a.services {
		result[id] = srv.GetInfo()
	}
	return result
}

// AddService adds a new service to the default group
func (a *App) AddService(config ServiceConfig) *service.Service {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Find or create default group
	defaultGroupId := ""
	groups := a.groups.GetGroups()
	for id, grp := range groups {
		if grp.Name == "Default" {
			defaultGroupId = id
			break
		}
	}
	if defaultGroupId == "" {
		defaultGroupId = a.AddGroup("Default", make(ServiceEnv))
	}

	serviceId := a.AddServiceToGroup(defaultGroupId, config)
	return a.services[serviceId]
}

// GetService returns a service by ID
func (a *App) GetService(id string) *ServiceInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()
	srv, exists := a.services[id]
	if !exists {
		return nil
	}
	info := srv.GetInfo()
	return &info
}

// UpdateService updates a service (assumes it's in default group for backward compatibility)
func (a *App) UpdateService(id string, config ServiceConfig) *service.Service {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Find the group containing this service
	if groupId, found := a.groups.FindGroupByService(id); found {
		a.UpdateServiceInGroup(groupId, id, config)
		return a.services[id]
	}
	return nil
}

// StartService starts a service
func (a *App) StartService(id string) error {
	a.mu.RLock()
	srv, exists := a.services[id]
	a.mu.RUnlock()
	if !exists {
		return fmt.Errorf("service not found")
	}
	return srv.Start()
}

// StopService stops a service
func (a *App) StopService(id string) error {
	a.mu.RLock()
	srv, exists := a.services[id]
	a.mu.RUnlock()
	if !exists {
		return fmt.Errorf("service not found")
	}
	return srv.Stop()
}

// ClearLogs clears logs for a service
func (a *App) ClearLogs(id string) {
	a.mu.RLock()
	srv, exists := a.services[id]
	a.mu.RUnlock()
	if !exists {
		return
	}
	srv.ClearLogs()
}

// ReloadServices reloads services from config
func (a *App) ReloadServices() {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Reload config
	cfg, err := config.Load()
	if err != nil {
		return
	}
	a.config = cfg
	a.groups = group.NewManager(cfg.Groups)

	// Stop services not in config
	groupServices := a.groups.GetGroupServices()
	for id, srv := range a.services {
		if _, exists := groupServices[id]; !exists {
			srv.Stop()
			delete(a.services, id)
		}
	}

	// Update or create services
	for id, serviceConfig := range groupServices {
		if srv, exists := a.services[id]; exists {
			srv.UpdateConfig(serviceConfig)
		} else {
			srv := service.NewService(id, serviceConfig, a)
			a.services[id] = srv
		}
	}
}

// GetGroups returns all groups
func (a *App) GetGroups() map[string]config.GroupConfig {
	return a.groups.GetGroups()
}

// AddGroup adds a new group
func (a *App) AddGroup(name string, env config.ServiceEnv) string {
	groupId := a.groups.AddGroup(name, env)
	a.saveConfig()
	return groupId
}

// UpdateGroup updates a group
func (a *App) UpdateGroup(id string, name string, env config.ServiceEnv) {
	a.groups.UpdateGroup(id, name, env)
	a.saveConfig()

	// Update all services in the group with new merged env
	groups := a.groups.GetGroups()
	if grp, exists := groups[id]; exists {
		for serviceId := range grp.Services {
			if srv, exists := a.services[serviceId]; exists {
				groupServices := a.groups.GetGroupServices()
				if serviceConfig, exists := groupServices[serviceId]; exists {
					srv.UpdateConfig(serviceConfig)
				}
			}
		}
	}
}

// AddServiceToGroup adds a service to a group
func (a *App) AddServiceToGroup(groupId string, config config.ServiceConfig) string {
	serviceId := a.groups.AddServiceToGroup(groupId, config)
	a.saveConfig()

	// Create the service
	groupServices := a.groups.GetGroupServices()
	if serviceConfig, exists := groupServices[serviceId]; exists {
		srv := service.NewService(serviceId, serviceConfig, a)
		a.services[serviceId] = srv
	}
	return serviceId
}

// UpdateServiceInGroup updates a service in a group
func (a *App) UpdateServiceInGroup(groupId string, serviceId string, config config.ServiceConfig) {
	a.groups.UpdateServiceInGroup(groupId, serviceId, config)
	a.saveConfig()

	// Update the service
	groupServices := a.groups.GetGroupServices()
	if serviceConfig, exists := groupServices[serviceId]; exists {
		if srv, exists := a.services[serviceId]; exists {
			srv.UpdateConfig(serviceConfig)
		}
	}
}

// ImportSLN imports projects from a .sln file and creates a group
func (a *App) ImportSLN(slnPath string) error {
	err := a.groups.ImportSLN(slnPath)
	if err != nil {
		return err
	}
	a.saveConfig()

	// Create services for the new group
	groupServices := a.groups.GetGroupServices()
	for serviceId, serviceConfig := range groupServices {
		if _, exists := a.services[serviceId]; !exists {
			srv := service.NewService(serviceId, serviceConfig, a)
			a.services[serviceId] = srv
		}
	}

	return nil
}

// saveConfig saves the configuration
func (a *App) saveConfig() {
	a.config.Groups = a.groups.GetGroups()
	a.config.Save()
}
