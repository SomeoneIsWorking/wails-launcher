package processsearch

import (
	"os/exec"
	"strings"
)

// ProcessInfo holds information about a process
type ProcessInfo struct {
	PID  string
	Cmd  string
	Path string
}

// FindProcessesByCWD finds processes by current working directory
func FindProcessesByCWD(cwd string) ([]ProcessInfo, error) {
	cmd := exec.Command("lsof", "-c", "*", "-a", "-d", "cwd")
	output, err := cmd.Output()
	if err != nil {
		// If lsof fails with exit code 1 and no output, no processes found
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 && len(output) == 0 {
			return []ProcessInfo{}, nil
		}
		return nil, err
	}

	var results []ProcessInfo
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, "cwd") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 9 {
			continue
		}
		pid := parts[1]
		path := parts[8]
		if strings.Contains(strings.ToLower(path), strings.ToLower(cwd)) {
			results = append(results, ProcessInfo{
				PID:  pid,
				Cmd:  cwd,
				Path: path,
			})
		}
	}
	return results, nil
}

// FindProcessesByName finds processes by command name
func FindProcessesByName(name string) ([]ProcessInfo, error) {
	cmd := exec.Command("ps", "aux")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var results []ProcessInfo
	lines := strings.Split(string(output), "\n")
	for _, line := range lines[1:] { // Skip header
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 11 {
			continue
		}
		pid := parts[1]
		command := strings.Join(parts[10:], " ")
		if strings.Contains(command, name) {
			results = append(results, ProcessInfo{
				PID: pid,
				Cmd: command,
			})
		}
	}
	return results, nil
}
