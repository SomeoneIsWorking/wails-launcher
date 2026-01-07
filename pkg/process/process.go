package process

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"wails-launcher/pkg/executablesearch"
	"wails-launcher/pkg/processsearch"
)

// DotnetService manages dotnet processes
type DotnetService struct {
	path       string
	env        ServiceEnv
	process    *exec.Cmd
	logChan    chan LogEntry
	urlChan    chan string
	statusChan chan ServiceStatus
}

// NewDotnetService creates a new DotnetService
func NewDotnetService(path string, env ServiceEnv) *DotnetService {
	return &DotnetService{
		path:       path,
		env:        env,
		logChan:    make(chan LogEntry, 100),
		urlChan:    make(chan string, 10),
		statusChan: make(chan ServiceStatus, 10),
	}
}

// UpdateConfig updates the config
func (ds *DotnetService) UpdateConfig(path string, env ServiceEnv) {
	ds.path = path
	ds.env = env
}

// Start starts the service
func (ds *DotnetService) Start() error {
	if ds.process != nil {
		return fmt.Errorf("service already running")
	}
	ds.emitStatus(Starting)
	err := ds.cleanup()
	if err != nil {
		ds.emitStatus(Error)
		return err
	}
	cmd, err := ds.spawn()
	if err != nil {
		ds.emitLog(Err, fmt.Sprintf("Failed to spawn process: %v", err), "")
		ds.emitStatus(Error)
		return err
	}
	ds.process = cmd
	ds.emitStatus(Initializing)
	go ds.monitorProcess()
	return nil
}

// Stop stops the service
func (ds *DotnetService) Stop() error {
	if ds.process == nil {
		ds.emitStatus(Stopped)
		return nil
	}
	ds.emitStatus(Stopping)
	err := ds.process.Process.Kill()
	if err != nil {
		return err
	}
	ds.process.Wait()
	ds.process = nil
	ds.emitStatus(Stopped)
	return nil
}

// GetChannels returns the channels for listening
func (ds *DotnetService) GetChannels() (<-chan LogEntry, <-chan string, <-chan ServiceStatus) {
	return ds.logChan, ds.urlChan, ds.statusChan
}

// emitLog emits a log entry
func (ds *DotnetService) emitLog(level LogLevel, message, raw string) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Message:   message,
		Raw:       raw,
	}
	select {
	case ds.logChan <- entry:
	default:
	}
}

// emitURL emits a URL
func (ds *DotnetService) emitURL(url string) {
	select {
	case ds.urlChan <- url:
	default:
	}
}

// emitStatus emits a status change
func (ds *DotnetService) emitStatus(status ServiceStatus) {
	select {
	case ds.statusChan <- status:
	default:
	}
}

// cleanup kills existing processes
func (ds *DotnetService) cleanup() error {
	runningProcesses, err := ds.findProcess()
	if err != nil {
		ds.emitLog(Err, fmt.Sprintf("Process search error: %v", err), "")
		return err
	}

	for _, proc := range runningProcesses {
		ds.emitLog(Inf, fmt.Sprintf("Killing process %s (%s)", proc.PID, proc.Cmd), "")
		err := ds.killProcess(proc.PID)
		if err != nil {
			ds.emitLog(Err, fmt.Sprintf("Failed to kill process %s: %v", proc.PID, err), "")
		}
	}
	return nil
}

// findProcess finds running processes for this service
func (ds *DotnetService) findProcess() ([]processsearch.ProcessInfo, error) {
	return processsearch.FindProcessesByCWD(ds.path)
}

// killProcess kills a process by PID
func (ds *DotnetService) killProcess(pid string) error {
	// First try graceful kill
	cmd := exec.Command("kill", pid)
	err := cmd.Run()
	if err != nil {
		return err
	}

	// Wait a bit
	time.Sleep(time.Second)

	// Check if still running
	cmd = exec.Command("ps", "-p", pid)
	err = cmd.Run()
	if err != nil {
		// Process is gone
		ds.emitLog(Inf, fmt.Sprintf("Process %s terminated gracefully", pid), "")
		return nil
	}

	// Force kill
	cmd = exec.Command("kill", "-9", pid)
	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

// spawn spawns the dotnet process
func (ds *DotnetService) spawn() (*exec.Cmd, error) {
	dotnetPath, err := executablesearch.FindExecutable("dotnet")
	if err != nil {
		return nil, fmt.Errorf("dotnet not found: %v", err)
	}
	ds.emitLog(Inf, fmt.Sprintf("Using dotnet at: %s", dotnetPath), "")
	cmd := exec.Command(dotnetPath, "run")
	cmd.Dir = ds.path
	cmd.Env = os.Environ()
	for k, v := range ds.env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	go ds.readOutput(stdout)
	go ds.readOutput(stderr)
	return cmd, nil
}

// readOutput reads from pipe
func (ds *DotnetService) readOutput(pipe io.ReadCloser) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()
		ds.processLine(line)
	}
}

// processLine processes a line of output
func (ds *DotnetService) processLine(line string) {
	// Check for URL
	if strings.Contains(line, "Now listening on") {
		// Extract URL
		parts := strings.Split(line, "Now listening on:")
		if len(parts) > 1 {
			url := strings.TrimSpace(parts[1])
			ds.emitURL(url)
		}
	}
	// Parse log
	entry := ds.parseLog(line)
	if entry != nil {
		ds.emitLog(entry.Level, entry.Message, entry.Raw)
	}
}

// parseLog parses a log line
func (ds *DotnetService) parseLog(line string) *LogEntry {
	if strings.Contains(line, "NETSDK1138") {
		return nil
	}
	re := regexp.MustCompile(`^\[(.*?)\s+(ERR|INF|WARN|DBG).*?\]\s+(.*)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) == 4 {
		return &LogEntry{
			Timestamp: matches[1],
			Level:     LogLevel(matches[2]),
			Message:   matches[3],
			Raw:       line,
		}
	}
	return &LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     Inf,
		Message:   line,
		Raw:       line,
	}
}

// monitorProcess monitors the process
func (ds *DotnetService) monitorProcess() {
	if ds.process == nil {
		return
	}
	err := ds.process.Wait()
	ds.process = nil
	status := Stopped
	if err != nil {
		status = Error
	}
	ds.emitStatus(status)
}
