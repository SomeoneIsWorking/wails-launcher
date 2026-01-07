package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"nuxt-launcher/pkg/process"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// LogLevel represents the log level
type LogLevel = process.LogLevel

// ServiceStatus represents the service status
type ServiceStatus = process.ServiceStatus

// LogEntry represents a log entry
type LogEntry = process.LogEntry

// ServiceEnv represents environment variables
type ServiceEnv = process.ServiceEnv

// ServiceConfig represents service configuration
type ServiceConfig struct {
	Name string     `json:"name"`
	Path string     `json:"path"`
	Env  ServiceEnv `json:"env"`
}

// ServiceInfo represents service information
type ServiceInfo struct {
	Name   string        `json:"name"`
	Path   string        `json:"path"`
	Status ServiceStatus `json:"status"`
	URL    *string       `json:"url,omitempty"`
	Logs   []LogEntry    `json:"logs"`
	Env    ServiceEnv    `json:"env"`
}

// Service represents a service
type Service struct {
	ID             string
	Config         ServiceConfig
	Status         ServiceStatus
	Logs           []LogEntry
	URL            *string
	processManager *process.DotnetService
	mu             sync.RWMutex
	app            *App
}

// NewService creates a new service
func NewService(id string, config ServiceConfig, app *App) *Service {
	service := &Service{
		ID:     id,
		Config: config,
		Status: process.Stopped,
		Logs:   []LogEntry{},
		app:    app,
	}
	service.processManager = process.NewDotnetService(config.Path, config.Env)
	go service.listenEvents()
	return service
}

// listenEvents listens to process manager events
func (s *Service) listenEvents() {
	logChan, urlChan, statusChan := s.processManager.GetChannels()
	for {
		select {
		case log := <-logChan:
			s.mu.Lock()
			s.Logs = append(s.Logs, log)
			if len(s.Logs) > 100 { // MAX_LOGS
				s.Logs = s.Logs[1:]
			}
			s.mu.Unlock()
			// Emit to frontend
			s.app.emitToFrontend("newLog", s.ID, map[string]interface{}{"log": log})
			if log.Level == process.Err {
				s.app.emitToFrontend("statusUpdate", s.ID, map[string]interface{}{
					"status": s.Status,
					"url":    s.URL,
				})
			}
		case url := <-urlChan:
			s.mu.Lock()
			s.URL = &url
			s.mu.Unlock()
			s.app.emitToFrontend("statusUpdate", s.ID, map[string]interface{}{
				"status": s.Status,
				"url":    s.URL,
			})
		case status := <-statusChan:
			s.mu.Lock()
			if status == process.Stopped || status == process.Error {
				s.URL = nil
			}
			if status == process.Initializing {
				s.Status = process.Running
			} else {
				s.Status = status
			}
			s.mu.Unlock()
			s.app.emitToFrontend("statusUpdate", s.ID, map[string]interface{}{
				"status": s.Status,
				"url":    s.URL,
			})
		}
	}
}

// App struct
type App struct {
	ctx      context.Context
	services map[string]*Service
	mu       sync.RWMutex
}

// emitToFrontend emits an event to the frontend
func (a *App) emitToFrontend(event string, serviceId string, data interface{}) {
	runtime.EventsEmit(a.ctx, "serviceEvent", map[string]interface{}{
		"type":      event,
		"serviceId": serviceId,
		"data":      data,
	})
}

// NewApp creates a new App application struct
func NewApp() *App {
	app := &App{
		services: make(map[string]*Service),
	}
	app.loadServices()
	return app
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// loadServices loads services from services.json
func (a *App) loadServices() {
	data, err := os.ReadFile("services.json")
	if err != nil {
		return
	}
	var configs map[string]ServiceConfig
	json.Unmarshal(data, &configs)
	for id, config := range configs {
		service := NewService(id, config, a)
		a.services[id] = service
	}
}

// GetServices returns all services
func (a *App) GetServices() map[string]ServiceInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make(map[string]ServiceInfo)
	for id, service := range a.services {
		service.mu.RLock()
		result[id] = ServiceInfo{
			Name:   service.Config.Name,
			Path:   service.Config.Path,
			Status: service.Status,
			URL:    service.URL,
			Logs:   service.Logs,
			Env:    service.Config.Env,
		}
		service.mu.RUnlock()
	}
	return result
}

