package service

import (
	"crypto/rand"
	"fmt"
	"sync"

	"wails-launcher/pkg/config"
	"wails-launcher/pkg/process"
)

// ServiceInfo represents service information
type ServiceInfo struct {
	Name         string                `json:"name"`
	Path         string                `json:"path"`
	Status       process.ServiceStatus `json:"status"`
	URL          *string               `json:"url,omitempty"`
	Logs         []process.LogEntry    `json:"logs"`
	Env          config.ServiceEnv     `json:"env"`
	InheritedEnv config.ServiceEnv     `json:"inheritedEnv"`
	Type         string                `json:"type"`
}

// Service represents a service
type Service struct {
	ID             string
	Config         config.ServiceConfig
	InheritedEnv   config.ServiceEnv
	Status         process.ServiceStatus
	Logs           []process.LogEntry
	URL            *string
	processManager process.ServiceManager
	mu             sync.RWMutex
	app            AppInterface
}

// AppInterface defines the interface that Service needs from the App
type AppInterface interface {
	EmitToFrontend(event string, serviceId string, data interface{})
}

// NewService creates a new service
func NewService(id string, config config.ServiceConfig, inheritedEnv config.ServiceEnv, app AppInterface) *Service {
	service := &Service{
		ID:           id,
		Config:       config,
		InheritedEnv: inheritedEnv,
		Status:       process.Stopped,
		Logs:         []process.LogEntry{},
		app:          app,
	}

	mergedEnv := make(process.ServiceEnv)
	for k, v := range inheritedEnv {
		mergedEnv[k] = v
	}
	for k, v := range config.Env {
		if v == "" {
			delete(mergedEnv, k)
		} else {
			mergedEnv[k] = v
		}
	}

	if config.Type == "npm" {
		service.processManager = process.NewNpmService(config.Path, mergedEnv)
	} else {
		// Default to dotnet for backward compatibility
		service.processManager = process.NewDotnetService(config.Path, mergedEnv)
	}

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
			s.app.EmitToFrontend("newLog", s.ID, map[string]interface{}{"log": log})
			if log.Level == process.Err {
				s.app.EmitToFrontend("statusUpdate", s.ID, map[string]interface{}{
					"status": s.Status,
					"url":    s.URL,
				})
			}
		case url := <-urlChan:
			s.mu.Lock()
			s.URL = &url
			s.mu.Unlock()
			s.app.EmitToFrontend("statusUpdate", s.ID, map[string]interface{}{
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
			s.app.EmitToFrontend("statusUpdate", s.ID, map[string]interface{}{
				"status": s.Status,
				"url":    s.URL,
			})
		}
	}
}

// UpdateConfig updates the service configuration
func (s *Service) UpdateConfig(config config.ServiceConfig, inheritedEnv config.ServiceEnv) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Config = config
	s.InheritedEnv = inheritedEnv

	mergedEnv := make(process.ServiceEnv)
	for k, v := range inheritedEnv {
		mergedEnv[k] = v
	}
	for k, v := range config.Env {
		if v == "" {
			delete(mergedEnv, k)
		} else {
			mergedEnv[k] = v
		}
	}
	s.processManager.UpdateConfig(config.Path, mergedEnv)
}

// GetInfo returns service information
func (s *Service) GetInfo() ServiceInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return ServiceInfo{
		Name:         s.Config.Name,
		Path:         s.Config.Path,
		Status:       s.Status,
		URL:          s.URL,
		Logs:         s.Logs,
		Env:          s.Config.Env,
		InheritedEnv: s.InheritedEnv,
		Type:         s.Config.Type,
	}
}

// Start starts the service
func (s *Service) Start() error {
	return s.processManager.Start()
}

// StartWithoutBuild starts the service without building
func (s *Service) StartWithoutBuild() error {
	return s.processManager.StartWithoutBuild()
}

// Stop stops the service
func (s *Service) Stop() error {
	return s.processManager.Stop()
}

// ClearLogs clears the service logs
func (s *Service) ClearLogs() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Logs = []process.LogEntry{}
}

// GenerateID generates a random ID
func GenerateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}
