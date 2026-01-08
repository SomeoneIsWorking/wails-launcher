package process

// ServiceManager defines the interface for managing a service process
type ServiceManager interface {
	Start() error
	Stop() error
	UpdateConfig(path string, env ServiceEnv)
	GetChannels() (<-chan LogEntry, <-chan string, <-chan ServiceStatus)
}
