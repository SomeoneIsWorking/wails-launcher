package process

// LogLevel represents the log level
type LogLevel string

const (
	Err  LogLevel = "ERR"
	Inf  LogLevel = "INF"
	Warn LogLevel = "WARN"
	Dbg  LogLevel = "DBG"
)

// ServiceStatus represents the service status
type ServiceStatus string

const (
	Stopped      ServiceStatus = "stopped"
	Starting     ServiceStatus = "starting"
	Initializing ServiceStatus = "initializing"
	Running      ServiceStatus = "running"
	Stopping     ServiceStatus = "stopping"
	Error        ServiceStatus = "error"
)

// LogEntry represents a log entry
type LogEntry struct {
	Timestamp string   `json:"timestamp"`
	Level     LogLevel `json:"level"`
	Message   string   `json:"message"`
	Raw       string   `json:"raw"`
	Stream    string   `json:"stream"` // "stdout" or "stderr"
}

// ServiceEnv represents environment variables
type ServiceEnv map[string]string

// ProcessInfo represents information about a running process
type ProcessInfo struct {
	pid string
	cmd string
}
