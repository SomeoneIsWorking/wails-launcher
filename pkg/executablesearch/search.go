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
	homeDir, _ := os.UserHomeDir()
	commonPaths := []string{
		"/usr/local/bin/" + name,
		"/opt/homebrew/bin/" + name,
		"/usr/bin/" + name,
		"/usr/local/share/dotnet/" + name,
	}

	if homeDir != "" {
		commonPaths = append(commonPaths,
			homeDir+"/.nvm/versions/node/v22.19.0/bin/"+name,
			homeDir+"/.nvm/versions/node/v23.5.0/bin/"+name,
			homeDir+"/.local/bin/"+name,
		)

		// Try to find any version in .nvm/versions/node/
		nvmDir := homeDir + "/.nvm/versions/node/"
		if dirs, err := os.ReadDir(nvmDir); err == nil {
			for i := len(dirs) - 1; i >= 0; i-- { // Try newest versions first
				if dirs[i].IsDir() {
					commonPaths = append(commonPaths, nvmDir+dirs[i].Name()+"/bin/"+name)
				}
			}
		}
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", err
}
