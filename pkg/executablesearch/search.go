package executablesearch

import (
	"os"
	"os/exec"
)

// FindExecutable finds the path to an executable
func FindExecutable(name string) (string, error) {
	// First try exec.LookPath
	path, err := exec.LookPath(name)
	if err == nil {
		return path, nil
	}
	// Fallback to common paths
	commonPaths := []string{
		"/usr/local/bin/" + name,
		"/opt/homebrew/bin/" + name,
		"/usr/bin/" + name,
		"/usr/local/share/dotnet/" + name,
		"/Users/" + os.Getenv("USER") + "/.nvm/versions/node/v22.19.0/bin/" + name,
	}
	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", err
}
