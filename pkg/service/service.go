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
	Name   string                `json:"name"`
	Path   string                `json:"path"`
	Status process.ServiceStatus `json:"status"`
	URL    *string               `json:"url,omitempty"`
	Logs   []process.LogEntry    `json:"logs"`
	Env    config.ServiceEnv     `json:"env"`
}

// Service represents a service
type Service struct {
	ID             string
	Config         config.ServiceConfig
	Status         process.ServiceStatus
	Logs           []process.LogEntry
	URL            *string
	processManager *process.DotnetService
	mu             sync.RWMutex
	app            AppInterface
}

// AppInterface defines the interface that Service needs from the App
type AppInterface interface {
	EmitToFrontend(event string, serviceId string, data interface{})
}

// NewService creates a new service
func NewService(id string, config config.ServiceConfig, app AppInterface) *Service {
	service := &Service{
		ID:     id,
		Config: config,
		Status: process.Stopped,
		Logs:   []process.LogEntry{},
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
func (s *Service) UpdateConfig(config config.ServiceConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Config = config
	s.processManager.UpdateConfig(config.Path, config.Env)
}

// GetInfo returns service information
func (s *Service) GetInfo() ServiceInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return ServiceInfo{
		Name:   s.Config.Name,
		Path:   s.Config.Path,
		Status: s.Status,
		URL:    s.URL,
		Logs:   s.Logs,
		Env:    s.Config.Env,
	}
}

// Start starts the service
func (s *Service) Start() error {
	return s.processManager.Start()
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
