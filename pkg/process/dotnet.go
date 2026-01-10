package process

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"wails-launcher/pkg/bridge"
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
		ds.emitLog(Err, fmt.Sprintf("Failed to spawn process: %v", err), "", "stdout")
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

	// Send interrupt signal to bridge to allow it to kill children
	ds.process.Process.Signal(os.Interrupt)

	// Give it a moment to cleanup
	time.Sleep(500 * time.Millisecond)

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
func (ds *DotnetService) emitLog(level LogLevel, message, raw string, stream string) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Message:   message,
		Raw:       raw,
		Stream:    stream,
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
		ds.emitLog(Err, fmt.Sprintf("Process search error: %v", err), "", "stdout")
		return err
	}

	for _, proc := range runningProcesses {
		ds.emitLog(Inf, fmt.Sprintf("Killing process %s (%s)", proc.PID, proc.Cmd), "", "stdout")
		err := ds.killProcess(proc.PID)
		if err != nil {
			ds.emitLog(Err, fmt.Sprintf("Failed to kill process %s: %v", proc.PID, err), "", "stdout")
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
		ds.emitLog(Inf, fmt.Sprintf("Process %s terminated gracefully", pid), "", "stdout")
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
	ds.emitLog(Inf, fmt.Sprintf("Using dotnet at: %s", dotnetPath), "", "stdout")

	env := os.Environ()
	for k, v := range ds.env {
		env = append(env, k+"="+v)
	}

	cmd, err := bridge.CreateCommand([]string{dotnetPath, "run"}, env, ds.path)
	if err != nil {
		return nil, fmt.Errorf("failed to create bridge command: %v", err)
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
	go ds.readOutput(stdout, "stdout")
	go ds.readOutput(stderr, "stderr")
	return cmd, nil
}

// readOutput reads from pipe and buffers multi-line log entries
func (ds *DotnetService) readOutput(pipe io.ReadCloser, stream string) {
	scanner := bufio.NewScanner(pipe)
	var buffer strings.Builder

	// Matches log levels (info:, warn:, error:, debug:, trace:, critical:, fail:)
	// or compiler output (paths with warnings/errors)
	// or build/restore output lines
	isLogStart := func(line string) bool {
		if len(line) == 0 {
			return false
		}
		// Skip lines that are just stream indicators
		trimmed := strings.TrimSpace(line)
		if trimmed == "STDERR" || trimmed == "STDOUT" {
			return false
		}
		// Check if line starts with common log prefixes (not indented)
		if line[0] != ' ' && line[0] != '\t' {
			lower := strings.ToLower(line)
			if strings.HasPrefix(lower, "info:") ||
				strings.HasPrefix(lower, "warn:") ||
				strings.HasPrefix(lower, "error:") ||
				strings.HasPrefix(lower, "debug:") ||
				strings.HasPrefix(lower, "trace:") ||
				strings.HasPrefix(lower, "critical:") ||
				strings.HasPrefix(lower, "fail:") {
				return true
			}
			// Compiler warnings/errors (start with path)
			if strings.Contains(line, "): warning ") || strings.Contains(line, "): error ") {
				return true
			}
			// NuGet warnings/errors (path : warning/error format)
			if strings.Contains(line, " : warning ") || strings.Contains(line, " : error ") {
				return true
			}
			// Build/restore messages
			if strings.HasPrefix(line, "Build ") || strings.HasPrefix(line, "Restore ") ||
				strings.HasPrefix(line, "Determining ") || strings.HasPrefix(line, "Building...") {
				return true
			}
		}
		return false
	}

	flushBuffer := func() {
		if buffer.Len() > 0 {
			ds.processLine(buffer.String(), stream)
			buffer.Reset()
		}
	}

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and flush buffer
		if trimmed == "" {
			flushBuffer()
			continue
		}

		// Check if this line starts a new log entry
		if isLogStart(line) {
			// Flush any buffered content first
			flushBuffer()
			// Start new buffer with this line
			buffer.WriteString(line)
		} else if buffer.Len() > 0 {
			// This is a continuation line, append to buffer
			buffer.WriteString("\n")
			buffer.WriteString(line)
		} else {
			// Start new buffer with this line (instead of processing immediately)
			buffer.WriteString(line)
		}
	}

	// Flush any remaining buffered content
	flushBuffer()
}

// processLine processes a single log entry (may be multi-line)
func (ds *DotnetService) processLine(line string, stream string) {
	// Check for URL in the log
	if strings.Contains(line, "Now listening on:") {
		parts := strings.Split(line, "Now listening on:")
		if len(parts) == 2 {
			url := strings.TrimSpace(parts[1])
			ds.emitURL(url)
		}
	}

	// Parse and emit log
	entry := ds.parseLog(line, stream)
	if entry != nil {
		ds.emitLog(entry.Level, entry.Message, entry.Raw, entry.Stream)
	}
}

// parseLog parses a log entry (may be multi-line)
func (ds *DotnetService) parseLog(line string, stream string) *LogEntry {
	if strings.Contains(line, "NETSDK1138") {
		return nil
	}

	// Determine log level from the line
	level := Inf
	lower := strings.ToLower(line)

	if strings.HasPrefix(lower, "error:") || strings.Contains(line, "): error ") || strings.Contains(line, " ERR]") || strings.Contains(line, " ERR ") {
		level = Err
	} else if strings.HasPrefix(lower, "warn:") || strings.Contains(line, "): warning ") || strings.Contains(line, ": warning ") {
		level = Warn
	} else if strings.HasPrefix(lower, "debug:") || strings.HasPrefix(lower, "trace:") {
		level = Dbg
	} else if strings.HasPrefix(lower, "info:") {
		level = Inf
	} else if strings.HasPrefix(lower, "critical:") || strings.HasPrefix(lower, "fail:") {
		level = Err
	} else if stream == "stderr" {
		level = Err
	}

	return &LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Message:   line,
		Raw:       line,
		Stream:    stream,
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
