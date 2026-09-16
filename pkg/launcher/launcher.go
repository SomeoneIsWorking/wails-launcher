package launcher

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"wails-launcher/pkg/config"
	"wails-launcher/pkg/group"
	"wails-launcher/pkg/process"
	"wails-launcher/pkg/service"

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

// Emitter receives a service event. The GUI shell forwards it to the window; a
// headless host can ignore it or log it.
type Emitter func(event string, serviceId string, data interface{})

// FilePicker asks the user for a path. Only a shell with a window can answer, so a
// headless host leaves it unset and Browse says so rather than hanging.
type FilePicker func(title string, filterName string, pattern string) (string, error)

// App holds the services and drives them. It knows nothing about how it is hosted:
// the window and the file dialog arrive as callbacks, so the same App runs under the
// Wails GUI and under the headless daemon.
type App struct {
	ctx        context.Context
	services   map[string]*service.Service
	groups     *group.Manager
	config     *config.Config
	mu         sync.RWMutex
	httpServer *http.Server
	emit       Emitter
	pickFile   FilePicker
}

// EmitToFrontend emits a service event to whoever is hosting this App.
func (a *App) EmitToFrontend(event string, serviceId string, data interface{}) {
	if a.emit == nil {
		return
	}
	a.emit(event, serviceId, data)
}


// NewApp creates a launcher with no window behind it: service events go nowhere and
// Browse reports that there is no dialog. This is what the headless daemon uses.
func NewApp() *App {
	return NewAppWith(nil, nil)
}

// NewAppWith creates a launcher that reports its service events through emit and asks
// for paths through pick. Either may be nil. The callbacks are read at call time, so a
// shell may pass methods that only become usable once its window exists.
func NewAppWith(emit Emitter, pick FilePicker) *App {
	cfg, err := config.Load()
	if err != nil {
		// Handle error, maybe create empty config
		cfg = &config.Config{Groups: make(map[string]config.GroupConfig)}
	}

	app := &App{
		services: make(map[string]*service.Service),
		groups:   group.NewManager(cfg.Groups),
		config:   cfg,
		emit:     emit,
		pickFile: pick,
	}
	app.loadServices()
	return app
}

// Startup starts the control API. The context is kept so the host can cancel it.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.startHTTPServer(ctx)
}

// Shutdown stops the control API.
func (a *App) Shutdown(ctx context.Context) {
	if a.httpServer != nil {
		a.httpServer.Shutdown(ctx) //nolint:errcheck
	}
}

// loadServices loads services from configuration
func (a *App) loadServices() {
	groupServices := a.groups.GetGroupServices()
	for serviceId, enriched := range groupServices {
		srv := service.NewService(serviceId, enriched.Config, enriched.InheritedEnv, a)
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

// StartServiceWithoutBuild starts a service without building
func (a *App) StartServiceWithoutBuild(id string) error {
	a.mu.RLock()
	srv, exists := a.services[id]
	a.mu.RUnlock()
	if !exists {
		return fmt.Errorf("service not found")
	}
	return srv.StartWithoutBuild()
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

// RestartService stops then starts a service
func (a *App) RestartService(id string) error {
	if err := a.StopService(id); err != nil {
		return err
	}
	return a.StartService(id)
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
	for id, enriched := range groupServices {
		if srv, exists := a.services[id]; exists {
			srv.UpdateConfig(enriched.Config, enriched.InheritedEnv)
		} else {
			srv := service.NewService(id, enriched.Config, enriched.InheritedEnv, a)
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
				if enriched, exists := groupServices[serviceId]; exists {
					srv.UpdateConfig(enriched.Config, enriched.InheritedEnv)
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
	if enriched, exists := groupServices[serviceId]; exists {
		srv := service.NewService(serviceId, enriched.Config, enriched.InheritedEnv, a)
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
	if enriched, exists := groupServices[serviceId]; exists {
		if srv, exists := a.services[serviceId]; exists {
			srv.UpdateConfig(enriched.Config, enriched.InheritedEnv)
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
	for serviceId, enriched := range groupServices {
		if _, exists := a.services[serviceId]; !exists {
			srv := service.NewService(serviceId, enriched.Config, enriched.InheritedEnv, a)
			a.services[serviceId] = srv
		}
	}

	return nil
}

// ImportProject imports a single project into a group
func (a *App) ImportProject(groupId string, path string, projectType string) error {
	serviceId, err := a.groups.ImportProject(groupId, path, projectType)
	if err != nil {
		return err
	}
	a.saveConfig()

	// Create the service
	groupServices := a.groups.GetGroupServices()
	if enriched, exists := groupServices[serviceId]; exists {
		if _, exists := a.services[serviceId]; !exists {
			srv := service.NewService(serviceId, enriched.Config, enriched.InheritedEnv, a)
			a.services[serviceId] = srv
		}
	}

	return nil
}

// GetLaunchProfiles reads Properties/launchSettings.json for a dotnet project
// and returns the available profile names.
func (a *App) GetLaunchProfiles(projectPath string) ([]string, error) {
	fmt.Printf("GetLaunchProfiles called for: %s\n", projectPath)
	settingsPath := filepath.Join(projectPath, "Properties", "launchSettings.json")
	fmt.Printf("Looking for settings at: %s\n", settingsPath)
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var settings struct {
		Profiles map[string]json.RawMessage `json:"profiles"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}

	names := make([]string, 0, len(settings.Profiles))
	for name := range settings.Profiles {
		names = append(names, name)
	}
	return names, nil
}

// Browse opens a file dialog and returns the selected path.
func (a *App) Browse(title string, filterName string, pattern string) (string, error) {
	if a.pickFile == nil {
		return "", fmt.Errorf("no file dialog available: this launcher is running without a window")
	}
	return a.pickFile(title, filterName, pattern)
}

// DeleteService deletes a service
func (a *App) DeleteService(serviceId string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Stop the service if running
	if srv, exists := a.services[serviceId]; exists {
		srv.Stop()
		delete(a.services, serviceId)
	}

	// Find the group and remove from it
	if groupId, found := a.groups.FindGroupByService(serviceId); found {
		a.groups.DeleteServiceFromGroup(groupId, serviceId)
		a.saveConfig()
		return nil
	}

	return fmt.Errorf("service not found")
}

// StartGroup starts all services in a group
func (a *App) StartGroup(groupId string) {
	groups := a.groups.GetGroups()
	if group, exists := groups[groupId]; exists {
		for serviceId := range group.Services {
			go func(id string) {
				a.StartService(id)
			}(serviceId)
		}
	}
}

// StartGroupWithoutBuild starts all services in a group without building
func (a *App) StartGroupWithoutBuild(groupId string) {
	groups := a.groups.GetGroups()
	if group, exists := groups[groupId]; exists {
		for serviceId := range group.Services {
			go func(id string) {
				a.StartServiceWithoutBuild(id)
			}(serviceId)
		}
	}
}

// saveConfig saves the configuration
func (a *App) saveConfig() {
	a.config.Groups = a.groups.GetGroups()
	a.config.Save()
}
