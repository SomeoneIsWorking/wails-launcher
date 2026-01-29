package bridge

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"wails-launcher/pkg/executablesearch"
)

//go:embed bridge.py
var bridgeScript []byte

// CreateCommand returns a command that runs the provided command through the bridge
func CreateCommand(originalCmd []string, env []string, workDir string) (*exec.Cmd, error) {
	// 1. Setup paths
	tmpDir := os.TempDir()
	scriptPath := filepath.Join(tmpDir, "wails-bridge.py")

	// 2. Write bridge script to temp file
	if err := os.WriteFile(scriptPath, bridgeScript, 0755); err != nil {
		return nil, fmt.Errorf("failed to write bridge script: %w", err)
	}

	// 3. Find python executable
	pythonPath, err := executablesearch.FindExecutable("python3")
	if err != nil {
		pythonPath, err = executablesearch.FindExecutable("python")
		if err != nil {
			return nil, fmt.Errorf("python not found: %w", err)
		}
	}

	// 4. Construct arguments
	parentPid := os.Getpid()
	cmdStr := strings.Join(originalCmd, " ")
	args := []string{scriptPath, "--parent-pid", fmt.Sprint(parentPid), "--cmd", cmdStr}

	// 5. Initialize command
	cmd := exec.Command(pythonPath, args...)
	cmd.Dir = workDir
	cmd.Env = env

	return cmd, nil
}
