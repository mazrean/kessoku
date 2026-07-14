package llmsetup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// expandTilde replaces a leading ~ with the user's home directory.
// It handles "~" alone and "~/..." paths. Other paths are returned as-is.
func expandTilde(path string) (string, error) {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w (use an absolute path instead of ~)", err)
	}

	if path == "~" {
		return home, nil
	}

	return filepath.Join(home, path[2:]), nil
}

// ResolvePath resolves the installation path based on flags and agent configuration.
// Priority: customPath > userFlag > project-level (default)
func ResolvePath(customPath string, userFlag bool, agent Agent) (string, error) {
	if customPath != "" {
		expanded, err := expandTilde(customPath)
		if err != nil {
			return "", err
		}
		return filepath.Abs(expanded)
	}

	if userFlag {
		return resolveUserPath(agent)
	}

	return resolveProjectPath(agent)
}

// resolveUserPath resolves the user-level installation path.
func resolveUserPath(agent Agent) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w (use --path flag)", err)
	}

	return filepath.Join(home, agent.UserSubPath()), nil
}

// resolveProjectPath resolves the project-level installation path.
func resolveProjectPath(agent Agent) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("cannot determine current directory: %w", err)
	}

	return filepath.Join(cwd, agent.ProjectSubPath()), nil
}

// ValidatePath validates that the path is usable for installation.
// Returns nil if path doesn't exist (will be created) or is a directory.
// Returns error if path exists and is a file.
func ValidatePath(path string) error {
	info, err := os.Stat(path)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("path is a file, not a directory: %s", path)
		}
		return nil
	}

	if os.IsNotExist(err) {
		// Path doesn't exist, that's OK - will be created
		return nil
	}

	return fmt.Errorf("cannot access path: %s: %w", path, err)
}
