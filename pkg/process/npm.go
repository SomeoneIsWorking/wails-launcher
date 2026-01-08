package process

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"wails-launcher/pkg/bridge"
	"wails-launcher/pkg/executablesearch"
	"wails-launcher/pkg/processsearch"
)

// NpmService manages npm processes
type NpmService struct {
	path       string
	env        ServiceEnv
	process    *exec.Cmd
	logChan    chan LogEntry
	urlChan    chan string
	statusChan chan ServiceStatus
}

// NewNpmService creates a new NpmService
func NewNpmService(path string, env ServiceEnv) *NpmService {
	return &NpmService{
		path:       path,
		env:        env,
		logChan:    make(chan LogEntry, 100),
		urlChan:    make(chan string, 10),
		statusChan: make(chan ServiceStatus, 10),
	}
}

// UpdateConfig updates the config
func (ns *NpmService) UpdateConfig(path string, env ServiceEnv) {
	ns.path = path
	ns.env = env
}

// Start starts the service
func (ns *NpmService) Start() error {
	if ns.process != nil {
		return fmt.Errorf("service already running")
	}
	ns.emitStatus(Starting)
	err := ns.cleanup()
	if err != nil {
		ns.emitStatus(Error)
		return err
	}
	cmd, err := ns.spawn()
	if err != nil {
		ns.emitLog(Err, fmt.Sprintf("Failed to spawn process: %v", err), "", "stdout")
		ns.emitStatus(Error)
		return err
	}
	ns.process = cmd
	ns.emitStatus(Initializing)
	go ns.monitorProcess()
	return nil
}

// Stop stops the service
func (ns *NpmService) Stop() error {
	if ns.process == nil {
		ns.emitStatus(Stopped)
		return nil
	}
	ns.emitStatus(Stopping)

	// Send interrupt signal to bridge to allow it to kill children
	ns.process.Process.Signal(os.Interrupt)

	// Give it a moment to cleanup
	time.Sleep(500 * time.Millisecond)

	err := ns.process.Process.Kill()
	if err != nil {
		return err
	}
	ns.process.Wait()
	ns.process = nil
	ns.emitStatus(Stopped)
	return nil
}

// GetChannels returns the channels for listening
func (ns *NpmService) GetChannels() (<-chan LogEntry, <-chan string, <-chan ServiceStatus) {
	return ns.logChan, ns.urlChan, ns.statusChan
}

// emitLog emits a log entry
func (ns *NpmService) emitLog(level LogLevel, message, raw string, stream string) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Message:   message,
		Raw:       raw,
		Stream:    stream,
	}
	select {
	case ns.logChan <- entry:
	default:
	}
}

// emitURL emits a URL
func (ns *NpmService) emitURL(url string) {
	select {
	case ns.urlChan <- url:
	default:
	}
}

// emitStatus emits a status change
func (ns *NpmService) emitStatus(status ServiceStatus) {
	select {
	case ns.statusChan <- status:
	default:
	}
}

// cleanup kills existing processes
func (ns *NpmService) cleanup() error {
	runningProcesses, err := ns.findProcess()
	if err != nil {
		ns.emitLog(Err, fmt.Sprintf("Process search error: %v", err), "", "stdout")
		return err
	}

	for _, proc := range runningProcesses {
		ns.emitLog(Inf, fmt.Sprintf("Killing process %s (%s)", proc.PID, proc.Cmd), "", "stdout")
		err := ns.killProcess(proc.PID)
		if err != nil {
			ns.emitLog(Err, fmt.Sprintf("Failed to kill process %s: %v", proc.PID, err), "", "stdout")
		}
	}
	return nil
}

// findProcess finds running processes for this service
func (ns *NpmService) findProcess() ([]processsearch.ProcessInfo, error) {
	return processsearch.FindProcessesByCWD(ns.path)
}

// killProcess kills a process by PID
func (ns *NpmService) killProcess(pid string) error {
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
		ns.emitLog(Inf, fmt.Sprintf("Process %s terminated gracefully", pid), "", "stdout")
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

// spawn spawns the npm process
func (ns *NpmService) spawn() (*exec.Cmd, error) {
	npmPath, err := executablesearch.FindExecutable("npm")
	if err != nil {
		return nil, fmt.Errorf("npm not found: %v", err)
	}
	ns.emitLog(Inf, fmt.Sprintf("Using npm at: %s", npmPath), "", "stdout")

	env := os.Environ()

	// Update PATH to include npm's directory, so it can find node
	npmDir := filepath.Dir(npmPath)
	pathFound := false
	for i, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			env[i] = "PATH=" + npmDir + ":" + strings.TrimPrefix(e, "PATH=")
			pathFound = true
			break
		}
	}
	if !pathFound {
		env = append(env, "PATH="+npmDir)
	}

	for k, v := range ns.env {
		env = append(env, k+"="+v)
	}

	cmd, err := bridge.CreateCommand([]string{npmPath, "run", "dev"}, env, ns.path)
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
	go ns.readOutput(stdout, "stdout")
	go ns.readOutput(stderr, "stderr")
	return cmd, nil
}

// readOutput reads from pipe
func (ns *NpmService) readOutput(pipe io.ReadCloser, stream string) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()
		ns.processLine(line, stream)
	}
}

// processLine processes a line of output
func (ns *NpmService) processLine(line string, stream string) {
	// Check for URL (Vite/Nuxt/etc)
	if (strings.Contains(line, "Local:") || strings.Contains(line, "listening on")) && strings.Contains(line, "http") {
		re := regexp.MustCompile(`http(s)?://\S+`)
		url := re.FindString(line)
		if url != "" {
			ns.emitURL(strings.TrimRight(url, "/"))
		}
	}
	// Parse log
	entry := ns.parseLog(line, stream)
	if entry != nil {
		ns.emitLog(entry.Level, entry.Message, entry.Raw, entry.Stream)
	}
}

// parseLog parses a log line
func (ns *NpmService) parseLog(line string, stream string) *LogEntry {
	level := Inf
	if stream == "stderr" {
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
func (ns *NpmService) monitorProcess() {
	if ns.process == nil {
		return
	}
	err := ns.process.Wait()
	ns.process = nil
	status := Stopped
	if err != nil {
		status = Error
	}
	ns.emitStatus(status)
}