// AddService adds a new service
func (a *App) AddService(config ServiceConfig) *Service {
	a.mu.Lock()
	defer a.mu.Unlock()
	id := generateID()
	service := NewService(id, config, a)
	a.services[id] = service
	a.saveServices()
	return service
}

// GetService returns a service by ID
func (a *App) GetService(id string) *ServiceInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()
	service, exists := a.services[id]
	if !exists {
		return nil
	}
	service.mu.RLock()
	defer service.mu.RUnlock()
	return &ServiceInfo{
		Name:   service.Config.Name,
		Path:   service.Config.Path,
		Status: service.Status,
		URL:    service.URL,
		Logs:   service.Logs,
		Env:    service.Config.Env,
	}
}

// UpdateService updates a service
func (a *App) UpdateService(id string, config ServiceConfig) *Service {
	a.mu.Lock()
	defer a.mu.Unlock()
	service, exists := a.services[id]
	if !exists {
		return nil
	}
	service.mu.Lock()
	service.Config = config
	service.processManager.UpdateConfig(config.Path, config.Env)
	service.mu.Unlock()
	a.saveServices()
	return service
}

// StartService starts a service
func (a *App) StartService(id string) error {
	a.mu.RLock()
	service, exists := a.services[id]
	a.mu.RUnlock()
	if !exists {
		return fmt.Errorf("service not found")
	}
	return service.processManager.Start()
}

// StopService stops a service
func (a *App) StopService(id string) error {
	a.mu.RLock()
	service, exists := a.services[id]
	a.mu.RUnlock()
	if !exists {
		return fmt.Errorf("service not found")
	}
	return service.processManager.Stop()
}

// ClearLogs clears logs for a service
func (a *App) ClearLogs(id string) {
	a.mu.RLock()
	service, exists := a.services[id]
	a.mu.RUnlock()
	if !exists {
		return
	}
	service.mu.Lock()
	service.Logs = []LogEntry{}
	service.mu.Unlock()
}

// ReloadServices reloads services from config
func (a *App) ReloadServices() {
	a.mu.Lock()
	defer a.mu.Unlock()
	// Stop services not in config
	configs := a.loadConfigs()
	for id, service := range a.services {
		if _, exists := configs[id]; !exists {
			if service.Status == process.Running {
				service.processManager.Stop()
			}
			delete(a.services, id)
		}
	}
	// Update or create services
	for id, config := range configs {
		if service, exists := a.services[id]; exists {
			service.mu.Lock()
			service.Config = config
			service.processManager.UpdateConfig(config.Path, config.Env)
			service.mu.Unlock()
		} else {
			service := NewService(id, config, a)
			a.services[id] = service
		}
	}
}

// loadConfigs loads configs from file
func (a *App) loadConfigs() map[string]ServiceConfig {
	data, err := os.ReadFile("services.json")
	if err != nil {
		return make(map[string]ServiceConfig)
	}
	var configs map[string]ServiceConfig
	json.Unmarshal(data, &configs)
	return configs
}

// generateID generates a random ID
func generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}

// saveServices saves services to services.json
func (a *App) saveServices() {
	a.mu.RLock()
	defer a.mu.RUnlock()
	configs := make(map[string]ServiceConfig)
	for id, service := range a.services {
		configs[id] = service.Config
	}
	data, _ := json.MarshalIndent(configs, "", "  ")
	os.WriteFile("services.json", data, 0644)
}
